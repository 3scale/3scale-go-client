package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/3scale/3scale-go-client/threescale"

	"github.com/3scale/3scale-go-client/fake"
	"github.com/3scale/3scale-go-client/threescale/api"
)

func TestClient_Authorize(t *testing.T) {
	const svcID = "test"

	inputs := []struct {
		name           string
		auth           api.ClientAuth
		transaction    api.Transaction
		expectErr      bool
		expectErrMsg   string
		extensions     api.Extensions
		expectResponse *AuthorizeResponse
		client         *Client
		injectClient   *http.Client
	}{
		{
			name:         "Test expect failure bad url passed",
			auth:         api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction:  api.Transaction{Params: api.Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: httpReqErrText,
			client: &Client{
				backendHost: "/some/invalid/value%_",
				baseURL:     "/some/invalid/value%_",
				httpClient:  http.DefaultClient,
			},
		},
		{
			name:         "Test expect failure simulated network error",
			auth:         api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction:  api.Transaction{Params: api.Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: "Timeout exceeded",
			client: &Client{
				baseURL: defaultBackendUrl,
				httpClient: &http.Client{
					Timeout: time.Nanosecond,
				},
			},
		},
		{
			name:         "Test expect failure simulated bad response from 3scale error",
			auth:         api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction:  api.Transaction{Params: api.Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: "EOF",
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("EOF")),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test params formatting",
			auth: api.ClientAuth{
				Type:  api.ServiceToken,
				Value: "any",
			},
			transaction: api.Transaction{
				Params: api.Params{
					AppID:  "any",
					AppKey: "key",
				},
				Metrics: api.Metrics{"hits": 1, "other": 2},
			},
			expectResponse: &AuthorizeResponse{
				success:    true,
				StatusCode: 200,
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				// decodes to app_id=any&app_key=key&service_id=test&service_token=any&usage[hits]=1&usage[other]=2
				expect := `app_id=any&app_key=key&service_id=test&service_token=any&usage%5Bhits%5D=1&usage%5Bother%5D=2`

				if req.URL.RawQuery != expect {
					t.Error("unexpected result in query string")
				}

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test extension formatting",
			auth: api.ClientAuth{
				Type:  api.ServiceToken,
				Value: "any",
			},
			transaction: api.Transaction{
				Params: api.Params{
					AppID: "any",
				},
			},
			expectResponse: &AuthorizeResponse{
				success:    true,
				StatusCode: 200,
			},
			extensions: getExtensions(t),
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				ok, errMsg := checkExtensions(t, req)
				if !ok {
					t.Errorf("error in extensions - %s", errMsg)
				}

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name:        "Test usage reports",
			auth:        api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction: api.Transaction{Params: api.Params{AppID: "any"}},
			expectResponse: &AuthorizeResponse{
				success:    true,
				StatusCode: 200,
				usageReports: api.UsageReports{
					"hits": api.UsageReport{
						PeriodWindow: api.PeriodWindow{
							Period: api.Minute,
							Start:  1550845920,
							End:    1550845980,
						},
						MaxValue:     4,
						CurrentValue: 1,
					},
					"test_metric": api.UsageReport{
						PeriodWindow: api.PeriodWindow{
							Period: api.Week,
							Start:  1550448000,
							End:    1551052800,
						},
						MaxValue:     6,
						CurrentValue: 0,
					},
				},
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				equals(t, req.URL.Path, authzEndpoint)
				resp := getUsageReportXML(t)

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name:        "Test hierarchy extension",
			auth:        api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction: api.Transaction{Params: api.Params{AppID: "any"}},
			expectResponse: &AuthorizeResponse{
				success:    true,
				StatusCode: 200,
				hierarchy:  api.Hierarchy{"hits": []string{"example", "sample", "test"}},
			},
			extensions: api.Extensions{api.HierarchyExtension: "1"},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				expectValSet := req.Header.Get("3scale-Options")
				if expectValSet != "hierarchy=1" {
					t.Error("expected hierarchy feature to have been enabled via header")
				}
				equals(t, req.URL.Path, authzEndpoint)
				resp := getHierarchyXML(t)

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name:        "Test authorization extensions - rate limiting",
			auth:        api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction: api.Transaction{Params: api.Params{AppID: "any"}},
			extensions:  api.Extensions{api.LimitExtension: "1"},
			expectResponse: &AuthorizeResponse{
				success:    true,
				StatusCode: 200,
				rateLimits: &api.RateLimits{
					LimitRemaining: 5,
					LimitReset:     100,
				},
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				if strings.Contains(req.URL.RawQuery, "usage") {
					t.Error("unexpected usage has been generated for empty transaction")
				}
				expectValSet := req.Header.Get("3scale-Options")
				if expectValSet != "limit_headers=1" {
					t.Error("expected rate limiting feature to have been enabled via header")
				}
				equals(t, req.URL.Path, authzEndpoint)

				header := http.Header{}
				header.Add(limitRemainingHeaderKey, "5")
				header.Add(limitResetHeaderKey, "100")

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     header,
				}
			}),
		},
	}

	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			if input.injectClient == nil {
				// fallback client
				input.injectClient = NewTestClient(func(req *http.Request) *http.Response {
					equals(t, req.URL.Path, authzEndpoint)
					return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess()))}
				})
			}

			c := input.client
			if c == nil {
				c = threeScaleTestClient(t, input.injectClient)
			}

			apiCall := threescale.Request{
				Auth:         input.auth,
				Extensions:   input.extensions,
				Service:      svcID,
				Transactions: []api.Transaction{input.transaction},
			}

			resp, err := c.Authorize(apiCall)
			if err != nil {
				if !input.expectErr {
					t.Error("unexpected error")
				}
				// we expected an error so ensure our err conditions are met
				if !strings.Contains(err.Error(), input.expectErrMsg) {
					t.Errorf("expected our error message to contain substring %s", input.expectErrMsg)
				}
				return
			}

			equals(t, input.expectResponse, resp)
			equals(t, input.expectResponse.rateLimits, resp.GetRateLimits())
			equals(t, input.expectResponse.hierarchy, resp.GetHierarchy())
			equals(t, input.expectResponse.usageReports, resp.GetUsageReports())
			equals(t, input.expectResponse.success, resp.Success())
		})
	}
}

// because auth and auth rep essentially follow the same pattern, we can minimise the test in this instance
// ensure our query param is correct and we are calling the correct endpoint
func TestClient_AuthRep(t *testing.T) {
	const svcID = "test"
	type input struct {
		name           string
		auth           api.ClientAuth
		transaction    api.Transaction
		extensions     api.Extensions
		expectErr      bool
		expectErrMsg   string
		expectResponse *AuthorizeResponse
		client         *Client
		injectClient   *http.Client
	}

	inputs := []input{
		{
			name:         "Test expect failure bad url passed",
			auth:         api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transaction:  api.Transaction{Params: api.Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: httpReqErrText,
			client: &Client{
				backendHost: "/some/invalid/value%_",
				baseURL:     "/some/invalid/value%_",
				httpClient:  http.DefaultClient,
			},
		},
		{
			name: "Test params formatting",
			auth: api.ClientAuth{
				Type:  api.ServiceToken,
				Value: "any",
			},
			transaction: api.Transaction{
				Params: api.Params{
					AppID:  "any",
					AppKey: "key",
				},
				Metrics: api.Metrics{"hits": 1, "other": 2},
			},
			expectResponse: &AuthorizeResponse{
				success:    true,
				StatusCode: 200,
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				equals(t, req.URL.Path, authRepEndpoint)
				// decodes to app_id=any&app_key=key&service_id=test&service_token=any&usage[hits]=1&usage[other]=2
				expect := `app_id=any&app_key=key&service_id=test&service_token=any&usage%5Bhits%5D=1&usage%5Bother%5D=2`

				if req.URL.RawQuery != expect {
					t.Error("unexpected result in query string")
				}

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     make(http.Header),
				}
			}),
		},
	}

	for _, fixture := range inputs {
		c := fixture.client
		if c == nil {
			c = threeScaleTestClient(t, fixture.injectClient)
		}

		apiCall := threescale.Request{
			Auth:         fixture.auth,
			Extensions:   fixture.extensions,
			Service:      svcID,
			Transactions: []api.Transaction{fixture.transaction},
		}

		resp, err := c.AuthRep(apiCall)
		if err != nil {
			if !fixture.expectErr {
				t.Error("unexpected error")
			}
			// we expected an error so ensure our err conditions are met
			if !strings.Contains(err.Error(), fixture.expectErrMsg) {
				t.Errorf("expected our error message to contain substring %s", fixture.expectErrMsg)
			}
			return
		}
		equals(t, fixture.expectResponse, resp)
	}

}

func TestClient_Report(t *testing.T) {
	const svcID = "test-id"

	inputs := []struct {
		name           string
		auth           api.ClientAuth
		transactions   []api.Transaction
		expectErr      bool
		expectErrMsg   string
		expectResponse *ReportResponse
		client         *Client
		injectClient   *http.Client
		waitForCB      bool
	}{
		{
			name:         "Test expect failure bad url passed",
			auth:         api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transactions: []api.Transaction{{Params: api.Params{AppID: "any"}}},
			expectErr:    true,
			expectErrMsg: httpReqErrText,
			client: &Client{
				backendHost: "/some/invalid/value%_",
				baseURL:     "/some/invalid/value%_",
				httpClient:  http.DefaultClient,
			},
		},
		{
			name:         "Test expect failure simulated network error",
			auth:         api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transactions: []api.Transaction{{Params: api.Params{AppID: "any"}}},
			expectErr:    true,
			expectErrMsg: "Timeout exceeded",
			client: &Client{
				baseURL: defaultBackendUrl,
				httpClient: &http.Client{
					Timeout: time.Nanosecond,
				},
			},
		},
		{
			name: "Test expect failure simulated bad response from 3scale error",
			auth: api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transactions: []api.Transaction{
				{
					Params: api.Params{
						AppID: "any",
					},
				},
			},
			expectErr:    true,
			expectErrMsg: "EOF",
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       ioutil.NopCloser(bytes.NewBufferString("EOF")),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test expect failure 403",
			auth: api.ClientAuth{Type: api.ProviderKey, Value: "any"},
			transactions: []api.Transaction{
				{
					Params: api.Params{
						UserKey: "any",
					},
				},
			},
			expectResponse: &ReportResponse{
				accepted:   false,
				Reason:     "user_key_invalid",
				StatusCode: http.StatusForbidden,
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GenInvalidUserKey("any"))),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test params formatting",
			auth: api.ClientAuth{
				Type:  api.ServiceToken,
				Value: "st",
			},
			transactions: []api.Transaction{
				{
					Params: api.Params{
						UserKey: "test",
					},
					Metrics: api.Metrics{"hits": 1},
				},
				{
					Params: api.Params{
						UserKey: "test-2",
					},
					Metrics: api.Metrics{"hits": 1, "other": 2},
				},
			},
			expectResponse: &ReportResponse{
				accepted:   true,
				StatusCode: http.StatusAccepted,
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				// we know that Encode will sort by keys so we can predict this output
				// decoded to service_id=test-id&service_token=st&transactions[0][usage][hits]=1&transactions[0][user_key]=test&transactions[1][usage][hits]=1&transactions[1][usage][other]=2&transactions[1][user_key]=test-2
				expect := `service_id=test-id&service_token=st&transactions%5B0%5D%5Busage%5D%5Bhits%5D=1&transactions%5B0%5D%5Buser_key%5D=test&transactions%5B1%5D%5Busage%5D%5Bhits%5D=1&transactions%5B1%5D%5Busage%5D%5Bother%5D=2&transactions%5B1%5D%5Buser_key%5D=test-2`
				equals(t, expect, req.URL.RawQuery)

				return &http.Response{
					StatusCode: 202,
					Body:       ioutil.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}
			}),
		},
	}

	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			if input.injectClient == nil {
				// fallback client
				input.injectClient = NewTestClient(func(req *http.Request) *http.Response {
					equals(t, req.Method, http.MethodPost)
					equals(t, req.URL.Path, reportEndpoint)
					return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess()))}
				})
			}

			c := input.client
			if c == nil {
				c = threeScaleTestClient(t, input.injectClient)
			}

			apiCall := threescale.Request{
				Auth:         input.auth,
				Service:      svcID,
				Transactions: input.transactions,
			}

			resp, err := c.Report(apiCall)
			if err != nil {
				if !input.expectErr {
					t.Error("unexpected error")
				}
				// we expected an error so ensure our err conditions are met
				if !strings.Contains(err.Error(), input.expectErrMsg) {
					t.Errorf("expected our error message to contain substring %s", input.expectErrMsg)
				}
				return
			}
			equals(t, input.expectResponse, resp)
			equals(t, input.expectResponse.accepted, resp.Accepted())
		})
	}
}

func TestNewClient(t *testing.T) {
	_, err := NewClient("ftp://invalid.com", http.DefaultClient)
	if err == nil {
		t.Error("expected error for invalid scheme")
	}

	c, err := NewClient(defaultBackendUrl, http.DefaultClient)
	if err != nil {
		t.Error("unexpected error when creating client")
	}

	if c.GetPeer() != "su1.3scale.net" {
		t.Error("unexpected hostname set via constructor")
	}
}

func TestNewDefaultClient(t *testing.T) {
	c, _ := NewDefaultClient()

	if c.baseURL != defaultBackendUrl {
		t.Error("unexpected setting in default client")
	}

	if c.httpClient.Timeout != defaultTimeout {
		t.Error("unexpected setting in default client")
	}
}

// ******
// Helpers

// equals fails the test if exp is not equal to act.
func equals(t *testing.T, exp, act interface{}) {
	t.Helper()
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		t.Error("unexpected result when calling equals")
	}
}

// Returns a default client for testing
func threeScaleTestClient(t *testing.T, hc *http.Client) *Client {
	client, err := NewClient(defaultBackendUrl, hc)
	if err != nil {
		t.Error("failed to build test client")
	}
	return client
}

func getUsageReportXML(t *testing.T) string {
	t.Helper()
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
   <authorized>true</authorized>
   <plan>Basic</plan>
   <usage_reports>
      <usage_report metric="hits" period="minute">
         <period_start>2019-02-22 14:32:00 +0000</period_start>
         <period_end>2019-02-22 14:33:00 +0000</period_end>
         <max_value>4</max_value>
         <current_value>1</current_value>
      </usage_report>
      <usage_report metric="test_metric" period="week">
         <period_start>2019-02-18 00:00:00 +0000</period_start>
         <period_end>2019-02-25 00:00:00 +0000</period_end>
         <max_value>6</max_value>
         <current_value>0</current_value>
      </usage_report>
   </usage_reports>
</status>`
}

func getHierarchyXML(t *testing.T) string {
	t.Helper()
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
   <authorized>true</authorized>
   <plan>Basic</plan>
   <hierarchy>
      <metric name="hits" children="example sample test test" />
      <metric name="test_metric" children="" />
   </hierarchy>
</status>`
}

var extTested bool

func getExtensions(t *testing.T) map[string]string {
	t.Helper()

	// ensure we at least return the extensions the first time we get called
	if !extTested || rand.Intn(2) != 0 {
		extTested = true
		return map[string]string{
			"no_body":       "1",
			"asingle;field": "and;single;value",
			"many@@and==":   "should@@befine==",
			"a test&":       "&ok",
		}
	}
	return nil
}

// returns a randomly-ordered list of strings for extensions with format "key=value"
func getExtensionsValue(t *testing.T) []string {
	t.Helper()
	expected := map[string]string{
		"no_body":             "1",
		"asingle%3Bfield":     "and%3Bsingle%3Bvalue",
		"many%40%40and%3D%3D": "should%40%40befine%3D%3D",
		"a+test%26":           "%26ok",
	}

	exp := make([]string, 0, unsafe.Sizeof(expected))
	// golang's iteration over maps randomizes order of kv's
	for k, v := range expected {
		exp = append(exp, fmt.Sprintf("%s=%s", k, v))
	}

	return exp
}

func checkExtensions(t *testing.T, req *http.Request) (bool, string) {
	t.Helper()

	value := req.Header.Get("3scale-options")
	expected := getExtensionsValue(t)

	found := strings.Split(value, "&")

	if compareUnorderedStringLists(found, expected) {
		return true, ""
	}
	sort.Strings(expected)
	sort.Strings(found)

	return false, fmt.Sprintf("\nexpected extension header value %s\n"+
		"                      but found %s",
		strings.Join(expected, ", "), strings.Join(found, ", "))

}

func compareUnorderedStringLists(one []string, other []string) bool {
	if len(one) != len(other) {
		return false
	}

	for _, x := range one {
		found := false

		for _, y := range other {
			if x == y {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// ******

// *****
// Mocking objects for HTTP tests
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// Get a test client with transport overridden for mocking
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

// ******