package client

import (
	"errors"
	"fmt"
	"net/url"
)

const reportEndpoint = "/transactions.xml"

// Report - Wrapper function to allow the client to determine, by parsing the provided data, what 3scale API (reporting) should be called.
// Note if both application types are provided then user_key authentication is prioritised.
func (client *ThreeScaleClient) Report(req Request, serviceId string, transactions ReportTransactions, extensions map[string]string) (ApiResponse, error) {
	if req.Application.UserKey != "" {
		return client.ReportUserKey(req.Credentials, serviceId, transactions, extensions)
	}


	return client.ReportAppID(req.Credentials, serviceId, transactions, extensions)
}

//ReportAppID - Report for the Application Id authentication pattern with serviceToken
func (client *ThreeScaleClient) ReportAppID(auth TokenAuth, serviceId string, transactions ReportTransactions, extensions map[string]string) (ApiResponse, error) {
	values := parseQueries(transactions, url.Values{}, transactions.Metrics, transactions.Log)

	err := auth.SetURLValues(&values)
	if err != nil {
		return ApiResponse{}, err
	}

	values.Add("service_id", serviceId)

	return client.report(values, extensions)
}

//ReportUserKey - Report for the API Key authentication pattern with service token
func (client *ThreeScaleClient) ReportUserKey(auth TokenAuth, serviceId string, transactions ReportTransactions, extensions map[string]string) (ApiResponse, error) {
	values := parseQueries(transactions, url.Values{}, transactions.Metrics, transactions.Log)

	err := auth.SetURLValues(&values)
	if err != nil {
		return ApiResponse{}, err
	}

	values.Add("service_id", serviceId)
	return client.report(values, extensions)
}

func (client *ThreeScaleClient) report(values url.Values, extensions map[string]string) (ApiResponse, error) {
	var resp ApiResponse

	req, err := client.buildGetReq(reportEndpoint, extensions)
	if err != nil {
		return resp, errors.New(httpReqError.Error() + " for report")
	}

	req.URL.RawQuery = values.Encode()
	resp, err = client.doHttpReq(req, extensions)
	if err != nil {
		return resp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return resp, nil
}

func NewTransactionAppID(AppID string, Timestamp string, UserId string, metrics Metrics, log Log) ReportTransactions {
	return ReportTransactions{
		AppID:     AppID,
		UserKey:   "",
		UserId:    UserId,
		Timestamp: Timestamp,
		Metrics:   metrics,
		Log:       log,
	}
}

func NewTransactionUserKey(UserKey string, Timestamp string, UserId string, metrics Metrics, log Log) ReportTransactions {
	return ReportTransactions{
		AppID:     "",
		UserKey:   UserKey,
		UserId:    UserId,
		Timestamp: Timestamp,
		Metrics:   metrics,
		Log:       log,
	}
}
