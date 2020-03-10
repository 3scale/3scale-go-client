package api

const (
	// ClientAuth authentication types

	// ServiceToken as a specific key for the service
	ServiceToken AuthType = "service_token"
	// ProviderKey for all services under an account
	ProviderKey AuthType = "provider_key"
)

const (
	// Rate limiting extension keys - see https://github.com/3scale/apisonator/blob/v2.96.2/docs/rfcs/api-extensions.md
	// and https://github.com/3scale/apisonator/blob/v2.96.2/docs/extensions.md#limit_headers-boolean

	// LimitExtension is the key to enable this extension when calling 3scale backend - set to 1 to enable
	LimitExtension = "limit_headers"

	// HierarchyExtension is the key to enabling hierarchy feature. Set its bool value to 1 to enable.
	// https://github.com/3scale/apisonator/issues/75
	HierarchyExtension = "hierarchy"

	// FlatUsageExtension is the key to enabling the "flat usage" feature for reporting purposes - set to 1 to enable
	// Enabling this feature implies that the backend will not calculate the relationships between hierarchies and
	// pushes this compute responsibility back to the client.
	// Therefore when enabled, it is the clients responsibility to ensure that parent --> child metrics
	// are calculated correctly. This feature is supported in versions >= 2.8
	// Use the GetVersion() function to ensure suitability or risk incurring unreported state.
	FlatUsageExtension = "flat_usage"
)

// Period wraps the known rate limiting periods as defined in 3scale
type Period int

// Predefined, known LimitPeriods which can be used in 3scale rate limiting functionality
// These values represent time durations.
const (
	Minute Period = iota
	Hour
	Day
	Week
	Month
	Year
	Eternity
)

// AuthType maps to a known client authentication pattern
// Currently known and supported are 0=ServiceToken 1=ProviderKey
type AuthType string

// ClientAuth holds the key type (ProviderKey, ServiceToken) and their respective value for
// authenticating the client against a given service.
type ClientAuth struct {
	Type  AuthType
	Value string
}

// Extensions are features or behaviours that are not part of the standard API for a variety of reasons
// See https://github.com/3scale/apisonator/blob/v2.96.2/docs/extensions.md for context
type Extensions map[string]string

// Hierarchy maps a parent metric to its child metrics
type Hierarchy map[string][]string

// Metrics let you track the usage of your API in 3scale
type Metrics map[string]int

// Params that are embedded in each Transaction to 3scale API
// This structure simplifies the formatting of the transaction from the callers perspective
// It is used to authenticate the application
type Params struct {

	// AppID is used in the Application Identifier and Key pairs authentication method.
	// It is mutually exclusive with the API Key authentication method outlined below
	// therefore if both are provided, the value defined in 'UserKey' will be prioritised.
	AppID string `json:"app_id"`

	// AppKey is an optional, secret key which can be used in conjunction with 'AppID'
	AppKey string `json:"app_key"`

	// Referrer is an optional value which is required only if referrer filtering is enabled.
	// If special value '*' (wildcard) is passed, the referrer check is bypassed.
	Referrer string `json:"referrer"`

	// UserID is an optional value for identifying an end user.
	// Required only when the application is rate limiting end users.
	UserID string `json:"user_id"`

	// UserKey is the identifier and shared secret of the application if the authentication pattern is API Key.
	// Mutually exclusive with, and prioritised over 'AppID'.
	UserKey string `json:"user_key"`
}

// PeriodWindow holds information about the start and end time of the specified period
// Start and End are unix timestamp
type PeriodWindow struct {
	Period Period
	Start  int64
	End    int64
}

// RateLimits holds the values returned when using rate limiting extension
type RateLimits struct {
	LimitRemaining int
	LimitReset     int
}

// Service represents a 3scale service marked by its identifier (service_id)
type Service string

// Transaction holds the params and optional additions that will be sent
// to 3scale as query parameters or headers.
type Transaction struct {
	Metrics Metrics
	Params  Params
	// Timestamp is a unix timestamp.
	// Timestamp will only be taken into account when calling the Report API
	Timestamp int64
}

// UsageReport for rate limiting information gathered from using extensions
type UsageReport struct {
	PeriodWindow PeriodWindow
	MaxValue     int
	CurrentValue int
}

// UsageReports defines a map of metric names to a list of 'UsageReport'
type UsageReports map[string][]UsageReport
