package client

//AuthRep is a 'one-shot' operation to authorize an application and report the associated transaction at the same time.
//The main difference between this call and the regular authorize call is that usage will be reported if the authorization is successful.
//AuthRep is the most convenient way to integrate your API with the 3scale's Service Management API.
//It does a 1:1 mapping between a request to your API, and a request to 3scale's API.
//AuthRep is not a read-only operation and will increment the values if the authorization step is a success.

import (
	"errors"
	"fmt"
	"net/url"
)

const authRepEndpoint = "/transactions/authrep.xml"

//AuthRep - Authorize & Report for the Application Id authentication pattern
func (client *ThreeScaleClient) AuthRepAppID(auth TokenAuth, appId string, serviceId string, params AuthRepParams, extensions map[string]string) (ApiResponse, error) {
	values := parseQueries(params, url.Values{}, params.Metrics, params.Log)
	values.Add("app_id", appId)
	values.Add("service_id", serviceId)

	err := auth.SetURLValues(&values)
	if err != nil {
		return ApiResponse{}, err
	}

	return client.authRep(values, extensions)
}

//AuthRepKey - Authorize & Report for the API Key authentication pattern with service token
func (client *ThreeScaleClient) AuthRepUserKey(auth TokenAuth, userKey string, serviceId string, params AuthRepParams, extensions map[string]string) (ApiResponse, error) {
	values := parseQueries(params, url.Values{}, params.Metrics, params.Log)
	values.Add("user_key", userKey)
	values.Add("service_id", serviceId)

	err := auth.SetURLValues(&values)
	if err != nil {
		return ApiResponse{}, err
	}

	return client.authRep(values, extensions)
}

func (client *ThreeScaleClient) authRep(values url.Values, extensions map[string]string) (ApiResponse, error) {
	var resp ApiResponse

	req, err := client.buildGetReq(authRepEndpoint, extensions)
	if err != nil {
		return resp, errors.New(httpReqError.Error() + " for AuthRep")
	}

	req.URL.RawQuery = values.Encode()
	resp, err = client.doHttpReq(req)
	if err != nil {
		return resp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return resp, nil
}

func NewAuthRepParamsAppID(key string, referrer string, userId string, metrics Metrics, log Log) AuthRepParams {
	return AuthRepParams{
		AuthorizeParams: AuthorizeParams{
			AppKey:   key,
			Referrer: referrer,
			UserId:   userId,
			Metrics:  metrics,
		},
		Log: log,
	}
}

func NewAuthRepParamsUserKey(referrer string, userId string, metrics Metrics, log Log) AuthRepParams {
	return AuthRepParams{
		AuthorizeParams: AuthorizeParams{
			AppKey:   "",
			Referrer: referrer,
			UserId:   userId,
			Metrics:  metrics,
		},
		Log: log,
	}
}
