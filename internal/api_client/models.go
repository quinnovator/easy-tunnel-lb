package api_client

// TunnelRequest represents a request to create or update a tunnel
type TunnelRequest struct {
	IngressName      string            `json:"ingressName"`
	IngressNamespace string            `json:"ingressNamespace"`
	Hostname         string            `json:"hostname"`
	Ports           []int             `json:"ports"`
	Annotations     map[string]string `json:"annotations"`
}

// TunnelResponse represents the response from the server for a tunnel request
type TunnelResponse struct {
	TunnelID     string `json:"tunnelId"`
	ExternalIP   string `json:"externalIp,omitempty"`
	ExternalHost string `json:"externalHost,omitempty"`
	Status       string `json:"status"`
	WGConfig     string `json:"wgConfig,omitempty"`
}

// TunnelStatus represents the current status of a tunnel
type TunnelStatus struct {
	TunnelID string `json:"tunnelId"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

// Error types
const (
	StatusActive    = "active"
	StatusPending   = "pending"
	StatusError     = "error"
	StatusNotFound  = "not_found"
) 