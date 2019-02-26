package client

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const (
	defaultBackendUrl       = "https://su1.3scale.net:443"
	queryTag                = "query"
	limitExtensions         = "limit_headers"
	limitRemainingHeaderKey = "3scale-limit-remaining"
	limitResetHeaderKey     = "3scale-limit-reset"
)

var httpReqError = errors.New("error building http request")

// Returns a Backend which will interact with a SaaS based 3scale backend
func DefaultBackend() *Backend {
	url2, err := verifyBackendUrl(defaultBackendUrl)
	if err != nil {
		panic("error parsing default backend")
	}
	return &Backend{baseUrl: url2}
}

// Returns a custom Backend
// Can be used for on-premise installations
// Supported schemes are http and https
func NewBackend(scheme string, host string, port int) (*Backend, error) {
	url2, err := verifyBackendUrl(fmt.Sprintf("%s://%s:%d", scheme, host, port))
	if err != nil {
		return nil, err
	}
	return &Backend{scheme, host, port, url2}, nil
}

// Creates a ThreeScaleClient to communicate with the provided backend.
// If Backend is nil, the default SaaS backend will be used
// If http Client is nil, the default http client will be used
func NewThreeScale(backEnd *Backend, httpClient *http.Client) *ThreeScaleClient {
	if backEnd == nil {
		backEnd = DefaultBackend()
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &ThreeScaleClient{backEnd, httpClient}
}

// GetPeer - a utility method that returns the remote hostname of the client
func (client *ThreeScaleClient) GetPeer() string {
	return client.backend.host
}

// Request builder for GET request to the provided endpoint
func (client *ThreeScaleClient) buildGetReq(ep string, extensions map[string]string) (*http.Request, error) {
	path := &url.URL{Path: ep}
	req, err := http.NewRequest("GET", client.backend.baseUrl.ResolveReference(path).String(), nil)
	req.Header.Set("Accept", "application/xml")

	if extensions != nil {
		req.Header.Set("3scale-options", encodeExtensions(extensions))
	}

	return req, err
}

func encodeExtensions(extensions map[string]string) string {
	var exts string

	if extensions != nil {
		for k, v := range extensions {
			// the extensions mechanism requires escaping keys and values
			// we are using QueryEscape because it escapes characters that
			// PathEscape does not and are needed to disambiguate (ie. '=').
			k = url.QueryEscape(k)
			v = url.QueryEscape(v)

			// add separator if needed
			if exts != "" {
				exts = exts + "&"
			}

			exts = exts + fmt.Sprintf("%s=%s", k, v)
		}
	}

	return exts
}

// Call 3scale backend with the provided HTTP request
func (client *ThreeScaleClient) doHttpReq(req *http.Request, ext map[string]string) (ApiResponse, error) {
	var authRepRes ApiResponse

	resp, err := client.httpClient.Do(req)
	defer resp.Body.Close()

	if err != nil {
		return authRepRes, err
	}

	authRepRes, err = getApiResp(resp.Body)

	if err != nil {
		return authRepRes, err
	}

	authRepRes.StatusCode = resp.StatusCode

	if ext != nil {
		if _, ok := ext[limitExtensions]; ok {
			authRepRes.RateLimits = &RateLimits{}
			if limitRem := resp.Header.Get(limitRemainingHeaderKey); limitRem != "" {
				remainingLimit, err := strconv.Atoi(limitRem)
				if err != nil {
					authRepRes.RateLimits = nil
					goto out
				}
				authRepRes.RateLimits.limitRemaining = remainingLimit
			}

			if limReset := resp.Header.Get(limitResetHeaderKey); limReset != "" {
				resetLimit, err := strconv.Atoi(limReset)
				if err != nil {
					authRepRes.RateLimits = nil
					goto out
				}
				authRepRes.RateLimits.limitReset = resetLimit
			}
		}
	}

out:
	return authRepRes, nil
}

// GetLimitRemaining - An integer stating the amount of hits left for the full combination of metrics authorized in this call
// before the rate limiting logic would start denying authorizations for the current period.
// A value of -1 indicates there is no limit in the amount of hits.
// Nil value will indicate the extension has not been used.
func (r RateLimits) GetLimitRemaining() int {
	return r.limitRemaining
}

// GetLimitReset - An integer stating the amount of seconds left for the current limiting period to elapse.
// A value of -1 indicates there i is no limit in time.
// Nil value will indicate the extension has not been used.
func (r RateLimits) GetLimitReset() int {
	return r.limitReset
}

// GetHierarchy - A list of children (methods) associated with a parent(metric)
func (r ApiResponse) GetHierarchy() map[string][]string {
	return r.hierarchy
}

// Add a metric to list of metrics to be reported
// Returns error if provided value is non-positive and entry will be ignored
func (m Metrics) Add(name string, value int) error {
	if value < 1 {
		return fmt.Errorf("integer value for metric %s must be positive", name)
	}
	m[name] = value
	return nil
}

// Converts a Metrics type into formatted map as expected by 3scale API
func (m Metrics) convert() map[string]string {
	formatted := make(map[string]string, len(m))
	for k, v := range m {
		if v > 0 {
			formatted[fmt.Sprintf("usage[%s]", k)] = strconv.Itoa(v)
		}
	}
	return formatted
}

// Set a Log value - expects plain text which will be url encoded
func (l Log) Set(request string, response string, statusCode int) {
	l["request"] = url.QueryEscape(request)
	l["response"] = url.QueryEscape(response)
	l["code"] = url.QueryEscape(strconv.Itoa(statusCode))
}

// Converts a Log type into formatted map as expected by 3scale API
func (l Log) convert() map[string]string {
	formatted := make(map[string]string, 3)
	for k, v := range l {
		if v != "" {
			formatted[fmt.Sprintf("log[%s]", k)] = v
		}
	}
	return formatted
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

// Verifies a custom backend is valid
func verifyBackendUrl(urlToCheck string) (*url.URL, error) {
	url2, err := url.ParseRequestURI(urlToCheck)
	if err == nil {
		if url2.Scheme != "http" && url2.Scheme != "https" {
			err = fmt.Errorf("unsupported schema %s passed to backend", url2.Scheme)
		}

	}
	return url2, err
}

// Wrapper function for XML response from 3scale API
func getApiResp(r io.Reader) (ApiResponse, error) {
	var resp ApiResponse
	var apiResp ApiResponseXML

	if err := xml.NewDecoder(r).Decode(&apiResp); err != nil {
		return resp, err
	}
	resp.Success = apiResp.Authorized
	if !apiResp.Authorized {
		if apiResp.Reason != "" {
			resp.Reason = apiResp.Reason
		} else if apiResp.Code != "" {
			resp.Reason = apiResp.Code
		}
	}

	if len(apiResp.Hierarchy.Metric) > 0 {
		resp.hierarchy = make(map[string][]string, len(apiResp.Hierarchy.Metric))
		for _, i := range apiResp.Hierarchy.Metric {
			if i.Children != "" {
				children := strings.Split(i.Children, " ")
				for _, child := range children {
					if !contains(child, resp.hierarchy[i.Name]) {
						resp.hierarchy[i.Name] = append(resp.hierarchy[i.Name], child)
					}
				}
			}
		}
	}
	return resp, nil
}

func contains(key string, in []string) bool {
	for _, i := range in {
		if key == i {
			return true
		}
	}
	return false
}

// Helper function to read custom tags and add them to query string
// Returns a list of values formatted as expected by 3scale API
func parseQueries(obj interface{}, values url.Values, m Metrics, l Log) url.Values {
	if obj == nil {
		return values
	}

	v := reflect.Indirect(reflect.ValueOf(obj))
	for i := 0; i < v.NumField(); i++ {

		if v.Type().Field(i).Type.Kind() == reflect.Struct {
			parseQueries(v.Field(i).Interface(), values, nil, nil)
			continue
		}

		tag := v.Type().Field(i).Tag.Get(queryTag)
		if tag != "" {
			var queryVal string
			tagVal := v.Field(i).Interface()
			switch tagVal.(type) {
			case string, int:
				queryVal = fmt.Sprintf("%v", tagVal)
			default:
				continue
			}
			if queryVal != "" {
				values.Add(tag, queryVal)
			}

		}
	}

	for k, v := range m.convert() {
		values.Add(k, v)
	}

	for k, v := range l.convert() {
		values.Add(k, v)
	}

	return values
}
