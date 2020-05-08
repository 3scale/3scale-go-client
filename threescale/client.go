package threescale

import (
	"github.com/3scale/3scale-go-client/threescale/api"
)

// Client specifies the behaviour expected for a 3scale backend client
type Client interface {
	// Authorize is a read-only operation to authorize an application with the authentication
	// provided in the transaction params
	// Where multiple transactions are provided, all but the first should be discarded
	Authorize(request Request) (*AuthorizeResult, error)
	// AuthRep should be used to authorize and report, in a single transaction for an application with
	// the authentication provided in the transaction params
	// Where multiple transactions are provided, all but the first should be discarded
	AuthRep(request Request) (*AuthorizeResult, error)
	// Report the transactions to 3scale backend with the authentication provided in the transactions params
	Report(request Request) (*ReportResult, error)
	// GetPeer returns the hostname of the connected backend
	GetPeer() string
}

// AuthorizeExtensions may be returned by a client when the caller leverages the extensions
// provided by backend. Not all clients will support returning extensions.
type AuthorizeExtensions struct {
	// List of children (methods) associated with a parent(metric)
	Hierarchy api.Hierarchy
	// Result from rate limiting extension 'limit_headers' - will be nil if not leveraged or unsupported
	RateLimits *api.RateLimits
}

// AuthorizeResult is returned by a client for Auth and AuthRep requests
type AuthorizeResult struct {
	// Authorized states if the call has been authorized by 3scale
	Authorized bool
	// List of usage reports - list will be empty if no limits set
	UsageReports api.UsageReports
	// ErrorCode as returned by backend - see https://github.com/3scale/apisonator/blob/v2.96.2/docs/rfcs/error_responses.md
	ErrorCode string
	// RejectionReason - human readable string explaining why authorization has not been granted
	RejectionReason string
	// RawResponse may be set by the underlying client implementation
	RawResponse interface{}
	AuthorizeExtensions
}

// ReportResult should be returned by client implementations for report requests
type ReportResult struct {
	// Accepted notifies us that the report request was accepted for processing in 3scale backend
	Accepted bool
	// ErrorCode as returned by backend - see https://github.com/3scale/apisonator/blob/v2.96.2/docs/rfcs/error_responses.md
	ErrorCode string
	// RawResponse may be set by the underlying client implementation
	RawResponse interface{}
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
