package threescale

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	authzEndpoint   = "/transactions/authorize.xml"
	authRepEndpoint = "/transactions/authrep.xml"
	reportEndpoint  = "/transactions.xml"
)

var (
	httpReqError = errors.New(httpReqErrText)
)

// Authorize is a read-only operation to authorize an application with the authentication provided in the requests params
func (c *Client) Authorize(serviceID string, auth ClientAuth, request *Request) (*AuthorizeResponse, error) {
	return c.authOrAuthRep(authzEndpoint, serviceID, auth, request)
}

// AuthRep should be used to authorize and report, in a single request
// for an application with the authentication provided in the requests params
func (c *Client) AuthRep(serviceID string, auth ClientAuth, request *Request) (*AuthorizeResponse, error) {
	return c.authOrAuthRep(authRepEndpoint, serviceID, auth, request)
}

// Report the transactions to 3scale backend with the authentication provided in the requests params
func (c *Client) Report(serviceID string, auth ClientAuth, reqs ...*Request) (*ReportResponse, error) {
	values := auth.joinToValues(url.Values{serviceIDKey: []string{serviceID}})
	for index, req := range reqs {
		req.convertAndAddToTransactionValues(values, index, req)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+reportEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%s - %s ", httpReqError.Error(), err.Error())
	}

	if len(reqs) == 1 {
		req = c.annotateRequest(reqs[0], req)
	}
	return c.doReportReq(req, values)
}

func (c *Client) authOrAuthRep(endpoint, serviceID string, auth ClientAuth, request *Request) (*AuthorizeResponse, error) {
	// build out http request for the provided Request object
	req, err := c.buildGetReq(c.baseURL+endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("%s - %s ", httpReqError.Error(), err.Error())
	}
	// take the user input and encode to query string formatted to the expectations of 3scale backend
	req.URL.RawQuery = c.inputToValues(serviceID, request, auth).Encode()
	return c.doAuthorizeReq(req, request.extensions)
}

// GetPeer is a utility method that returns the remote hostname of the client
func (c *Client) GetPeer() string {
	return c.backendHost
}

// Call 3scale backend with the provided HTTP request
func (c *Client) doAuthorizeReq(req *http.Request, extensions Extensions) (*AuthorizeResponse, error) {
	var xmlResponse ApiAuthResponseXML

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if err := xml.NewDecoder(resp.Body).Decode(&xmlResponse); err != nil {
		return nil, err
	}
	response := &AuthorizeResponse{
		Reason:     xmlResponse.Reason,
		Success:    xmlResponse.Authorized,
		StatusCode: resp.StatusCode,
	}

	reportLen := len(xmlResponse.UsageReports.Reports)
	if reportLen > 0 {
		response.usageReports = make(UsageReports, reportLen)
		for _, report := range xmlResponse.UsageReports.Reports {
			if converted, err := report.convert(); err == nil {
				//nothing we can do here if we hit an error besides continue
				response.usageReports[report.Metric] = converted
			}
		}
	}

	hierarchyLen := len(xmlResponse.Hierarchy.Metric)
	if hierarchyLen > 0 {
		response.hierarchy = make(map[string][]string, hierarchyLen)
		for _, i := range xmlResponse.Hierarchy.Metric {
			if i.Children != "" {
				children := strings.Split(i.Children, " ")
				for _, child := range children {
					// avoid duplication
					if !contains(child, response.hierarchy[i.Name]) {
						response.hierarchy[i.Name] = append(response.hierarchy[i.Name], child)
					}
				}
			}
		}
	}
	return c.handleAuthorizeExtensions(resp, response, extensions), nil
}

func (c *Client) doReportReq(req *http.Request, values url.Values) (*ReportResponse, error) {
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// ensure response is in 2xx range
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		var xmlResponse ReportErrorXML
		if err := xml.NewDecoder(resp.Body).Decode(&xmlResponse); err != nil {
			return nil, err
		}
		return &ReportResponse{
			Accepted:   false,
			Reason:     xmlResponse.Code,
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &ReportResponse{
		Accepted:   true,
		StatusCode: resp.StatusCode,
	}, nil
}

// handleAuthorizeExtensions parses the provided http response for extensions and appends their information to the provided AuthorizeResponse.
// Provides a best effort and if we hit an error during handling extensions, we do not tarnish the overall valid response,
// instead treating it as corrupt and choose to remove the information learned from the extension
func (c *Client) handleAuthorizeExtensions(resp *http.Response, response *AuthorizeResponse, extensions Extensions) *AuthorizeResponse {
	if _, ok := extensions[LimitExtension]; ok {
		response.RateLimits = &RateLimits{}
		if limitRem := resp.Header.Get(limitRemainingHeaderKey); limitRem != "" {
			if remainingLimit, err := strconv.Atoi(limitRem); err == nil {
				response.RateLimits.limitRemaining = remainingLimit
			}
		}

		if limReset := resp.Header.Get(limitResetHeaderKey); limReset != "" {
			if resetLimit, err := strconv.Atoi(limReset); err == nil {
				response.RateLimits.limitReset = resetLimit
			}
		}
	}
	return response
}

func (c *Client) inputToValues(svcID string, req *Request, clientAuth ClientAuth) url.Values {
	values := make(url.Values)
	values.Add(serviceIDKey, svcID)
	values = req.Params.joinToValues(values)
	values = req.Metrics.joinToValues(values)
	values = clientAuth.joinToValues(values)
	return values
}

func (c *Client) buildGetReq(url string, request *Request) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return req, err

	}
	return c.annotateRequest(request, req), nil
}

// annotateRequest modifies the *http.Request with required metadata and formatting for 3scale
func (c *Client) annotateRequest(request *Request, httpReq *http.Request) *http.Request {
	httpReq.Header.Set("Accept", "application/xml")

	if request.extensions != nil {
		httpReq.Header.Set(enableExtensions, encodeExtensions(request.extensions))
	}

	if request.context != nil {
		httpReq = httpReq.WithContext(request.context)
	}

	return httpReq
}
