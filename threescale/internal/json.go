package internal

// StatusResponse from the "/status" endpoint
type StatusResponse struct {
	Status  string `json:"status"`
	Version struct {
		Backend string `json:"backend"`
	} `json:"version"`
}
