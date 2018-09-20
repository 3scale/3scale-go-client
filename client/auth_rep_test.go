package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
)

func TestAuthRep(t *testing.T) {
	fakeAppId, fakeServiceToken, fakeServiceId := "appId12345", "servicetoken54321", "555000"
	authRepInputs := []struct {
		appId, svcToken, svcId string
		expectErr              bool
		expectSuccess          bool
		expectReason           string
		expectStatus           int
		expectParamLength      int
		buildParams            func() AuthRepParams
	}{
		{
			appId:             fakeAppId,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 4,
			buildParams:       func() AuthRepParams { return NewAuthRepParams("example", "", "") },
		},
		{
			appId:             "failme",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectErr:         true,
			expectSuccess:     false,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthRepParams { return AuthRepParams{} },
		},
	}
	for _, input := range authRepInputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, authRepEndpoint)

			params := req.URL.Query()
			if input.expectParamLength != len(params) {
				t.Fatalf("unexpected param length, expect %d got  %d", input.expectParamLength, len(params))
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
				Body:       ioutil.NopCloser(bytes.NewBufferString(getAuthSuccess())),
				Header:     make(http.Header),
			}
		})
		c := threeScaleTestClient(httpClient)
		resp, err := c.AuthRep(input.appId, input.svcToken, input.svcId, input.buildParams())
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

func TestAuthRepKey(t *testing.T) {
	fakeUserKey, fakeServiceToken, fakeServiceId := "userkey12345", "servicetoken54321", "555000"
	fakeMetricKey := "usage[hits]"
	authRepInputs := []struct {
		userKey, svcToken, svcId string
		expectErr                bool
		expectSuccess            bool
		expectReason             string
		expectStatus             int
		expectParamLength        int
		buildParams              func() AuthRepKeyParams
	}{
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthRepKeyParams { return AuthRepKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthRepKeyParams { return AuthRepKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          "invalid",
			expectReason:      "service_token_invalid",
			expectSuccess:     false,
			expectStatus:      403,
			expectParamLength: 3,
			buildParams:       func() AuthRepKeyParams { return AuthRepKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             "invalid",
			svcToken:          fakeServiceToken,
			expectReason:      "service_token_invalid",
			expectSuccess:     false,
			expectStatus:      403,
			expectParamLength: 3,
			buildParams:       func() AuthRepKeyParams { return AuthRepKeyParams{} },
		},
		{
			userKey:           "invalid",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectReason:      "user_key_invalid",
			expectSuccess:     false,
			expectStatus:      403,
			expectParamLength: 3,
			buildParams:       func() AuthRepKeyParams { return AuthRepKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 4,
			buildParams: func() AuthRepKeyParams {
				params := NewAuthRepKeyParams("", "")
				params.Metrics.Add("hits", 5)
				return params
			},
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     false,
			expectStatus:      409,
			expectParamLength: 4,
			expectReason:      "usage limits are exceeded",
			buildParams: func() AuthRepKeyParams {
				params := NewAuthRepKeyParams("", "")
				params.Metrics.Add("hits", 6)
				return params
			},
		},
		{
			userKey:           "failme",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectErr:         true,
			expectSuccess:     false,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthRepKeyParams { return AuthRepKeyParams{} },
		},
		{
			userKey:           fakeUserKey,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 9,
			buildParams: func() AuthRepKeyParams {
				params := NewAuthRepKeyParams("testR", "testUid")
				params.Metrics.Add("hits", 5)
				params.Log.Set("testlog", "testresp", 200)
				return params
			},
		},
	}
	for _, input := range authRepInputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, authRepEndpoint)
			r := regexp.MustCompile(`^usage\[\S*\]$`)

			params := req.URL.Query()
			if input.expectParamLength != len(params) {
				t.Fatalf("unexpected param length, expect %d got  %d", input.expectParamLength, len(params))
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
					Body:       ioutil.NopCloser(bytes.NewBufferString(genInvalidIdOrTokenResp(queryToken, queryId))),
					Header:     make(http.Header),
				}
			}

			if queryUserKey != fakeUserKey {
				return &http.Response{
					StatusCode: 403,
					Body:       ioutil.NopCloser(bytes.NewBufferString(genInvalidUserKey(queryUserKey))),
					Header:     make(http.Header),
				}
			}

			for k, v := range params {
				if r.MatchString(k) {
					if k != fakeMetricKey {
						return &http.Response{
							StatusCode: 409,
							Body:       ioutil.NopCloser(bytes.NewBufferString(getInvalidMetricResp())),
							Header:     make(http.Header),
						}
					}
					if k == fakeMetricKey && v[0] == "6" {
						return &http.Response{
							StatusCode: 409,
							Body:       ioutil.NopCloser(bytes.NewBufferString(getLimitExceededResp())),
							Header:     make(http.Header),
						}
					}
				}

			}
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(getAuthSuccess())),
				Header:     make(http.Header),
			}
		})

		c := threeScaleTestClient(httpClient)
		resp, err := c.AuthRepKey(input.userKey, input.svcToken, input.svcId, input.buildParams())
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
