package fake

import "fmt"

// Get default success response for authorize endpoint
func GetAuthSuccess() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
  <authorized>true</authorized>
  <plan>Basic</plan>
</status>`
}

// Get mock response for invalid service token or id
func GenInvalidIdOrTokenResp(token string, id string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<error code="service_token_invalid">service token "%s" or service id "%s" is invalid</error>`, token, id)
}

// Get mock response for invalid metric
func GetInvalidMetricResp() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<error code="metric_invalid">metric "anyButHits" is invalid</error>`
}

// Get mock response for invalid user key
func GenInvalidUserKey(key string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<error code="user_key_invalid">user key "%s" is invalid</error>`, key)
}

// Get mock response for limit exceeded
func GetLimitExceededResp() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
  <authorized>false</authorized>
  <reason>usage limits are exceeded</reason>
  <plan>Basic</plan>
  <usage_reports>
    <usage_report metric="hits" period="minute">
      <period_start>2018-09-01 14:44:00 +0000</period_start>
      <period_end>2018-09-01 14:45:00 +0000</period_end>
      <max_value>1</max_value>
      <current_value>1</current_value>
    </usage_report>
  </usage_reports>
</status>`
}
