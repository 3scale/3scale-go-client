package http

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/3scale/3scale-go-client/threescale"

	"github.com/3scale/3scale-go-client/threescale/api"
	"github.com/3scale/3scale-go-client/threescale/internal"
)

const (
	authzEndpoint   = "/transactions/authorize.xml"
	authRepEndpoint = "/transactions/authrep.xml"
	reportEndpoint  = "/transactions.xml"
)

const (
	defaultBackendUrl = "https://su1.3scale.net:443"
	defaultTimeout    = 10 * time.Second

	serviceIDKey = "service_id"

	enableExtensions = "3scale-options"
	// limitRemainingHeaderKey has a value set to the remaining calls in a current period
	limitRemainingHeaderKey = "3scale-limit-remaining"
	// limitResetHeaderKey has a value set to an integer stating the amount of seconds left for the current limiting period to elapse
	limitResetHeaderKey = "3scale-limit-reset"

	httpReqErrText = "error building http transaction"

	// a parsable time format used to convert Ruby time to time type
	timeLayout = "2006-01-02 15:04:05 -0700"
)

var (
	errHttpReq = errors.New(httpReqErrText)
)

// Client interacts with 3scale Service Management API and implements a threescale client
type Client struct {
	backendHost string
	baseURL     string
	httpClient  *http.Client
}

// AuthorizeResponse from 3scale backend when calling the Authorize and AuthRep endpoints
type AuthorizeResponse struct {
	// Reason provides the reason for rejection in case the report failed - expect "" on 2xx StatusCode
	Reason     string
	StatusCode int
	hierarchy  api.Hierarchy
	// nil value indicates 'limit_headers' extension not in use or parsing error with 3scale response.
	rateLimits   *api.RateLimits
	success      bool
	usageReports api.UsageReports
}

// ReportResponse is the object returned when a successful call to the Report API is made
type ReportResponse struct {
	accepted bool
	// Reason provides the reason for rejection in case the report failed - expect "" on 2xx StatusCode
	Reason     string
	StatusCode int
}

// NewClient returns a pointer to a Client providing some verification and sanity checking
// of the backendURL input. backendURL should take one of the following formats:
//	* http://example.com - provided scheme with no port
//	* https://example.com:443 - provided scheme and defined port
func NewClient(backendURL string, httpClient *http.Client) (*Client, error) {
	url, err := verifyBackendUrl(backendURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		backendHost: url.Hostname(),
		baseURL:     backendURL,
		httpClient:  httpClient,
	}, nil
}

// NewDefaultClient returns a pointer to Client which is configured for 3scale SaaS platform.
func NewDefaultClient() (*Client, error) {
	return NewClient(defaultBackendUrl, defaultHttpClient())
}

// Authorize is a read-only operation to authorize an application with the authentication provided in the transaction params
func (c *Client) Authorize(apiCall threescale.Request) (threescale.AuthorizeResult, error) {
	req, err := requestBuilder{}.build(apiCall, c.baseURL, auth)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return c.executeAuthCall(req, apiCall.Extensions)
}

// AuthRep should be used to authorize and report, in a single transaction
// for an application with the authentication provided in the transaction params
func (c *Client) AuthRep(apiCall threescale.Request) (threescale.AuthorizeResult, error) {
	req, err := requestBuilder{}.build(apiCall, c.baseURL, authRep)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return c.executeAuthCall(req, apiCall.Extensions)
}

func (c *Client) Report(apiCall threescale.Request) (threescale.ReportResult, error) {
	req, err := requestBuilder{}.build(apiCall, c.baseURL, report)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return c.executeReportCall(req, apiCall.Extensions)
}

func (c *Client) GetPeer() string {
	return c.backendHost
}

func (c *Client) executeAuthCall(req *http.Request, extensions api.Extensions) (*AuthorizeResponse, error) {
	var xmlResponse internal.AuthResponseXML

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := xml.NewDecoder(resp.Body).Decode(&xmlResponse); err != nil {
		return nil, err
	}
	response := &AuthorizeResponse{
		Reason:     xmlResponse.Code,
		success:    xmlResponse.Authorized,
		StatusCode: resp.StatusCode,
	}

	if reportLen := len(xmlResponse.UsageReports.Reports); reportLen > 0 {
		response.usageReports = c.convertXmlUsageReports(xmlResponse.UsageReports.Reports, reportLen)
	}

	if extensions != nil {
		response = c.handleAuthExtensions(xmlResponse, resp, extensions, response)
	}

	return response, err
}

func (c *Client) executeReportCall(req *http.Request, extensions api.Extensions) (*ReportResponse, error) {
	var xmlResponse internal.ReportErrorXML

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// ensure response is in 2xx range
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {

		if err := xml.NewDecoder(resp.Body).Decode(&xmlResponse); err != nil {
			return nil, err
		}
		return &ReportResponse{
			accepted:   false,
			Reason:     xmlResponse.Code,
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &ReportResponse{
		accepted:   true,
		StatusCode: resp.StatusCode,
	}, nil
}

// handleAuthExtensions handles known extensions
// extensions must not be nil
func (c *Client) handleAuthExtensions(xmlResp internal.AuthResponseXML, resp *http.Response, extensions api.Extensions, annotatedResp *AuthorizeResponse) *AuthorizeResponse {
	if _, ok := extensions[api.HierarchyExtension]; ok {
		annotatedResp.hierarchy = c.convertXmlHierarchy(xmlResp.Hierarchy)
	}

	if _, ok := extensions[api.LimitExtension]; ok {
		annotatedResp.rateLimits = c.handleRateLimitExtensions(resp)
	}

	return annotatedResp
}

func (c *Client) convertXmlUsageReports(xmlReports []internal.UsageReportXML, mapLen int) api.UsageReports {
	usageReports := make(api.UsageReports, mapLen)
	for _, report := range xmlReports {
		if converted, err := convertXmlToUsageReport(report); err == nil {
			//nothing we can do here if we hit an error besides continue
			usageReports[report.Metric] = converted
		}
	}
	return usageReports
}

func (c *Client) convertXmlHierarchy(xmlHierarchy internal.HierarchyXML) api.Hierarchy {
	hierarchy := make(api.Hierarchy, len(xmlHierarchy.Metric))
	for _, i := range xmlHierarchy.Metric {
		if i.Children != "" {
			children := strings.Split(i.Children, " ")
			for _, child := range children {
				// avoid duplication
				if !contains(child, hierarchy[i.Name]) {
					hierarchy[i.Name] = append(hierarchy[i.Name], child)
				}
			}
		}
	}
	return hierarchy
}

// handleRateLimitExtensions parses the provided http response for extensions and appends their information to the provided AuthorizeResponse.
// Provides a best effort and if we hit an error during handling extensions, we do not tarnish the overall valid response,
// instead treating it as corrupt and choose to remove the information learned from the extension
func (c *Client) handleRateLimitExtensions(resp *http.Response) *api.RateLimits {
	rl := &api.RateLimits{}

	if limitRem := resp.Header.Get(limitRemainingHeaderKey); limitRem != "" {
		if remainingLimit, err := strconv.Atoi(limitRem); err == nil {
			rl.LimitRemaining = remainingLimit
		}
	}

	if limReset := resp.Header.Get(limitResetHeaderKey); limReset != "" {
		if resetLimit, err := strconv.Atoi(limReset); err == nil {
			rl.LimitReset = resetLimit
		}
	}
	return rl
}

func (c *Client) wrapError(err error) error {
	return fmt.Errorf("%s - %s ", errHttpReq.Error(), err.Error())
}

// GetRateLimits for auth/authrep request if the extension was enabled
func (ar AuthorizeResponse) GetRateLimits() *api.RateLimits {
	return ar.rateLimits
}

// GetHierarchy returns the responses Hierarchy if the extension was enabled
func (ar AuthorizeResponse) GetHierarchy() api.Hierarchy {
	return ar.hierarchy
}

// GetUsageReports returns the responses UsageReports if the extension was enabled
func (ar AuthorizeResponse) GetUsageReports() api.UsageReports {
	return ar.usageReports
}

// GetUsageReports returns the responses UsageReports if the extension was enabled
func (ar AuthorizeResponse) Success() bool {
	return ar.success
}

// Accepted returns true if the report request has been accepted for processing by 3scale
func (rr ReportResponse) Accepted() bool {
	return rr.accepted
}

type kind int

const (
	auth kind = iota
	authRep
	report
)

// Verifies a custom backend is valid
func verifyBackendUrl(urlToCheck string) (*url.URL, error) {
	backendURL, err := url.ParseRequestURI(urlToCheck)
	if err == nil {
		scheme := backendURL.Scheme
		if scheme != "" && scheme != "http" && scheme != "https" {
			err = fmt.Errorf("unsupported scheme %s passed to backend", scheme)
		}

	}
	return backendURL, err
}

func defaultHttpClient() *http.Client {
	c := http.DefaultClient
	c.Timeout = defaultTimeout
	return c
}

func contains(key string, in []string) bool {
	for _, i := range in {
		if key == i {
			return true
		}
	}
	return false
}

// convert an xml decoded response into a user friendly UsageReport
func convertXmlToUsageReport(ur internal.UsageReportXML) (api.UsageReport, error) {
	var err error
	report := api.UsageReport{
		MaxValue:     ur.MaxValue,
		CurrentValue: ur.CurrentValue,
	}

	pw := api.PeriodWindow{
		Period: ur.Period,
	}

	parseTime := func(timestamp string) (int64, error) {
		t, err := time.Parse(timeLayout, timestamp)
		if err != nil {
			return 0, err
		}
		return t.Unix(), nil
	}

	if pw.Start, err = parseTime(ur.PeriodStart); err != nil {
		return report, err
	}

	if pw.End, err = parseTime(ur.PeriodEnd); err != nil {
		return report, err
	}

	report.PeriodWindow = pw
	return report, err
}
