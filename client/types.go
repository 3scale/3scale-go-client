package client

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
)

const serviceToken = "service_token"
const providerKey = "provider_key"

// ApiResponse - formatted response to client
type ApiResponse struct {
	Reason     string
	Success    bool
	StatusCode int
	// nil value indicates 'limit_headers' extension not in use or parsing error with 3scale response.
	RateLimits *RateLimits
	hierarchy  map[string][]string
}

// ApiResponseXML - response from backend API
type ApiResponseXML struct {
	Name       xml.Name  `xml:",any"`
	Authorized bool      `xml:"authorized,omitempty"`
	Reason     string    `xml:"reason,omitempty"`
	Code       string    `xml:"code,attr,omitempty"`
	Hierarchy  Hierarchy `xml:"hierarchy"`
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

// RateLimits encapsulates the return values when using the "limit_headers" extension
type RateLimits struct {
	limitRemaining int
	limitReset     int
}

func (auth *TokenAuth) SetURLValues(values *url.Values) error {

	switch auth.Type {
	case serviceToken:
		values.Add("service_token", auth.Value)
		return nil

	case providerKey:
		values.Add("provider_key", auth.Value)
		return nil

	default:
		return errors.New("invalid token type value")
	}

}
