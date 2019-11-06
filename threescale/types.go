package threescale

import (
	"context"
	"encoding/xml"
	"net/http"
)

const (
	// ClientAuth authentication types

	// ServiceToken as a specific key for the service
	ServiceToken AuthType = iota
	// ProviderKey for all services under an account
	ProviderKey
)

const (
	// Rate limiting extension keys - see https://github.com/3scale/apisonator/blob/v2.96.2/docs/rfcs/api-extensions.md
	// and https://github.com/3scale/apisonator/blob/v2.96.2/docs/extensions.md#limit_headers-boolean

	// LimitExtension is the key to enable this extension when calling 3scale backend - set to 1 to enable
	LimitExtension = "limit_headers"

	// https://github.com/3scale/apisonator/issues/75
	// HierarchyExtension is the key to enabling hierarchy feature. Set its bool value to 1 to enable.
	HierarchyExtension = "hierarchy"
)

const (

	// Predefined, known LimitPeriods

	Minute   LimitPeriod = "minute"
	Hour     LimitPeriod = "hour"
	Day      LimitPeriod = "day"
	Week     LimitPeriod = "week"
	Month    LimitPeriod = "month"
	Eternity LimitPeriod = "eternity"
)

// Backend is the interface for the 3scale backend Service Management API
type Backend interface {
	Authorize(serviceID string, auth ClientAuth, request *Request) (*AuthorizeResponse, error)
}

// AuthorizeResponse from 3scale backend when calling the Authorize and AuthRep endpoints
type AuthorizeResponse struct {
	// Reason provides the reason for rejection in case the report failed - expect "" on 2xx StatusCode
	Reason     string
	Success    bool
	StatusCode int
	// nil value indicates 'limit_headers' extension not in use or parsing error with 3scale response.
	RateLimits   *RateLimits
	hierarchy    Hierarchy
	usageReports UsageReports
}

// AuthType maps to a known client authentication pattern
// Currently known and supported are 0=ServiceToken 1=ProviderKey
type AuthType int

// Client interacts with 3scale Service Management API
type Client struct {
	backendHost string
	baseURL     string
	httpClient  *http.Client
}

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

// LimitPeriod wraps the known rate limiting periods as defined in 3scale
type LimitPeriod string

// Metrics let you track the usage of your API in 3scale
type Metrics map[string]int

// Option defines a callback function which is used to provide functional options to the construction of a Request object
type Option func(*Request)

// Params that are embedded in each Request to 3scale API
// This structure simplifies the formatting of the request from the callers perspective
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

// RateLimits holds the values returned when using rate limiting extension
type RateLimits struct {
	limitRemaining int
	limitReset     int
}

// Request holds the params and optional additions that will be sent
// to 3scale as query parameters or headers.
type Request struct {
	Metrics    Metrics
	Params     Params
	Timestamp  string
	context    context.Context
	extensions Extensions
}

type ReportResponse struct {
	Accepted bool
	// Reason provides the reason for rejection in case the report failed - expect "" on 2xx StatusCode
	Reason     string
	StatusCode int
}

// UsageReport for rate limiting information gathered from using extensions
type UsageReport struct {
	Period       LimitPeriod
	PeriodStart  int64
	PeriodEnd    int64
	MaxValue     int
	CurrentValue int
}

// UsageReports defines a map of metric names to 'UsageReport'
type UsageReports map[string]UsageReport

// ***** XML return types from 3scale API

// ApiAuthResponseXML formatted response from backend API for Authorize and AuthRep
type ApiAuthResponseXML struct {
	Name         xml.Name     `xml:",any"`
	Authorized   bool         `xml:"authorized,omitempty"`
	Reason       string       `xml:"reason,omitempty"`
	Code         string       `xml:"code,attr,omitempty"`
	Hierarchy    HierarchyXML `xml:"hierarchy"`
	UsageReports struct {
		Reports []UsageReportXML `xml:"usage_report"`
	} `xml:"usage_reports"`
}

// Hierarchy encapsulates the return value when using "hierarchy" extension
type HierarchyXML struct {
	Metric []struct {
		Name     string `xml:"name,attr"`
		Children string `xml:"children,attr"`
	} `xml:"metric"`
}

// UsageReportXML captures the XML response for rate limiting details
type UsageReportXML struct {
	Metric       string      `xml:"metric,attr"`
	Period       LimitPeriod `xml:"period,attr"`
	PeriodStart  string      `xml:"period_start"`
	PeriodEnd    string      `xml:"period_end"`
	MaxValue     int         `xml:"max_value"`
	CurrentValue int         `xml:"current_value"`
}

// *****
