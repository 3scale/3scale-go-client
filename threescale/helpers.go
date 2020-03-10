package threescale

import (
	"time"

	"github.com/3scale/3scale-go-client/threescale/api"
)

const timeLayout = "2006-01-02 15:04:05 -0700"

// GetServiceID from Request
func (r Request) GetServiceID() api.Service {
	return r.Service
}

// FormatTimestamp from unix time to string formatting as understood by 3scale
func FormatTimestamp(timestamp int64) string {
	return time.Unix(timestamp, 0).Format(timeLayout)
}
