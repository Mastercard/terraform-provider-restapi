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
		URI:                 "http://127.0.0.1:8083/",
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
	res, err = client.SendRequest(ctx, "GET", "/ok", "", debug)
	require.NoError(t, err, "client_test.go: SendRequest should not return an error")
	assert.Equal(t, "It works!", res, "client_test.go: Got back '%s' but expected 'It works!'", res)

	if debug {
		fmt.Printf("Testing redirect request\n")
	}
	res, err = client.SendRequest(ctx, "GET", "/redirect", "", debug)
	require.NoError(t, err, "client_test.go: SendRequest should not return an error")
	assert.Equal(t, "It works!", res, "client_test.go: Got back '%s' but expected 'It works!'", res)

	// Verify timeout works
	if debug {
		fmt.Printf("Testing timeout aborts requests\n")
	}
	_, err = client.SendRequest(ctx, "GET", "/slow", "", debug)
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
	res, err = httpsClient.SendRequest(ctx, "GET", "/ok", "", debug)
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
