package http

import (
	"encoding/json"
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

	statusEndpoint = "/status"
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
	// RejectionReasonHeader - This is used by authorization endpoints to provide a header that provides an error code
	// describing the different reasons an authorization can be denied.
	RejectionReasonHeaderExtension = "rejection_reason_header"
	// NoBodyExtension instructs backend to avoid generating response bodies for certain endpoints.
	// In particular, this is useful to avoid generating large response in the authorization endpoints
	NoBodyExtension = "no_body"

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
func (c *Client) Authorize(apiCall threescale.Request) (*threescale.AuthorizeResult, error) {
	return c.AuthorizeWithOptions(apiCall)
}

// AuthorizeWithOptions provides the same behaviour as Authorize with additional functionality provided by Option(s)
func (c *Client) AuthorizeWithOptions(apiCall threescale.Request, options ...Option) (*threescale.AuthorizeResult, error) {
	return c.doAuthOrAuthRep(apiCall, auth, newOptions(options...))
}

// AuthRep should be used to authorize and report, in a single transaction
// for an application with the authentication provided in the transaction params
func (c *Client) AuthRep(apiCall threescale.Request) (*threescale.AuthorizeResult, error) {
	return c.AuthRepWithOptions(apiCall)
}

// AuthRepWithOptions provides the same behaviour as AuthRep with additional functionality provided by Option(s)
func (c *Client) AuthRepWithOptions(apiCall threescale.Request, options ...Option) (*threescale.AuthorizeResult, error) {
	return c.doAuthOrAuthRep(apiCall, authRep, newOptions(options...))
}

func (c *Client) Report(apiCall threescale.Request) (*threescale.ReportResult, error) {
	return c.ReportWithOptions(apiCall)
}

// ReportWithOptions provides the same behaviour as Report with additional functionality provided by Option(s)
func (c *Client) ReportWithOptions(apiCall threescale.Request, options ...Option) (*threescale.ReportResult, error) {
	return c.doReport(apiCall, newOptions(options...))
}

// GetPeer returns the hostname of the backend for the client
func (c *Client) GetPeer() string {
	return c.backendHost
}

// GetVersion returns the version of the backend for this client (remote call)
func (c *Client) GetVersion() (string, error) {
	var version string
	var statusResponse internal.StatusResponse

	req, err := http.NewRequest(http.MethodGet, c.baseURL+statusEndpoint, nil)
	if err != nil {
		return version, fmt.Errorf("failed to build request for status endpoint - %s", err.Error())
	}
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return version, fmt.Errorf("failed to fetch backend version - %s", err.Error())
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&statusResponse)
	if err != nil {
		return version, fmt.Errorf("failed to fetch backend version - %s", err.Error())
	}

	return statusResponse.Version.Backend, nil
}

func (c *Client) doAuthOrAuthRep(apiCall threescale.Request, kind kind, options *Options) (*threescale.AuthorizeResult, error) {
	req, err := requestBuilder{}.build(apiCall, c.baseURL, kind)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return c.executeAuthCall(req, apiCall.Extensions, options)
}

func (c *Client) doReport(apiCall threescale.Request, options *Options) (*threescale.ReportResult, error) {
	req, err := requestBuilder{}.build(apiCall, c.baseURL, report)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return c.executeReportCall(req, apiCall.Extensions, options)
}

func (c *Client) executeAuthCall(req *http.Request, extensions api.Extensions, options *Options) (*threescale.AuthorizeResult, error) {
	if options != nil && options.context != nil {
		req = req.WithContext(options.context)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	requestDuration := time.Since(start)
	defer resp.Body.Close()

	go func() {
		if options != nil && options.instrumentationCB != nil {
			options.instrumentationCB(options.context, c.GetPeer(), resp.StatusCode, requestDuration)
		}
	}()

	if resp.StatusCode >= 500 {
		return &threescale.AuthorizeResult{
			Authorized:  false,
			RawResponse: resp,
		}, fmt.Errorf("unable to process request - status: %s", resp.Status)
	}

	if val, ok := extensions[NoBodyExtension]; ok && val == "1" {
		return c.handleNoBodyExtensionForAuth(resp, extensions), nil
	}

	return c.handleAuthXMLResp(resp, extensions)
}

func (c *Client) handleAuthXMLResp(resp *http.Response, extensions api.Extensions) (*threescale.AuthorizeResult, error) {
	var xmlResponse internal.AuthResponseXML

	if err := xml.NewDecoder(resp.Body).Decode(&xmlResponse); err != nil {
		return nil, err
	}

	return &threescale.AuthorizeResult{
		Authorized:   xmlResponse.Authorized,
		UsageReports: c.convertXmlUsageReports(xmlResponse.UsageReports.Reports),
		ErrorCode: func(code string, resp *http.Response) string {
			if headerCode := c.parseRejectionReasonHeader(resp); headerCode != "" {
				return headerCode
			}
			return code
		}(xmlResponse.Code, resp),
		RejectionReason:     xmlResponse.Reason,
		AuthorizeExtensions: c.handleAuthExtensions(xmlResponse, resp, extensions),
		RawResponse:         resp,
	}, nil
}

func (c *Client) executeReportCall(req *http.Request, extensions api.Extensions, options *Options) (*threescale.ReportResult, error) {
	if options != nil && options.context != nil {
		req = req.WithContext(options.context)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	requestDuration := time.Since(start)
	defer resp.Body.Close()

	go func() {
		if options != nil && options.instrumentationCB != nil {
			options.instrumentationCB(options.context, c.GetPeer(), resp.StatusCode, requestDuration)
		}
	}()

	// ensure response is in 2xx range
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return c.handleReportingError(resp)
	}

	return &threescale.ReportResult{
		Accepted:    true,
		RawResponse: resp,
	}, nil
}

func (c *Client) handleReportingError(resp *http.Response) (*threescale.ReportResult, error) {
	if resp.StatusCode >= 500 {
		return &threescale.ReportResult{
			Accepted:    false,
			RawResponse: resp,
		}, fmt.Errorf("unable to process request - status: %s", resp.Status)
	}

	var xmlResponse internal.ReportErrorXML
	if err := xml.NewDecoder(resp.Body).Decode(&xmlResponse); err != nil {
		return nil, err
	}
	return &threescale.ReportResult{
		Accepted:    false,
		ErrorCode:   xmlResponse.Code,
		RawResponse: resp,
	}, nil
}

// handleAuthExtensions handles known extensions
// extensions must not be nil
func (c *Client) handleAuthExtensions(xmlResp internal.AuthResponseXML, resp *http.Response, extensions api.Extensions) threescale.AuthorizeExtensions {
	var annotatedExts threescale.AuthorizeExtensions
	if extensions == nil {
		return annotatedExts
	}
	if _, ok := extensions[api.HierarchyExtension]; ok {
		annotatedExts.Hierarchy = c.convertXmlHierarchy(xmlResp.Hierarchy)
	}

	if _, ok := extensions[api.LimitExtension]; ok {
		annotatedExts.RateLimits = c.handleRateLimitExtensions(resp)
	}

	return annotatedExts
}

func (c *Client) convertXmlUsageReports(xmlReports []internal.UsageReportXML) api.UsageReports {
	if len(xmlReports) == 0 {
		return nil
	}
	usageReports := make(api.UsageReports)
	for _, report := range xmlReports {
		if converted, err := convertXmlToUsageReport(report); err == nil {
			//nothing we can do here if we hit an error besides continue
			currentReports := usageReports[report.Metric]
			usageReports[report.Metric] = append(currentReports, converted)
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

func (c *Client) handleNoBodyExtensionForAuth(resp *http.Response, extensions api.Extensions) *threescale.AuthorizeResult {
	var rl *api.RateLimits
	if _, ok := extensions[api.LimitExtension]; ok {
		rl = c.handleRateLimitExtensions(resp)
	}

	if resp.StatusCode == http.StatusOK {
		return &threescale.AuthorizeResult{
			Authorized:  true,
			RawResponse: resp,
			AuthorizeExtensions: threescale.AuthorizeExtensions{
				RateLimits: rl,
			},
		}
	}

	return &threescale.AuthorizeResult{
		Authorized:  false,
		ErrorCode:   c.parseRejectionReasonHeader(resp),
		RawResponse: resp,
		AuthorizeExtensions: threescale.AuthorizeExtensions{
			RateLimits: rl,
		},
	}

}

func (c *Client) parseRejectionReasonHeader(resp *http.Response) string {
	return resp.Header.Get("3scale-Rejection-Reason")
}

func (c *Client) wrapError(err error) error {
	return fmt.Errorf("%s - %s ", errHttpReq.Error(), err.Error())
}

// CodeToStatusCode transforms a client response code to http status code.
// See https://github.com/3scale/apisonator/blob/v2.96.2/docs/rfcs/error_responses.md
func CodeToStatusCode(errorCode string) int {
	transform := map[string]int{
		"access_token_storage_error":             http.StatusBadRequest,
		"not_valid_data":                         http.StatusBadRequest,
		"bad_request":                            http.StatusBadRequest,
		"access_token_already_exists":            http.StatusBadRequest,
		"content_type_invalid":                   http.StatusBadRequest,
		"provider_key_invalid":                   http.StatusForbidden,
		"user_requires_registration":             http.StatusForbidden,
		"user_key_invalid":                       http.StatusForbidden,
		"authentication_error":                   http.StatusForbidden,
		"provider_key_or_service_token_required": http.StatusForbidden,
		"service_token_invalid":                  http.StatusForbidden,
		"application_not_found":                  http.StatusNotFound,
		"application_token_invalid":              http.StatusNotFound,
		"service_id_invalid":                     http.StatusNotFound,
		"metric_invalid":                         http.StatusNotFound,
		"limits_exceeded":                        http.StatusConflict,
		"oauth_not_enabled":                      http.StatusConflict,
		"redirect_uri_invalid":                   http.StatusConflict,
		"redirect_url_invalid":                   http.StatusConflict,
		"application_not_active":                 http.StatusConflict,
		"application_key_invalid":                http.StatusConflict,
		"referrer_not_allowed":                   http.StatusConflict,
		"application_has_inconsistent_data":      http.StatusUnprocessableEntity,
		"referrer_filter_invalid":                http.StatusUnprocessableEntity,
		"required_params_missing":                http.StatusUnprocessableEntity,
		"usage_value_invalid":                    http.StatusUnprocessableEntity,
		"service_id_missing":                     http.StatusUnprocessableEntity,
	}[errorCode]
	return transform
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

var granularityMap = map[string]api.Period{
	"minute":   api.Minute,
	"hour":     api.Hour,
	"day":      api.Day,
	"week":     api.Week,
	"month":    api.Month,
	"year":     api.Year,
	"eternity": api.Eternity,
}

// convert an xml decoded response into a user friendly UsageReport
func convertXmlToUsageReport(ur internal.UsageReportXML) (api.UsageReport, error) {
	var err error
	report := api.UsageReport{
		MaxValue:     ur.MaxValue,
		CurrentValue: ur.CurrentValue,
	}

	pw := api.PeriodWindow{
		Period: granularityMap[ur.Period],
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
