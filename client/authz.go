package client

import (
	"errors"
	"fmt"
	"net/url"
)

const authzEndpoint = "/transactions/authorize.xml"

// Authorize - Wrapper function to allow the client to determine, by parsing the provided data, what 3scale API (auth) should be called.
// Note if both application types are provided then user_key authentication is prioritised.
func (client *ThreeScaleClient) Authorize(req Request, serviceId string, metrics Metrics, extensions map[string]string) (ApiResponse, error) {
	if req.Application.UserKey != "" {
		return client.AuthorizeKey(req.Application.UserKey, req.Credentials.Value, serviceId, AuthorizeKeyParams{Metrics:  metrics}, extensions)
	}

	return client.AuthorizeAppID(req.Application.AppID.ID, req.Credentials.Value, serviceId, AuthorizeParams{Metrics:  metrics}, extensions)
}


//AuthorizeAppID - Read-only operation to authorize an application in the App Id authentication pattern.
func (client *ThreeScaleClient) AuthorizeAppID(appId string, serviceToken string, serviceId string, arp AuthorizeParams, extensions map[string]string) (ApiResponse, error) {
	var authRepResp ApiResponse

	req, err := client.buildGetReq(authzEndpoint, extensions)
	if err != nil {
		return authRepResp, errors.New(httpReqError.Error() + " for AuthorizeAppID")
	}

	values := parseQueries(arp, url.Values{}, arp.Metrics, nil)
	values.Add("app_id", appId)
	values.Add("service_token", serviceToken)
	values.Add("service_id", serviceId)

	req.URL.RawQuery = values.Encode()
	authRepRes, err := client.doHttpReq(req, extensions)
	if err != nil {
		return authRepResp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return authRepRes, nil
}

//AuthorizeKey -  Read-only operation to authorize an application for the API Key authentication pattern
func (client *ThreeScaleClient) AuthorizeKey(userKey string, serviceToken string, serviceId string, arp AuthorizeKeyParams, extensions map[string]string) (ApiResponse, error) {
	var resp ApiResponse

	req, err := client.buildGetReq(authzEndpoint, extensions)
	if err != nil {
		return resp, errors.New(httpReqError.Error() + " for AuthorizeKey")
	}

	values := parseQueries(arp, url.Values{}, arp.Metrics, nil)
	values.Add("user_key", userKey)
	values.Add("service_token", serviceToken)
	values.Add("service_id", serviceId)

	req.URL.RawQuery = values.Encode()
	resp, err = client.doHttpReq(req, extensions)
	if err != nil {
		return resp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return resp, nil
}

// Create valid AuthorizeParams
func NewAuthorizeParams(appKey string, referrer string, userId string) AuthorizeParams {
	return AuthorizeParams{
		AppKey:   appKey,
		Referrer: referrer,
		UserId:   userId,
		Metrics:  make(Metrics),
	}
}

// Create valid AuthorizeKeyParams
func NewAuthorizeKeyParams(referrer string, userId string) AuthorizeKeyParams {
	return AuthorizeKeyParams{
		Referrer: referrer,
		UserId:   userId,
		Metrics:  make(Metrics),
	}
}
