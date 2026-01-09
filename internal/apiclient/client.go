package restapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

type APIClientOpt struct {
	URI                 string
	Insecure            bool
	Username            string
	Password            string
	Headers             map[string]string
	Timeout             int64 // Timeout in seconds for HTTP requests
	IDAttribute         string
	CreateMethod        string
	ReadMethod          string
	ReadData            string
	UpdateMethod        string
	UpdateData          string
	DestroyMethod       string
	DestroyData         string
	CopyKeys            []string
	WriteReturnsObject  bool
	CreateReturnsObject bool
	XSSIPrefix          string
	UseCookies          bool
	RateLimit           float64 // RateLimit in requests per second (0 = unlimited)
	OAuthClientID       string
	OAuthClientSecret   string
	OAuthScopes         []string
	OAuthTokenURL       string
	OAuthEndpointParams url.Values
	CertFile            string
	KeyFile             string
	RootCAFile          string
	CertString          string
	KeyString           string
	RootCAString        string
	Debug               bool
}

// APIClient is a HTTP client with additional controlling fields
type APIClient struct {
	httpClient          *http.Client
	uri                 string
	insecure            bool
	username            string
	password            string
	headers             map[string]string
	idAttribute         string
	createMethod        string
	readMethod          string
	readData            string
	updateMethod        string
	updateData          string
	destroyMethod       string
	destroyData         string
	copyKeys            []string
	writeReturnsObject  bool
	createReturnsObject bool
	xssiPrefix          string
	rateLimiter         *rate.Limiter
	debug               bool
	oauthConfig         *clientcredentials.Config
	Opts                APIClientOpt
}

// NewAPIClient makes a new api client for RESTful calls
func NewAPIClient(opt *APIClientOpt) (*APIClient, error) {
	ctx := context.Background()
	tflog.Debug(ctx, "Constructing API client", map[string]interface{}{"debug": opt.Debug})

	if opt.URI == "" {
		return nil, errors.New("uri must be set to construct an API client")
	}

	// Sane default
	if opt.IDAttribute == "" {
		opt.IDAttribute = "id"
	}

	// Remove any trailing slashes since we will append
	// to this URL with our own root-prefixed location
	opt.URI = strings.TrimSuffix(opt.URI, "/")

	if opt.CreateMethod == "" {
		opt.CreateMethod = "POST"
	}
	if opt.ReadMethod == "" {
		opt.ReadMethod = "GET"
	}
	if opt.UpdateMethod == "" {
		opt.UpdateMethod = "PUT"
	}
	if opt.DestroyMethod == "" {
		opt.DestroyMethod = "DELETE"
	}

	tlsConfig := &tls.Config{
		// Disable TLS verification if requested
		InsecureSkipVerify: opt.Insecure,
	}

	if opt.CertString != "" && opt.KeyString != "" {
		cert, err := tls.X509KeyPair([]byte(opt.CertString), []byte(opt.KeyString))
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if opt.CertFile != "" && opt.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load root CA
	if opt.RootCAFile != "" || opt.RootCAString != "" {
		caCertPool := x509.NewCertPool()
		var rootCA []byte
		var err error

		if opt.RootCAFile != "" {
			tflog.Debug(ctx, "Reading root CA file", map[string]interface{}{"rootCAFile": opt.RootCAFile})
			rootCA, err = os.ReadFile(opt.RootCAFile)
			if err != nil {
				return nil, fmt.Errorf("could not read root CA file: %v", err)
			}
		} else {
			tflog.Debug(ctx, "Using provided root CA string")
			rootCA = []byte(opt.RootCAString)
		}

		if !caCertPool.AppendCertsFromPEM(rootCA) {
			return nil, errors.New("failed to append root CA certificate(s)")
		}
		tlsConfig.RootCAs = caCertPool
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	var cookieJar http.CookieJar

	if opt.UseCookies {
		cookieJar, _ = cookiejar.New(nil)
	}

	rateLimit := rate.Limit(opt.RateLimit)

	// Bucket size determines burst capacity - at minimum 1 request, otherwise rounded rate
	bucketSize := int(math.Max(math.Round(opt.RateLimit), 1))
	tflog.Info(ctx, "rate limit configured", map[string]interface{}{"rateLimit": opt.RateLimit, "bucketSize": bucketSize})
	rateLimiter := rate.NewLimiter(rateLimit, bucketSize)

	httpClient := cleanhttp.DefaultClient()
	httpClient.Timeout = time.Second * time.Duration(opt.Timeout)
	httpClient.Transport = tr
	httpClient.Jar = cookieJar

	client := APIClient{
		httpClient:          httpClient,
		rateLimiter:         rateLimiter,
		uri:                 opt.URI,
		insecure:            opt.Insecure,
		username:            opt.Username,
		password:            opt.Password,
		headers:             opt.Headers,
		idAttribute:         opt.IDAttribute,
		createMethod:        opt.CreateMethod,
		readMethod:          opt.ReadMethod,
		readData:            opt.ReadData,
		updateMethod:        opt.UpdateMethod,
		updateData:          opt.UpdateData,
		destroyMethod:       opt.DestroyMethod,
		destroyData:         opt.DestroyData,
		copyKeys:            opt.CopyKeys,
		writeReturnsObject:  opt.WriteReturnsObject,
		createReturnsObject: opt.CreateReturnsObject,
		xssiPrefix:          opt.XSSIPrefix,
		debug:               opt.Debug,
		Opts:                *opt,
	}

	if opt.OAuthClientID != "" && opt.OAuthClientSecret != "" && opt.OAuthTokenURL != "" {
		client.oauthConfig = &clientcredentials.Config{
			ClientID:       opt.OAuthClientID,
			ClientSecret:   opt.OAuthClientSecret,
			TokenURL:       opt.OAuthTokenURL,
			Scopes:         opt.OAuthScopes,
			EndpointParams: opt.OAuthEndpointParams,
		}
	}

	tflog.Debug(ctx, "Constructed client", map[string]interface{}{"details": client.String()})
	return &client, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (client *APIClient) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("uri: %s\n", client.uri))
	buffer.WriteString(fmt.Sprintf("insecure: %t\n", client.insecure))
	buffer.WriteString(fmt.Sprintf("username: %s\n", client.username))
	buffer.WriteString(fmt.Sprintf("password: %s\n", client.password))
	buffer.WriteString(fmt.Sprintf("id_attribute: %s\n", client.idAttribute))
	buffer.WriteString(fmt.Sprintf("write_returns_object: %t\n", client.writeReturnsObject))
	buffer.WriteString(fmt.Sprintf("create_returns_object: %t\n", client.createReturnsObject))
	buffer.WriteString("headers:\n")
	for k, v := range client.headers {
		buffer.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	buffer.WriteString("copy_keys:\n")
	for _, n := range client.copyKeys {
		buffer.WriteString(fmt.Sprintf("  %s", n))
	}
	return buffer.String()
}

// SendRequest is a helper function that handles sending/receiving and handling of HTTP data in and out.
func (client *APIClient) SendRequest(ctx context.Context, method string, path string, data string, forceDebug bool) (string, int, error) {
	fullURI := client.uri + path
	var req *http.Request
	var err error

	tflog.Debug(ctx, "Sending request", map[string]interface{}{"method": method, "path": path, "fullURI": fullURI, "data": data})

	buffer := bytes.NewBuffer([]byte(data))

	if data == "" {
		req, err = http.NewRequest(method, fullURI, nil)
	} else {
		req, err = http.NewRequest(method, fullURI, buffer)

		// Default of application/json, but allow headers array to overwrite later
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return "", 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Allow for tokens or other pre-created secrets
	if len(client.headers) > 0 {
		for n, v := range client.headers {
			req.Header.Set(n, v)
		}
	}

	if client.oauthConfig != nil {
		// Embed our configured HTTP client (with certs, proxy, etc.) into the OAuth token request context
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client.httpClient)
		tokenSource := client.oauthConfig.TokenSource(ctx)
		token, err := tokenSource.Token()
		if err != nil {
			return "", 0, err
		}
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}

	if client.username != "" && client.password != "" {
		// Basic auth is applied after OAuth (if configured). If both are set, OAuth takes precedence
		// as it was set on the Authorization header above
		req.SetBasicAuth(client.username, client.password)
	}

	if client.debug || forceDebug {
		fmt.Println("----- HTTP Request -----")
		if dump, err := httputil.DumpRequest(req, true); err == nil {
			fmt.Println(string(dump))
		}
	}

	if client.rateLimiter != nil {
		tflog.Debug(ctx, "Waiting for rate limit availability")
		_ = client.rateLimiter.Wait(ctx)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}

	if client.debug || forceDebug {
		fmt.Println("----- HTTP Response -----")
		if dump, err := httputil.DumpResponse(resp, true); err == nil {
			fmt.Println(string(dump))
		}
	}

	bodyBytes, err2 := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err2 != nil {
		return "", resp.StatusCode, err2
	}
	body := strings.TrimPrefix(string(bodyBytes), client.xssiPrefix)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("unexpected response code '%d': %s", resp.StatusCode, body)
	}

	// Empty response bodies are normalized to empty JSON objects for consistent parsing
	if body == "" {
		return "{}", resp.StatusCode, nil
	}

	return body, resp.StatusCode, nil
}
