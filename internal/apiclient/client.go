package restapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
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

type apiClientOpt struct {
	uri                 string
	insecure            bool
	username            string
	password            string
	headers             map[string]string
	timeout             int
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
	useCookies          bool
	rateLimit           float64
	oauthClientID       string
	oauthClientSecret   string
	oauthScopes         []string
	oauthTokenURL       string
	oauthEndpointParams url.Values
	certFile            string
	keyFile             string
	rootCAFile          string
	certString          string
	keyString           string
	rootCAString        string
	debug               bool
}

/*APIClient is a HTTP client with additional controlling fields*/
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
}

// NewAPIClient makes a new api client for RESTful calls
func NewAPIClient(opt *apiClientOpt) (*APIClient, error) {
	ctx := context.Background()
	if opt.debug {
		log.Printf("api_client.go: Constructing debug api_client\n")
	}

	if opt.uri == "" {
		return nil, errors.New("uri must be set to construct an API client")
	}

	/* Sane default */
	if opt.idAttribute == "" {
		opt.idAttribute = "id"
	}

	/* Remove any trailing slashes since we will append
	   to this URL with our own root-prefixed location */
	opt.uri = strings.TrimSuffix(opt.uri, "/")

	if opt.createMethod == "" {
		opt.createMethod = "POST"
	}
	if opt.readMethod == "" {
		opt.readMethod = "GET"
	}
	if opt.updateMethod == "" {
		opt.updateMethod = "PUT"
	}
	if opt.destroyMethod == "" {
		opt.destroyMethod = "DELETE"
	}

	tlsConfig := &tls.Config{
		/* Disable TLS verification if requested */
		InsecureSkipVerify: opt.insecure,
	}

	if opt.certString != "" && opt.keyString != "" {
		cert, err := tls.X509KeyPair([]byte(opt.certString), []byte(opt.keyString))
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if opt.certFile != "" && opt.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(opt.certFile, opt.keyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load root CA
	if opt.rootCAFile != "" || opt.rootCAString != "" {
		caCertPool := x509.NewCertPool()
		var rootCA []byte
		var err error

		if opt.rootCAFile != "" {
			tflog.Debug(ctx, "api_client.go: Reading root CA file", map[string]interface{}{"rootCAFile": opt.rootCAFile})
			rootCA, err = os.ReadFile(opt.rootCAFile)
			if err != nil {
				return nil, fmt.Errorf("could not read root CA file: %v", err)
			}
		} else {
			tflog.Debug(ctx, "api_client.go: Using provided root CA string")
			rootCA = []byte(opt.rootCAString)
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

	if opt.useCookies {
		cookieJar, _ = cookiejar.New(nil)
	}

	rateLimit := rate.Limit(opt.rateLimit)
	bucketSize := int(math.Max(math.Round(opt.rateLimit), 1))
	tflog.Info(ctx, "rate limit configured", map[string]interface{}{"rateLimit": opt.rateLimit, "bucketSize": bucketSize})
	rateLimiter := rate.NewLimiter(rateLimit, bucketSize)

	httpClient := cleanhttp.DefaultClient()
	httpClient.Timeout = time.Second * time.Duration(opt.timeout)
	httpClient.Transport = tr
	httpClient.Jar = cookieJar

	client := APIClient{
		httpClient:          httpClient,
		rateLimiter:         rateLimiter,
		uri:                 opt.uri,
		insecure:            opt.insecure,
		username:            opt.username,
		password:            opt.password,
		headers:             opt.headers,
		idAttribute:         opt.idAttribute,
		createMethod:        opt.createMethod,
		readMethod:          opt.readMethod,
		readData:            opt.readData,
		updateMethod:        opt.updateMethod,
		updateData:          opt.updateData,
		destroyMethod:       opt.destroyMethod,
		destroyData:         opt.destroyData,
		copyKeys:            opt.copyKeys,
		writeReturnsObject:  opt.writeReturnsObject,
		createReturnsObject: opt.createReturnsObject,
		xssiPrefix:          opt.xssiPrefix,
		debug:               opt.debug,
	}

	if opt.oauthClientID != "" && opt.oauthClientSecret != "" && opt.oauthTokenURL != "" {
		client.oauthConfig = &clientcredentials.Config{
			ClientID:       opt.oauthClientID,
			ClientSecret:   opt.oauthClientSecret,
			TokenURL:       opt.oauthTokenURL,
			Scopes:         opt.oauthScopes,
			EndpointParams: opt.oauthEndpointParams,
		}
	}

	tflog.Debug(ctx, "api_client.go: Constructed client", map[string]interface{}{"details": client.String()})
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
	for _, n := range client.copyKeys {
		buffer.WriteString(fmt.Sprintf("  %s", n))
	}
	return buffer.String()
}

/*
Helper function that handles sending/receiving and handling

	of HTTP data in and out.
*/
func (client *APIClient) sendRequest(ctx context.Context, method string, path string, data string) (string, error) {
	fullURI := client.uri + path
	var req *http.Request
	var err error

	tflog.Debug(ctx, "api_client.go: Sending request", map[string]interface{}{"method": method, "path": path, "fullURI": fullURI, "data": data})

	buffer := bytes.NewBuffer([]byte(data))

	if data == "" {
		req, err = http.NewRequest(method, fullURI, nil)
	} else {
		req, err = http.NewRequest(method, fullURI, buffer)

		/* Default of application/json, but allow headers array to overwrite later */
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	/* Allow for tokens or other pre-created secrets */
	if len(client.headers) > 0 {
		for n, v := range client.headers {
			req.Header.Set(n, v)
		}
	}

	if client.oauthConfig != nil {
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client.httpClient)
		tokenSource := client.oauthConfig.TokenSource(ctx)
		token, err := tokenSource.Token()
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}

	if client.username != "" && client.password != "" {
		/* ... and fall back to basic auth if configured */
		req.SetBasicAuth(client.username, client.password)
	}

	if client.debug {
		fmt.Println(httputil.DumpRequest(req, true))
	}

	if client.rateLimiter != nil {
		tflog.Debug(ctx, "Waiting for rate limit availability")
		_ = client.rateLimiter.Wait(ctx)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	if client.debug {
		fmt.Println(httputil.DumpResponse(resp, true))
	}

	bodyBytes, err2 := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err2 != nil {
		return "", err2
	}
	body := strings.TrimPrefix(string(bodyBytes), client.xssiPrefix)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("unexpected response code '%d': %s", resp.StatusCode, body)
	}

	if body == "" {
		return "{}", nil
	}

	return body, nil
}
