package threescale

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"
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

	httpReqErrText = "error building http transaction"

	// a parsable time format used to convert Ruby time to time type
	timeLayout = "2006-01-02 15:04:05 -0700"
)

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

// GetHierarchy returns a list of children (methods) associated with a parent(metric)
func (r *AuthorizeResponse) GetHierarchy() Hierarchy {
	return r.hierarchy
}

// GetUsageReports returns a list of usage reports - list will be empty if no limits set
func (r *AuthorizeResponse) GetUsageReports() UsageReports {
	return r.usageReports
}

func (ca ClientAuth) joinToValues(values url.Values) url.Values {
	values.Add(string(ca.Type), ca.Value)
	return values
}

// DeepCopy returns a clone of the original Metrics. It provides a deep copy
// of both the key and the value of the original Hierarchy.
func (h Hierarchy) DeepCopy() Hierarchy {
	clone := make(Hierarchy, len(h))
	for k, v := range h {
		var clonedV []string
		clonedV = append(clonedV, v...)
		clone[k] = clonedV
	}
	return clone
}

// Add takes a provided key and value and adds them to the Metric 'm'
// If the metric already existed in 'm', then the value will be added (if positive) or subtracted (if negative) from the existing value.
// If a subtraction leads to a negative value Add returns an error  and the change will be discarded.
// Returns the updated value (or current value in error cases) as well as the error.
func (m Metrics) Add(name string, value int) (int, error) {
	if currentValue, ok := m[name]; ok {
		newValue := currentValue + value
		if newValue < 0 {
			return currentValue, fmt.Errorf("invalid value for metric %s post computation. this will result in 403 from 3scale", name)
		}
		m[name] = newValue
		return newValue, nil
	}
	m[name] = value
	return value, nil
}

// Set takes a provided key and value and sets that value of the key in 'm', overwriting any value that exists previously.
func (m Metrics) Set(name string, value int) error {
	if value < 0 {
		return fmt.Errorf("invalid value for metric %s post computation. this will result in 403 from 3scale", name)
	}
	m[name] = value
	return nil
}

// Delete a metric m['name'] if present
func (m Metrics) Delete(name string) {
	delete(m, name)
}

// DeepCopy returns a clone of the original Metrics
func (m Metrics) DeepCopy() Metrics {
	clone := make(Metrics, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// adds the metrics and their associated values to the provided url.Values - converting them as required in the process
func (m Metrics) joinToValues(values url.Values) url.Values {
	// metrics must be converted and formatted correctly for 3scale backend
	converted := m.convert()
	for k, v := range converted {
		values.Add(k, v)
	}
	return values
}

// Converts a Metrics type into formatted map as expected by 3scale API for Auth and AuthRep
func (m Metrics) convert() map[string]string {
	formatted := make(map[string]string, len(m))
	for k, v := range m {
		formatted[fmt.Sprintf("usage[%s]", k)] = strconv.Itoa(v)
	}
	return formatted
}

// joinToValues inspects the Params receiver for non-empty and values with json tags and appends them to the provided url.Values
func (p Params) joinToValues(values url.Values) url.Values {
	val := reflect.ValueOf(p)
	for i := 0; i < val.Type().NumField(); i++ {
		if tag, ok := val.Type().Field(i).Tag.Lookup("json"); ok {
			if valueToAdd := val.Field(i).String(); valueToAdd != "" {
				values.Add(tag, valueToAdd)
			}
		}
	}
	return values
}

// GetLimitRemaining returns an integer stating the amount of hits left for the full combination of metrics authorized in this call
// before the rate limiting logic would start denying authorizations for the current period.
// A value of -1 indicates there is no limit in the amount of hits.
// Nil value will indicate the extension has not been used.
func (r *RateLimits) GetLimitRemaining() int {
	return r.limitRemaining
}

// GetLimitReset returns ann integer stating the amount of seconds left for the current limiting period to elapse.
// A value of -1 indicates there i is no limit in time.
// Nil value will indicate the extension has not been used.
func (r *RateLimits) GetLimitReset() int {
	return r.limitReset
}

// because the report endpoint takes batches, we must alter the typical formatting here and customise it slightly to
// support reporting batches of transactions
func (r *Transaction) convertAndAddToTransactionValues(reportValues url.Values, index int, transaction Transaction) url.Values {
	paramValues := transaction.Params.joinToValues(make(url.Values))
	for k, v := range paramValues {
		reportValues.Add(fmt.Sprintf("transactions[%d][%s]", index, k), v[0])
	}

	for k, v := range transaction.Metrics {
		reportValues.Add(fmt.Sprintf("transactions[%d][usage][%s]", index, k), strconv.Itoa(v))
	}

	return reportValues
}

// convert an xml decoded response into a user friendly UsageReport
func (ur UsageReportXML) convert() (UsageReport, error) {
	var err error
	report := UsageReport{
		Period:       ur.Period,
		MaxValue:     ur.MaxValue,
		CurrentValue: ur.CurrentValue,
	}

	if t, err := time.Parse(timeLayout, ur.PeriodStart); err != nil {
		return report, err
	} else {
		report.PeriodStart = t.Unix()
	}

	if t, err := time.Parse(timeLayout, ur.PeriodEnd); err != nil {
		return report, err
	} else {
		report.PeriodEnd = t.Unix()
	}
	return report, err
}

func contains(key string, in []string) bool {
	for _, i := range in {
		if key == i {
			return true
		}
	}
	return false
}

func defaultHttpClient() *http.Client {
	c := http.DefaultClient
	c.Timeout = defaultTimeout
	return c
}

func encodeExtensions(extensions Extensions) string {
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
