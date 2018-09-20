## 3Scale Service Management API Client

**Note: This project is in _alpha_ and is currently unsupported.**

### Creating a client

A 3scale client can be configured with the default backend, which points to `https://su1.3scale.net:443`. Alternatively, a custom backend can be configured.
The 3scale client can be provided with a custom [HTTP client](https://golang.org/pkg/net/http/?#Client). If none is provided (not recommended), the default client will be used.

A client can be created for the default 3scale backend, which points to `"https://su1.3scale.net:443`

```go
	client := NewThreeScale(DefaultBackend(), &http.Client{
	        Timeout: 30 * time.Second,
        })

```

### Calling the Authorize endpoint

The client supports calling the `Authorize` endpoint using both the **_Application ID_** and **_Application Key_** authentication patterns. Metrics can be added as well as optional
parameters for both API's.

```go
        // App Id Pattern - No optional params
	resp, err := c.Authorize("myAppId", "mySvcToken", "myServiceId", AuthorizeParams{})
	if err != nil {
		// Handle error
	}
	if !resp.Success {
		fmt.Printf("request failed - reason: %s", resp.Reason)
	}
	
	// App Id pattern with optional params and metrics
	p := NewAuthorizeParams("myAppKey", "exampleReferrer", "")
	p.Metrics.Add("hits", 1)
	resp, _ = c.Authorize("myAppId", "mySvcToken", "myServiceId", p)
	
	// App key pattern with optional params
	p = NewAuthorizeKeyParams("exampleRef", "exampleId")
	resp, _ = c.AuthorizeKey("userKey", "svcToken", "svcID", p)
	
```

### Calling the AuthRep endpoint

The client supports calling the `AuthRep` endpoint using both the **_Application ID_** and **_Application Key_** authentication patterns. Metrics and Log can be added as well as optional
parameters for both API's.

```go
	client := NewThreeScale(nil, nil)
	// App Id Pattern - No optional params
	resp, err := client.AuthRep("myAppId", "mySvcToken", "myServiceId", AuthRepParams{})
	if err != nil {
		// Handle error
	}
	if !resp.Success {
		fmt.Printf("request failed - reason: %s", resp.Reason)
	}

	// App Id pattern with optional params and metrics
	p := NewAuthRepParams("myAppKey", "exampleReferrer", "")
	p.Metrics.Add("hits", 1)
	p.Log.Set("exampleReq", "exampleResp", 200)
	resp, _ = client.AuthRep("myAppId", "mySvcToken", "myServiceId", p)

	// App key pattern with optional params
	pms := NewAuthRepKeyParams("exampleRef", "exampleId")
	resp, _ = client.AuthRepKey("userKey", "svcToken", "svcID", pms)
	
```