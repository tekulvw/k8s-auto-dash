package health

import "github.com/tekulvw/k8s-auto-dash/internal/tile"

// Classify maps an HTTP response (or network error) to a tile state.
//
//	error != nil       -> down
//	200..399           -> up
//	otherwise (4xx/5xx)-> degraded
func Classify(statusCode int, err error) tile.StatusState {
	if err != nil {
		return tile.StateDown
	}
	if statusCode >= 200 && statusCode < 400 {
		return tile.StateUp
	}
	return tile.StateDegraded
}
