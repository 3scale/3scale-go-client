package threescale

import "github.com/3scale/3scale-go-client/threescale/api"

// GetServiceID from Request
func (r Request) GetServiceID() api.Service {
	return r.Service
}
