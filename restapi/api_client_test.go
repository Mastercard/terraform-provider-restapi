package restapi

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
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

	if debug {
		log.Println("client_test.go: Starting HTTP server")
	}
	setupAPIClientServer()

	/* Notice the intentional trailing / */
	opt := &apiClientOpt{
		uri:                 "http://127.0.0.1:8083/",
		insecure:            false,
		username:            "",
		password:            "",
		headers:             make(map[string]string),
		timeout:             2,
		idAttribute:         "id",
		copyKeys:            make([]string, 0),
		writeReturnsObject:  false,
		createReturnsObject: false,
		rateLimit:           1,
		debug:               debug,
	}
	client, _ := NewAPIClient(opt)

	var res string
	var err error

	if debug {
		log.Printf("api_client_test.go: Testing standard OK request\n")
	}
	res, err = client.sendRequest("GET", "/ok", "")
	if err != nil {
		t.Fatalf("client_test.go: %s", err)
	}
	if res != "It works!" {
		t.Fatalf("client_test.go: Got back '%s' but expected 'It works!'\n", res)
	}

	if debug {
		log.Printf("api_client_test.go: Testing redirect request\n")
	}
	res, err = client.sendRequest("GET", "/redirect", "")
	if err != nil {
		t.Fatalf("client_test.go: %s", err)
	}
	if res != "It works!" {
		t.Fatalf("client_test.go: Got back '%s' but expected 'It works!'\n", res)
	}

	/* Verify timeout works */
	if debug {
		log.Printf("api_client_test.go: Testing timeout aborts requests\n")
	}
	_, err = client.sendRequest("GET", "/slow", "")
	if err == nil {
		t.Fatalf("client_test.go: Timeout did not trigger on slow request")
	}

	if debug {
		log.Printf("api_client_test.go: Testing rate limited OK request\n")
	}
	startTime := time.Now().Unix()

	for i := 0; i < 4; i++ {
		client.sendRequest("GET", "/ok", "")
	}

	duration := time.Now().Unix() - startTime
	if duration < 3 {
		t.Fatalf("client_test.go: requests not delayed\n")
	}

	if debug {
		log.Println("client_test.go: Stopping HTTP server")
	}
	shutdownAPIClientServer()
	if debug {
		log.Println("client_test.go: Done")
	}

	// Setup and test HTTPS client with root CA
	setupAPIClientTLSServer()
	defer shutdownAPIClientTLSServer()
	defer os.Remove(rootCAFilePath)

	httpsOpt := &apiClientOpt{
		uri:                 "https://127.0.0.1:8443/",
		insecure:            false,
		username:            "",
		password:            "",
		headers:             make(map[string]string),
		timeout:             2,
		idAttribute:         "id",
		copyKeys:            make([]string, 0),
		writeReturnsObject:  false,
		createReturnsObject: false,
		rateLimit:           1,
		rootCAFile:          rootCAFilePath,
		debug:               debug,
	}
	httpsClient, httpsClientErr := NewAPIClient(httpsOpt)

	if httpsClientErr != nil {
		t.Fatalf("client_test.go: %s", httpsClientErr)
	}
	if debug {
		log.Printf("api_client_test.go: Testing HTTPS standard OK request\n")
	}
	res, err = httpsClient.sendRequest("GET", "/ok", "")
	if err != nil {
		t.Fatalf("client_test.go: %s", err)
	}
	if res != "It works!" {
		t.Fatalf("client_test.go: Got back '%s' but expected 'It works!'\n", res)
	}
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
	/* let the server start */
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
	/* let the server start */
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
