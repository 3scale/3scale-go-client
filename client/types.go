package client

import (
	"encoding/xml"
	"net/http"
	"net/url"
)

// ApiResponse - formatted response to client
type ApiResponse struct {
	Reason     string
	Success    bool
	StatusCode int
}

// ApiResponseXML - response from backend API
type ApiResponseXML struct {
	Name       xml.Name `xml:",any"`
	Authorized bool     `xml:"authorized,omitempty"`
	Reason     string   `xml:"reason,omitempty"`
	Code       string   `xml:"code,attr,omitempty"`
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
