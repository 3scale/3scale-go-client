package client

import (
	"encoding/xml"
	"net/http"
	"net/url"
)

const serviceToken = "service_token"
const providerKey = "provider_key"

const (
	Second   LimitPeriod = "second"
	Minute   LimitPeriod = "minute"
	Hour     LimitPeriod = "hour"
	Day      LimitPeriod = "day"
	Week     LimitPeriod = "week"
	Month    LimitPeriod = "month"
	Eternity LimitPeriod = "eternity"
)

// ApiResponse - formatted response to client
type ApiResponse struct {
	Reason     string
	Success    bool
	StatusCode int
	// nil value indicates 'limit_headers' extension not in use or parsing error with 3scale response.
	RateLimits   *RateLimits
	hierarchy    map[string][]string
	usageReports UsageReports
}

// ApiResponseXML - response from backend API
type ApiResponseXML struct {
	Name         xml.Name  `xml:",any"`
	Authorized   bool      `xml:"authorized,omitempty"`
	Reason       string    `xml:"reason,omitempty"`
	Code         string    `xml:"code,attr,omitempty"`
	Hierarchy    Hierarchy `xml:"hierarchy"`
	UsageReports struct {
		Reports []UsageReportXML `xml:"usage_report"`
	} `xml:"usage_reports"`
}

// AuthorizeParams - optional parameters for the Authorize API - App ID pattern
type AuthorizeParams struct {
	AppKey   string `query:"app_key"`
	Referrer string `query:"referrer"`
	UserId   string `query:"user_id"`
	Metrics  Metrics
}

// AuthorizeParams - optional parameters for the Authorize API - App key pattern
type AuthorizeKeyParams struct {
	Referrer string `query:"referrer"`
	UserId   string `query:"user_id"`
	Metrics  Metrics
}

// AuthRepParams - optional params for AuthRep API - App ID pattern
type AuthRepParams struct {
	AuthorizeParams
	Log Log
}

// Backend defines a 3scale backend service
type Backend struct {
	scheme  string
	host    string
	port    int
	baseUrl *url.URL
}

// Valid rate limiting period as defined in 3scale
type LimitPeriod string

// Log to be reported
type Log map[string]string

// Metrics to be reported
type Metrics map[string]int

// ThreeScaleClient interacts with 3scale Service Management API
type ThreeScaleClient struct {
	backend    *Backend
	httpClient *http.Client
}

type ReportTransactions struct {
	AppID     string `query:"app_id"`
	UserKey   string `query:"user_key"`
	UserId    string `query:"user_id"`
	Timestamp string `query:"timestamp"`
	Metrics   Metrics
	Log       Log
}

type TokenAuth struct {
	Type  string
	Value string
}

// Hierarchy encapsulates the return value when using "hierarchy" extension
type Hierarchy struct {
	Metric []struct {
		Name     string `xml:"name,attr"`
		Children string `xml:"children,attr"`
	} `xml:"metric"`
}

type UsageReports map[string]UsageReport

// UsageReport - captures the XML response for rate limiting details
type UsageReport struct {
	Period       LimitPeriod
	PeriodStart  int64
	PeriodEnd    int64
	MaxValue     int
	CurrentValue int
}

// UsageReportXML - captures the XML response for rate limiting details
type UsageReportXML struct {
	Metric       string      `xml:"metric,attr"`
	Period       LimitPeriod `xml:"period,attr"`
	PeriodStart  string      `xml:"period_start"`
	PeriodEnd    string      `xml:"period_end"`
	MaxValue     int         `xml:"max_value"`
	CurrentValue int         `xml:"current_value"`
}

// RateLimits encapsulates the return values when using the "limit_headers" extension
type RateLimits struct {
	limitRemaining int
	limitReset     int
}

type AppID struct {
	ID string
	//Optional AppKey
	AppKey string
}

type Application struct {
	AppID   AppID
	UserKey string
}

type Request struct {
	Application Application
	Credentials TokenAuth
}
