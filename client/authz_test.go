package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/3scale/3scale-go-client/fake"
)

func TestAuthorize(t *testing.T) {
	fakeAppId, fakeServiceToken, fakeServiceId := "appId12345", "servicetoken54321", "555000"
	authInputs := []struct {
		appId, svcToken, svcId string
		extensions             map[string]string
		expectErr              bool
		expectSuccess          bool
		expectReason           string
		expectStatus           int
		expectParamLength      int
		buildParams            func() AuthorizeParams
	}{
		{
			appId:             fakeAppId,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 4,
			buildParams:       func() AuthorizeParams { return NewAuthorizeParams("example", "", "") },
		},
		{
			appId:             "failme",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectErr:         true,
			expectSuccess:     false,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthorizeParams { return AuthorizeParams{} },
		},
		{
			appId:             fakeAppId,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 6,
			buildParams: func() AuthorizeParams {
				p := NewAuthorizeParams("example", "test", "")
				p.Metrics.Add("hits", 1)
				return p
			},
		},
	}
	for _, input := range authInputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, authzEndpoint)

			params := req.URL.Query()
			if input.expectParamLength != len(params) {
				t.Fatalf("unexpected param length, expect %d got  %d", input.expectParamLength, len(params))
			}

			if input.extensions != nil {
				if ok, err := checkExtensions(t, req); !ok {
					t.Fatal(err)
				}
			}

			queryAppId := params["app_id"][0]

			if queryAppId == "failme" {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("Some invalid xml")),
					Header:     make(http.Header),
				}
			}

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
				Header:     make(http.Header),
			}
		})
		c := threeScaleTestClient(httpClient)
		resp, err := c.AuthorizeAppID(input.appId, input.svcToken, input.svcId, input.buildParams(), input.extensions)
		if input.expectErr && err != nil {
			continue
		}

		if err != nil {
			t.Fatal(err.Error())
		}
		if input.expectSuccess != resp.Success {
			t.Fatalf("unexpected auth response returned")
		}
		if input.expectStatus != resp.StatusCode {
			t.Fatalf("unexpected status code")
		}
		if !input.expectSuccess {
			if input.expectReason != resp.Reason {
				t.Fatalf("unexpected xml parsing")
			}
		}
	}
}

func TestAuthorizeKey(t *testing.T) {
	fakeUserKey, fakeServiceToken, fakeServiceId := "userkey12345", "servicetoken54321", "555000"
	fakeMetricKey := "usage[hits]"
	authRepInputs := []struct {
		userKey, svcToken, svcId string
		extensions               map[string]string
		expectErr                bool
		expectSuccess            bool
		expectReason             string
		expectStatus             int
		expectParamLength        int
		buildParams              func() AuthorizeKeyParams
	}{
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthorizeKeyParams { return AuthorizeKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthorizeKeyParams { return AuthorizeKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          "invalid",
			extensions:        getExtensions(t),
			expectReason:      "service_token_invalid",
			expectSuccess:     false,
			expectStatus:      403,
			expectParamLength: 3,
			buildParams:       func() AuthorizeKeyParams { return AuthorizeKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             "invalid",
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectReason:      "service_token_invalid",
			expectSuccess:     false,
			expectStatus:      403,
			expectParamLength: 3,
			buildParams:       func() AuthorizeKeyParams { return AuthorizeKeyParams{} },
		},
		{
			userKey:           "invalid",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectReason:      "user_key_invalid",
			expectSuccess:     false,
			expectStatus:      403,
			expectParamLength: 3,
			buildParams:       func() AuthorizeKeyParams { return AuthorizeKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 4,
			buildParams: func() AuthorizeKeyParams {
				params := NewAuthorizeKeyParams("", "")
				params.Metrics.Add("hits", 5)
				return params
			},
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     false,
			expectStatus:      409,
			expectParamLength: 4,
			expectReason:      "usage limits are exceeded",
			buildParams: func() AuthorizeKeyParams {
				params := NewAuthorizeKeyParams("", "")
				params.Metrics.Add("hits", 6)
				return params
			},
		},
		{
			userKey:           "failme",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectErr:         true,
			expectSuccess:     false,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthorizeKeyParams { return AuthorizeKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			extensions:        getExtensions(t),
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 6,
			buildParams: func() AuthorizeKeyParams {
				params := NewAuthorizeKeyParams("testR", "testUid")
				params.Metrics.Add("hits", 5)
				return params
			},
		},
	}
	for _, input := range authRepInputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, authzEndpoint)
			r := regexp.MustCompile(`^usage\[\S*\]$`)

			params := req.URL.Query()
			if input.expectParamLength != len(params) {
				t.Fatalf("unexpected param length, expect %d got  %d", input.expectParamLength, len(params))
			}

			if input.extensions != nil {
				if ok, err := checkExtensions(t, req); !ok {
					t.Fatal(err)
				}
			}

			queryUserKey := params["user_key"][0]
			queryToken := params["service_token"][0]
			queryId := params["service_id"][0]

			if queryUserKey == "failme" {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("Some invalid xml")),
					Header:     make(http.Header),
				}
			}

			if queryId != fakeServiceId || queryToken != fakeServiceToken {
				return &http.Response{
					StatusCode: 403,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GenInvalidIdOrTokenResp(queryToken, queryId))),
					Header:     make(http.Header),
				}
			}

			if queryUserKey != fakeUserKey {
				return &http.Response{
					StatusCode: 403,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GenInvalidUserKey(queryUserKey))),
					Header:     make(http.Header),
				}
			}

			for k, v := range params {
				if r.MatchString(k) {
					if k != fakeMetricKey {
						return &http.Response{
							StatusCode: 409,
							Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetInvalidMetricResp())),
							Header:     make(http.Header),
						}
					}
					if k == fakeMetricKey && v[0] == "6" {
						return &http.Response{
							StatusCode: 409,
							Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetLimitExceededResp())),
							Header:     make(http.Header),
						}
					}
				}

			}
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
				Header:     make(http.Header),
			}
		})

		c := threeScaleTestClient(httpClient)
		resp, err := c.AuthorizeKey(input.userKey, input.svcToken, input.svcId, input.buildParams(), input.extensions)
		if input.expectErr && err != nil {
			continue
		}

		if err != nil {
			t.Fatal(err.Error())
		}
		if input.expectSuccess != resp.Success {
			t.Fatalf("unexpected auth response returned")
		}
		if input.expectStatus != resp.StatusCode {
			t.Fatalf("unexpected status code")
		}
		if !input.expectSuccess {
			if input.expectReason != resp.Reason {
				t.Fatalf("unexpected xml parsing")
			}
		}
	}
}
