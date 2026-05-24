package health

import (
	"math/rand"
	"sync"
	"time"

	"github.com/anomalyco/k8s-auto-dash/internal/tile"
)

type Target struct {
	ID                 string
	URL                string
	InsecureSkipVerify bool
}

type PoolOptions struct {
	Workers  int
	Interval time.Duration
	Prober   *Prober
	OnResult func(id string, s tile.Status)
}

type Pool struct {
	opts    PoolOptions
	mu      sync.Mutex
	targets []Target
}

func NewPool(o PoolOptions) *Pool {
	if o.Workers <= 0 {
		o.Workers = 5
	}
	if o.Interval <= 0 {
		o.Interval = 60 * time.Second
	}
	return &Pool{opts: o}
}

// Set replaces the target list. Safe to call from any goroutine.
func (p *Pool) Set(targets []Target) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets = append([]Target(nil), targets...)
}

func (p *Pool) snapshot() []Target {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Target(nil), p.targets...)
}

// Run blocks until stop is closed. It enqueues every target at each
// tick, fanning out across Workers goroutines.
func (p *Pool) Run(stop <-chan struct{}) {
	jobs := make(chan Target)
	var wg sync.WaitGroup
	for i := 0; i < p.opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				s := p.opts.Prober.Probe(t.URL, t.InsecureSkipVerify)
				if p.opts.OnResult != nil {
					p.opts.OnResult(t.ID, s)
				}
			}
		}()
	}

	tick := func() {
		for _, t := range p.snapshot() {
			select {
			case <-stop:
				return
			case jobs <- t:
			}
		}
	}

	// First tick immediately, then on interval with ±10% jitter.
	tick()
	timer := time.NewTimer(jitter(p.opts.Interval))
	defer timer.Stop()
	for {
		select {
		case <-stop:
			close(jobs)
			wg.Wait()
			return
		case <-timer.C:
			tick()
			timer.Reset(jitter(p.opts.Interval))
		}
	}
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	delta := float64(d) * 0.1
	off := (rand.Float64()*2 - 1) * delta // #nosec G404 - jitter, not crypto
	return d + time.Duration(off)
}
