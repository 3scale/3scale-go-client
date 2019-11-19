package threescale

import "github.com/3scale/3scale-go-client/threescale/api"

// Client specifies the behaviour expected for a 3scale backend client
type Client interface {
	// Authorize is a read-only operation to authorize an application with the authentication
	// provided in the transaction params
	// Where multiple transactions are provided, all but the first should be discarded
	Authorize(request Request) (AuthorizeResult, error)
	// AuthRep should be used to authorize and report, in a single transaction for an application with
	// the authentication provided in the transaction params
	// Where multiple transactions are provided, all but the first should be discarded
	AuthRep(request Request) (AuthorizeResult, error)
	// Report the transactions to 3scale backend with the authentication provided in the transactions params
	Report(request Request) (ReportResult, error)
	// GetPeer returns the hostname of the connected backend
	GetPeer() string
}

// AuthorizeResult should be returned by client implementations for auth and authrep requests
type AuthorizeResult interface {
	// GetHierarchy returns a list of children (methods) associated with a parent(metric)
	GetHierarchy() api.Hierarchy
	// GetRateLimits should return nil if the rate limiting extension has not been leveraged.
	// Valid rate limits should contain a LimitRemaining, an integer stating the amount of hits left for the full
	// combination of metrics authorized in this call before the rate limiting
	// logic would start denying authorizations for the current period.
	// A value of -1 indicates there is no limit in the amount of hits.
	// LimitReset returns ann integer stating the amount of seconds left for the current limiting period to elapse.
	// A value of -1 indicates there i is no limit in time.
	GetRateLimits() *api.RateLimits
	// GetUsageReports returns a list of usage reports - list will be empty if no limits set
	GetUsageReports() api.UsageReports
	// Success determines whether the call was authorized by 3scale backend or not
	Success() bool
}

// ReportResult should be returned by client implementations for report requests
type ReportResult interface {
	// Accepted notifies us that the report request was accepted for processing in 3scale backend
	Accepted() bool
}

// Request encapsulates the requirements for a successful api call to 3scale backend
type Request struct {
	Auth       api.ClientAuth
	Extensions api.Extensions
	Service    api.Service
	// Transactions must be non nil and non empty
	// For Authorize and AuthRep calls, a single transaction (index 0 will) be accepted, others will be discarded
	Transactions []api.Transaction
}
