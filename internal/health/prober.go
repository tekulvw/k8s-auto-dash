package health

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"

	"github.com/anomalyco/k8s-auto-dash/internal/tile"
)

type ProbeOptions struct {
	Timeout            time.Duration
	UserAgent          string
	InsecureSkipVerify bool
}

type Prober struct {
	opts        ProbeOptions
	clientStd   *http.Client
	clientInsec *http.Client
}

func NewProber(opts ProbeOptions) *Prober {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "k8s-auto-dash (health-check)"
	}
	mk := func(skip bool) *http.Client {
		return &http.Client{
			Timeout: opts.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skip},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
	}
	return &Prober{
		opts:        opts,
		clientStd:   mk(opts.InsecureSkipVerify),
		clientInsec: mk(true),
	}
}

// Probe issues HEAD then optionally retries with GET on 405/501.
// Per-call insecure flag overrides the per-prober default.
func (p *Prober) Probe(url string, insecure bool) tile.Status {
	client := p.clientStd
	if insecure || p.opts.InsecureSkipVerify {
		client = p.clientInsec
	}
	start := time.Now()
	code, err := p.do(client, http.MethodHead, url)
	if err == nil && (code == 405 || code == 501) {
		code, err = p.do(client, http.MethodGet, url)
	}
	state := Classify(code, err)
	status := tile.Status{
		State:      state,
		StatusCode: code,
		LatencyMs:  time.Since(start).Milliseconds(),
		CheckedAt:  time.Now().UTC(),
	}
	if err != nil {
		status.Error = err.Error()
	}
	return status
}

func (p *Prober) do(c *http.Client, method, url string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.opts.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", p.opts.UserAgent)
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if method == http.MethodGet {
		_, _ = io.Copy(io.Discard, resp.Body)
	}
	return resp.StatusCode, nil
}
