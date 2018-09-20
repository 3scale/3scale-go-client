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