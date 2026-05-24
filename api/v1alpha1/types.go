package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
type DashboardConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DashboardConfigSpec   `json:"spec,omitempty"`
	Status            DashboardConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DashboardConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DashboardConfig `json:"items"`
}

type DashboardConfigSpec struct {
	Settings  Settings       `json:"settings,omitempty"`
	Groups    []GroupSpec    `json:"groups,omitempty"`
	Tiles     []TileOverride `json:"tiles,omitempty"`
	Bookmarks []Bookmark     `json:"bookmarks,omitempty"`
}

type Settings struct {
	Title       string            `json:"title,omitempty"`
	Theme       string            `json:"theme,omitempty"` // dark|light|auto
	HealthCheck HealthCheckConfig `json:"healthCheck,omitempty"`
	Discovery   DiscoveryConfig   `json:"discovery,omitempty"`
}

type HealthCheckConfig struct {
	Enabled            *bool `json:"enabled,omitempty"`
	IntervalSeconds    int   `json:"intervalSeconds,omitempty"`
	TimeoutSeconds     int   `json:"timeoutSeconds,omitempty"`
	InsecureSkipVerify bool  `json:"insecureSkipVerify,omitempty"`
}

type DiscoveryConfig struct {
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	GatewayClassNames []string              `json:"gatewayClassNames,omitempty"`
}

type GroupSpec struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type TileOverride struct {
	ID                 string `json:"id"`
	Hidden             bool   `json:"hidden,omitempty"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	Icon               string `json:"icon,omitempty"`
	Group              string `json:"group,omitempty"`
	Order              int    `json:"order,omitempty"`
	URL                string `json:"url,omitempty"`
	InsecureSkipVerify *bool  `json:"insecureSkipVerify,omitempty"`
}

type Bookmark struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Icon  string `json:"icon,omitempty"`
	Group string `json:"group,omitempty"`
	Order int    `json:"order,omitempty"`
}

type DashboardConfigStatus struct {
	DiscoveredTiles int                `json:"discoveredTiles,omitempty"`
	LastReconciled  *metav1.Time       `json:"lastReconciled,omitempty"`
	Conditions      []metav1.Condition `json:"conditions,omitempty"`
}
