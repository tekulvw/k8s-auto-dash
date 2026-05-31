// Package tile contains pure data types and derivation logic for
// dashboard tiles. It must not import k8s client libraries.
package tile

import "time"

type Source string

const (
	SourceHTTPRoute Source = "httproute"
	SourceBookmark  Source = "bookmark"
)

type StatusState string

const (
	StateUp       StatusState = "up"
	StateDegraded StatusState = "degraded"
	StateDown     StatusState = "down"
	StateUnknown  StatusState = "unknown"
)

type Status struct {
	State      StatusState `json:"state"`
	StatusCode int         `json:"statusCode"`
	LatencyMs  int64       `json:"latencyMs"`
	CheckedAt  time.Time   `json:"checkedAt"`
	Error      string      `json:"error,omitempty"`
}

type GatewayRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type K8sInfo struct {
	Namespace     string       `json:"namespace"`
	HTTPRouteName string       `json:"httpRouteName"`
	GatewayRefs   []GatewayRef `json:"gatewayRefs"`
}

type Tile struct {
	ID                 string   `json:"id"`
	Source             Source   `json:"source"`
	Name               string   `json:"name"`
	URL                string   `json:"url"`
	Icon               string   `json:"icon"`
	Description        string   `json:"description,omitempty"`
	Group              string   `json:"group"`
	Order              int      `json:"order"`
	Hidden             bool     `json:"hidden"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify,omitempty"`
	Status             Status   `json:"status"`
	K8s                *K8sInfo `json:"k8s,omitempty"`
	// HealthCheckURL is the internal URL used for health probing. When set it
	// takes precedence over URL so that cluster-internal addresses are used
	// instead of externally-routed hostnames (e.g. Tailscale/headscale names).
	// It is intentionally excluded from the JSON API response.
	HealthCheckURL string `json:"-"`
}

type Group struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// ComputeID returns the stable tile id used as the override key.
// Format: "<namespace>/<httproute-name>/<hostname>".
func ComputeID(namespace, route, hostname string) string {
	return namespace + "/" + route + "/" + hostname
}
