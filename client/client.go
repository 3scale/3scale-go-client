package client

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strconv"
)

const (
	defaultBackendUrl = "https://su1.3scale.net:443"
	queryTag          = "query"
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

// Request builder for GET request to the provided endpoint
func (client *ThreeScaleClient) buildGetReq(ep string) (*http.Request, error) {
	path := &url.URL{Path: ep}
	req, err := http.NewRequest("GET", client.backend.baseUrl.ResolveReference(path).String(), nil)
	req.Header.Set("Accept", "application/xml")
	return req, err
}

// Call 3scale backend with the provided HTTP request
func (client *ThreeScaleClient) doHttpReq(req *http.Request) (ApiResponse, error) {
	var authRepRes ApiResponse

	//TODO Remove debug code
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
	// End TODO

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
	return authRepRes, nil
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
	return resp, nil
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
