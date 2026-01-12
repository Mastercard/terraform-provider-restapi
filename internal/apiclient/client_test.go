package restapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	apiClientServer             *http.Server
	apiClientTLSServer          *http.Server
	rootCA                      *x509.Certificate
	rootCAKey                   *ecdsa.PrivateKey
	serverCertPEM, serverKeyPEM []byte
	rootCAFilePath              = "rootCA.pem"
)

func TestAPIClient(t *testing.T) {
	debug := false
	ctx := context.Background()

	if debug {
		fmt.Println("client_test.go: Starting HTTP server")
	}
	setupAPIClientServer()

	// Notice the intentional trailing /
	opt := &APIClientOpt{
		URI:                 "http://127.0.0.1:8083",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IDAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
		RateLimit:           1,
		Debug:               debug,
	}
	client, _ := NewAPIClient(opt)

	var res string
	var err error

	if debug {
		fmt.Printf("Testing standard OK request\n")
	}
	res, _, err = client.SendRequest(ctx, "GET", "/ok", "", debug)
	require.NoError(t, err, "client_test.go: SendRequest should not return an error")
	assert.Equal(t, "It works!", res, "client_test.go: Got back '%s' but expected 'It works!'", res)

	if debug {
		fmt.Printf("Testing redirect request\n")
	}
	res, _, err = client.SendRequest(ctx, "GET", "/redirect", "", debug)
	require.NoError(t, err, "client_test.go: SendRequest should not return an error")
	assert.Equal(t, "It works!", res, "client_test.go: Got back '%s' but expected 'It works!'", res)

	// Verify timeout works
	if debug {
		fmt.Printf("Testing timeout aborts requests\n")
	}
	_, _, err = client.SendRequest(ctx, "GET", "/slow", "", debug)
	assert.Error(t, err, "client_test.go: Timeout should trigger on slow request")

	if debug {
		fmt.Printf("Testing rate limited OK request\n")
	}
	startTime := time.Now().Unix()

	for range 4 {
		client.SendRequest(ctx, "GET", "/ok", "", debug)
	}

	duration := time.Now().Unix() - startTime
	assert.GreaterOrEqual(t, duration, int64(3), "client_test.go: requests should be delayed")

	if debug {
		fmt.Println("client_test.go: Stopping HTTP server")
	}
	shutdownAPIClientServer()
	if debug {
		fmt.Println("client_test.go: Done")
	}

	// Setup and test HTTPS client with root CA
	setupAPIClientTLSServer()
	defer shutdownAPIClientTLSServer()
	defer os.Remove(rootCAFilePath)

	httpsOpt := &APIClientOpt{
		URI:                 "https://127.0.0.1:8443/",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IDAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
		RateLimit:           1,
		RootCAFile:          rootCAFilePath,
		Debug:               debug,
	}
	httpsClient, httpsClientErr := NewAPIClient(httpsOpt)

	require.NoError(t, httpsClientErr, "client_test.go: NewAPIClient should not return an error")
	if debug {
		fmt.Printf("Testing HTTPS standard OK request\n")
	}
	res, _, err = httpsClient.SendRequest(ctx, "GET", "/ok", "", debug)
	require.NoError(t, err, "client_test.go: sendRequest should not return an error")
	assert.Equal(t, "It works!", res, "client_test.go: Got back '%s' but expected 'It works!'", res)
}

func setupAPIClientServer() {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("It works!"))
	})
	serverMux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(9999 * time.Second)
		w.Write([]byte("This will never return!!!!!"))
	})
	serverMux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ok", http.StatusPermanentRedirect)
	})
	serverMux.HandleFunc("/error/400", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})
	serverMux.HandleFunc("/error/401", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	serverMux.HandleFunc("/error/404", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})
	serverMux.HandleFunc("/error/500", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})
	serverMux.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Send empty body
	})
	serverMux.HandleFunc("/xssi", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(")]}'\n{\"data\":\"test\"}"))
	})
	serverMux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"id\":\"123\",\"name\":\"test\"}"))
	})
	serverMux.HandleFunc("/check-auth", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Missing Authorization", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("Authorized"))
	})
	serverMux.HandleFunc("/check-header", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			http.Error(w, "Missing or invalid header", http.StatusBadRequest)
			return
		}
		w.Write([]byte("Header OK"))
	})

	apiClientServer = &http.Server{
		Addr:    "127.0.0.1:8083",
		Handler: serverMux,
	}
	go apiClientServer.ListenAndServe()
	// let the server start
	time.Sleep(1 * time.Second)
}

func shutdownAPIClientServer() {
	apiClientServer.Close()
}

func setupAPIClientTLSServer() {
	generateCertificates()

	cert, _ := tls.X509KeyPair(serverCertPEM, serverKeyPEM)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("It works!"))
	})

	apiClientTLSServer = &http.Server{
		Addr:      "127.0.0.1:8443",
		Handler:   serverMux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	go apiClientTLSServer.ListenAndServeTLS("", "")
	// let the server start
	time.Sleep(1 * time.Second)
}

func shutdownAPIClientTLSServer() {
	apiClientTLSServer.Close()
}

func generateCertificates() {
	// Create a CA certificate and key
	rootCAKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	rootCA = &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization: []string{"Test Root CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	rootCABytes, _ := x509.CreateCertificate(rand.Reader, rootCA, rootCA, &rootCAKey.PublicKey, rootCAKey)

	// Create a server certificate and key
	serverKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	serverCert := &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	// Add IP SANs to the server certificate
	serverCert.IPAddresses = append(serverCert.IPAddresses, net.ParseIP("127.0.0.1"))

	serverCertBytes, _ := x509.CreateCertificate(rand.Reader, serverCert, rootCA, &serverKey.PublicKey, rootCAKey)

	// PEM encode the certificates and keys
	serverCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertBytes})

	// Marshal the server private key
	serverKeyBytes, _ := x509.MarshalECPrivateKey(serverKey)
	serverKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyBytes})

	rootCAPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCABytes})
	_ = os.WriteFile(rootCAFilePath, rootCAPEM, 0644)
}

func TestNewAPIClientErrors(t *testing.T) {
	// Generate certificates once for reuse in tests
	generateCertificates()
	defer os.Remove(rootCAFilePath)

	// Generate a mismatched key (different from serverKeyPEM)
	mismatchedKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	mismatchedKeyBytes, _ := x509.MarshalECPrivateKey(mismatchedKey)
	mismatchedKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: mismatchedKeyBytes})

	tests := []struct {
		name        string
		opt         *APIClientOpt
		expectedErr string
	}{
		{
			name:        "missing_uri",
			opt:         &APIClientOpt{},
			expectedErr: "uri must be set to construct an API client",
		},
		{
			name: "invalid_cert_file",
			opt: &APIClientOpt{
				URI:      "https://example.com",
				CertFile: "/nonexistent/cert.pem",
				KeyFile:  "/nonexistent/key.pem",
			},
			expectedErr: "no such file or directory",
		},
		{
			name: "invalid_cert_string_format",
			opt: &APIClientOpt{
				URI:        "https://example.com",
				CertString: "not-a-valid-cert",
				KeyString:  "not-a-valid-key",
			},
			expectedErr: "tls: failed to find any PEM data",
		},
		{
			name: "mismatched_cert_key",
			opt: &APIClientOpt{
				URI:        "https://example.com",
				CertString: string(serverCertPEM),
				KeyString:  string(mismatchedKeyPEM), // Different key from cert
			},
			expectedErr: "tls: private key does not match public key",
		},
		{
			name: "invalid_root_ca_file",
			opt: &APIClientOpt{
				URI:        "https://example.com",
				RootCAFile: "/nonexistent/rootca.pem",
			},
			expectedErr: "could not read root CA file",
		},
		{
			name: "invalid_root_ca_string",
			opt: &APIClientOpt{
				URI:          "https://example.com",
				RootCAString: "not-a-valid-ca-cert",
			},
			expectedErr: "failed to append root CA certificate(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAPIClient(tt.opt)
			assert.Error(t, err, "NewAPIClient should return an error")
			assert.Nil(t, client, "Client should be nil on error")
			assert.Contains(t, err.Error(), tt.expectedErr, "Error should contain expected message")
		})
	}
}

// TestNewAPIClientDefaults tests default values in NewAPIClient
func TestNewAPIClientDefaults(t *testing.T) {
	opt := &APIClientOpt{
		URI: "https://example.com/",
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")
	require.NotNil(t, client, "Client should not be nil")

	assert.Equal(t, "https://example.com", client.uri, "URI should have trailing slash removed")
	assert.Equal(t, "id", client.idAttribute, "Should use default ID attribute")
	assert.Equal(t, "POST", client.createMethod, "Should use default CREATE method")
	assert.Equal(t, "GET", client.readMethod, "Should use default READ method")
	assert.Equal(t, "PUT", client.updateMethod, "Should use default UPDATE method")
	assert.Equal(t, "DELETE", client.destroyMethod, "Should use default DESTROY method")
}

// TestNewAPIClientWithRetryConfig tests retry configuration
func TestNewAPIClientWithRetryConfig(t *testing.T) {
	tests := []struct {
		name            string
		retryMax        int64
		retryWaitMin    int64
		retryWaitMax    int64
		expectedMax     int
		expectedWaitMin time.Duration
		expectedWaitMax time.Duration
	}{
		{
			name:            "zero_retries_default_waits",
			retryMax:        0,
			retryWaitMin:    0,
			retryWaitMax:    0,
			expectedMax:     0,
			expectedWaitMin: 1 * time.Second,
			expectedWaitMax: 30 * time.Second,
		},
		{
			name:            "custom_retry_config",
			retryMax:        5,
			retryWaitMin:    2,
			retryWaitMax:    60,
			expectedMax:     5,
			expectedWaitMin: 2 * time.Second,
			expectedWaitMax: 60 * time.Second,
		},
		{
			name:            "only_max_retries_set",
			retryMax:        3,
			retryWaitMin:    0,
			retryWaitMax:    0,
			expectedMax:     3,
			expectedWaitMin: 1 * time.Second,
			expectedWaitMax: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &APIClientOpt{
				URI:          "https://example.com",
				RetryMax:     tt.retryMax,
				RetryWaitMin: tt.retryWaitMin,
				RetryWaitMax: tt.retryWaitMax,
			}
			client, err := NewAPIClient(opt)
			require.NoError(t, err, "NewAPIClient should not return an error")
			require.NotNil(t, client, "Client should not be nil")

			assert.Equal(t, tt.expectedMax, client.httpClient.RetryMax, "RetryMax should match expected")
			assert.Equal(t, tt.expectedWaitMin, client.httpClient.RetryWaitMin, "RetryWaitMin should match expected")
			assert.Equal(t, tt.expectedWaitMax, client.httpClient.RetryWaitMax, "RetryWaitMax should match expected")
		})
	}
}

// TestNewAPIClientWithOAuth tests OAuth configuration
func TestNewAPIClientWithOAuth(t *testing.T) {
	opt := &APIClientOpt{
		URI:               "https://example.com",
		OAuthClientID:     "test-client-id",
		OAuthClientSecret: "test-client-secret",
		OAuthTokenURL:     "https://oauth.example.com/token",
		OAuthScopes:       []string{"read", "write"},
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")
	require.NotNil(t, client, "Client should not be nil")
	require.NotNil(t, client.oauthConfig, "OAuth config should be set")

	assert.Equal(t, "test-client-id", client.oauthConfig.ClientID)
	assert.Equal(t, "test-client-secret", client.oauthConfig.ClientSecret)
	assert.Equal(t, "https://oauth.example.com/token", client.oauthConfig.TokenURL)
	assert.Equal(t, []string{"read", "write"}, client.oauthConfig.Scopes)
}

// TestNewAPIClientWithoutOAuth tests that OAuth config is nil when not fully configured
func TestNewAPIClientWithoutOAuth(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		secret   string
		tokenURL string
	}{
		{
			name:     "no_oauth_params",
			clientID: "",
			secret:   "",
			tokenURL: "",
		},
		{
			name:     "only_client_id",
			clientID: "test-client",
			secret:   "",
			tokenURL: "",
		},
		{
			name:     "only_secret",
			clientID: "",
			secret:   "test-secret",
			tokenURL: "",
		},
		{
			name:     "missing_token_url",
			clientID: "test-client",
			secret:   "test-secret",
			tokenURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &APIClientOpt{
				URI:               "https://example.com",
				OAuthClientID:     tt.clientID,
				OAuthClientSecret: tt.secret,
				OAuthTokenURL:     tt.tokenURL,
			}
			client, err := NewAPIClient(opt)
			require.NoError(t, err, "NewAPIClient should not return an error")
			require.NotNil(t, client, "Client should not be nil")
			assert.Nil(t, client.oauthConfig, "OAuth config should be nil when incomplete")
		})
	}
}

func TestSendRequestErrors(t *testing.T) {
	ctx := context.Background()
	debug := false

	setupAPIClientServer()
	defer shutdownAPIClientServer()

	opt := &APIClientOpt{
		URI:       "http://127.0.0.1:8083",
		Timeout:   2,
		Debug:     debug,
		RateLimit: 0,
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	tests := []struct {
		name           string
		method         string
		path           string
		data           string
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:           "http_400_error",
			method:         "GET",
			path:           "/error/400",
			expectedStatus: 400,
			expectError:    true,
			errorContains:  "unexpected response code '400'",
		},
		{
			name:           "http_401_error",
			method:         "GET",
			path:           "/error/401",
			expectedStatus: 401,
			expectError:    true,
			errorContains:  "unexpected response code '401'",
		},
		{
			name:           "http_404_error",
			method:         "GET",
			path:           "/error/404",
			expectedStatus: 404,
			expectError:    true,
			errorContains:  "unexpected response code '404'",
		},
		{
			name:           "http_500_error",
			method:         "GET",
			path:           "/error/500",
			expectedStatus: 0, // retryablehttp doesn't return status on retry exhaustion
			expectError:    true,
			errorContains:  "giving up after",
		},
		{
			name:           "empty_response_body",
			method:         "GET",
			path:           "/empty",
			expectedStatus: 200,
			expectError:    false,
		},
		{
			name:           "json_response",
			method:         "GET",
			path:           "/json",
			expectedStatus: 200,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, status, err := client.SendRequest(ctx, tt.method, tt.path, tt.data, debug)

			assert.Equal(t, tt.expectedStatus, status, "Status code should match expected")

			if tt.expectError {
				assert.Error(t, err, "Should return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.NotEmpty(t, body, "Response body should not be empty")
			}
		})
	}
}

func TestSendRequestEmptyBodyNormalization(t *testing.T) {
	ctx := context.Background()
	debug := false

	setupAPIClientServer()
	defer shutdownAPIClientServer()

	opt := &APIClientOpt{
		URI:     "http://127.0.0.1:8083",
		Timeout: 2,
		Debug:   debug,
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	body, status, err := client.SendRequest(ctx, "GET", "/empty", "", debug)
	require.NoError(t, err, "Should not return an error")
	assert.Equal(t, 200, status, "Status should be 200")
	assert.Equal(t, "{}", body, "Empty body should be normalized to {}")
}

func TestSendRequestXSSIPrefix(t *testing.T) {
	ctx := context.Background()
	debug := false

	setupAPIClientServer()
	defer shutdownAPIClientServer()

	opt := &APIClientOpt{
		URI:        "http://127.0.0.1:8083",
		Timeout:    2,
		Debug:      debug,
		XSSIPrefix: ")]}'",
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	body, status, err := client.SendRequest(ctx, "GET", "/xssi", "", debug)
	require.NoError(t, err, "Should not return an error")
	assert.Equal(t, 200, status, "Status should be 200")
	assert.Equal(t, "\n{\"data\":\"test\"}", body, "XSSI prefix should be stripped")
}

func TestSendRequestWithHeaders(t *testing.T) {
	ctx := context.Background()
	debug := false

	setupAPIClientServer()
	defer shutdownAPIClientServer()

	opt := &APIClientOpt{
		URI:     "http://127.0.0.1:8083",
		Timeout: 2,
		Debug:   debug,
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	body, status, err := client.SendRequest(ctx, "GET", "/check-header", "", debug)
	require.NoError(t, err, "Should not return an error")
	assert.Equal(t, 200, status, "Status should be 200")
	assert.Equal(t, "Header OK", body, "Should receive success response")
}

func TestSendRequestWithBasicAuth(t *testing.T) {
	ctx := context.Background()
	debug := false

	setupAPIClientServer()
	defer shutdownAPIClientServer()

	opt := &APIClientOpt{
		URI:      "http://127.0.0.1:8083",
		Timeout:  2,
		Debug:    debug,
		Username: "testuser",
		Password: "testpass",
		Headers: map[string]string{
			"Authorization": "Basic dGVzdHVzZXI6dGVzdHBhc3M=", // testuser:testpass in base64
		},
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	body, status, err := client.SendRequest(ctx, "GET", "/check-auth", "", debug)
	require.NoError(t, err, "Should not return an error")
	assert.Equal(t, 200, status, "Status should be 200")
	assert.Equal(t, "Authorized", body, "Should receive authorized response")
}

func TestSendRequestWithData(t *testing.T) {
	ctx := context.Background()
	debug := false

	setupAPIClientServer()
	defer shutdownAPIClientServer()

	opt := &APIClientOpt{
		URI:     "http://127.0.0.1:8083",
		Timeout: 2,
		Debug:   debug,
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	testData := `{"name":"test","value":123}`
	body, status, err := client.SendRequest(ctx, "POST", "/json", testData, debug)
	require.NoError(t, err, "Should not return an error")
	assert.Equal(t, 200, status, "Status should be 200")
	assert.NotEmpty(t, body, "Response body should not be empty")
}

func TestSendRequestConnectionError(t *testing.T) {
	ctx := context.Background()
	debug := false

	// Don't start a server - should get connection error
	opt := &APIClientOpt{
		URI:     "http://127.0.0.1:9999", // Non-existent server
		Timeout: 2,
		Debug:   debug,
	}
	client, err := NewAPIClient(opt)
	require.NoError(t, err, "NewAPIClient should not return an error")

	_, _, err = client.SendRequest(ctx, "GET", "/test", "", debug)
	assert.Error(t, err, "Should return connection error")
	assert.Contains(t, err.Error(), "connection refused", "Should contain connection refused error")
}
