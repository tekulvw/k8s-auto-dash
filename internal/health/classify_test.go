package health

import (
	"errors"
	"testing"

	"github.com/tekulvw/k8s-auto-dash/internal/tile"
	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		code int
		err  error
		want tile.StatusState
	}{
		{"200 up", 200, nil, tile.StateUp},
		{"301 up", 301, nil, tile.StateUp},
		{"399 up", 399, nil, tile.StateUp},
		{"400 degraded", 400, nil, tile.StateDegraded},
		{"500 degraded", 500, nil, tile.StateDegraded},
		{"network err down", 0, errors.New("dial: refused"), tile.StateDown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := Classify(c.code, c.err)
			assert.Equal(t, c.want, s)
		})
	}
}
