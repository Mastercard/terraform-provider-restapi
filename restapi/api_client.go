package restapi

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type apiClientOpt struct {
	uri                   string
	insecure              bool
	username              string
	password              string
	headers               map[string]string
	use_cookie            bool
	timeout               int
	id_attribute          string
	create_method         string
	read_method           string
	update_method         string
	destroy_method        string
	copy_keys             []string
	write_returns_object  bool
	create_returns_object bool
	xssi_prefix           string
	use_cookies           bool
	debug                 bool
	retry_methods         []string
}

type api_client struct {
	http_client           *http.Client
	uri                   string
	insecure              bool
	username              string
	password              string
	headers               map[string]string
	use_cookie            bool
	timeout               int
	id_attribute          string
	create_method         string
	read_method           string
	update_method         string
	destroy_method        string
	copy_keys             []string
	write_returns_object  bool
	create_returns_object bool
	xssi_prefix           string
	debug                 bool
	retry_methods         []string
}

// Make a new api client for RESTful calls
func NewAPIClient(opt *apiClientOpt) (*api_client, error) {
	if opt.debug {
		log.Printf("api_client.go: Constructing debug api_client\n")
	}

	if opt.uri == "" {
		return nil, errors.New("uri must be set to construct an API client")
	}

	/* Sane default */
	if opt.id_attribute == "" {
		opt.id_attribute = "id"
	}

	/* Remove any trailing slashes since we will append
	   to this URL with our own root-prefixed location */
	if strings.HasSuffix(opt.uri, "/") {
		opt.uri = opt.uri[:len(opt.uri)-1]
	}

	if opt.create_method == "" {
		opt.create_method = "POST"
	}
	if opt.read_method == "" {
		opt.read_method = "GET"
	}
	if opt.update_method == "" {
		opt.create_method = "PUT"
	}
	if opt.destroy_method == "" {
		opt.destroy_method = "DELETE"
	}

	/* Disable TLS verification if requested */
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: opt.insecure},
	}

	var cookieJar http.CookieJar

	if opt.use_cookies {
		cookieJar, _ = cookiejar.New(nil)
	}

	client := api_client{
		http_client: &http.Client{
			Timeout:   time.Second * time.Duration(opt.timeout),
			Transport: tr,
			Jar:       cookieJar,
		},
		uri:                   opt.uri,
		insecure:              opt.insecure,
		username:              opt.username,
		password:              opt.password,
		headers:               opt.headers,
		id_attribute:          opt.id_attribute,
		create_method:         opt.create_method,
		read_method:           opt.read_method,
		update_method:         opt.update_method,
		destroy_method:        opt.destroy_method,
		copy_keys:             opt.copy_keys,
		write_returns_object:  opt.write_returns_object,
		create_returns_object: opt.create_returns_object,
		xssi_prefix:           opt.xssi_prefix,
		debug:                 opt.debug,
		retry_methods:         opt.retry_methods,
	}

	if opt.debug {
		log.Printf("api_client.go: Constructed object:\n%s", client.toString())
	}
	return &client, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *api_client) toString() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("uri: %s\n", obj.uri))
	buffer.WriteString(fmt.Sprintf("insecure: %t\n", obj.insecure))
	buffer.WriteString(fmt.Sprintf("username: %s\n", obj.username))
	buffer.WriteString(fmt.Sprintf("password: %s\n", obj.password))
	buffer.WriteString(fmt.Sprintf("id_attribute: %s\n", obj.id_attribute))
	buffer.WriteString(fmt.Sprintf("write_returns_object: %t\n", obj.write_returns_object))
	buffer.WriteString(fmt.Sprintf("create_returns_object: %t\n", obj.create_returns_object))
	buffer.WriteString(fmt.Sprintf("headers:\n"))
	for k, v := range obj.headers {
		buffer.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	for _, n := range obj.copy_keys {
		buffer.WriteString(fmt.Sprintf("  %s", n))
	}
	buffer.WriteString(fmt.Sprintf("retry_methods:\n"))
	for _, n := range obj.retry_methods {
		buffer.WriteString(fmt.Sprintf("  %s", n))
	}
	return buffer.String()
}

/* Helper function for retry_methods condition */
func sliceContains(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

/* Helper function that handles sending/receiving and handling
   of HTTP data in and out.
   TODO: Handle redirects */
func (client *api_client) send_request(method string, path string, data string) (string, error) {
	full_uri := client.uri + path
	var req *http.Request
	var err error

	if client.debug {
		log.Printf("api_client.go: method='%s', path='%s', full uri (derived)='%s', data='%s'\n", method, path, full_uri, data)
	}

	buffer := bytes.NewBuffer([]byte(data))

	if data == "" {
		req, err = http.NewRequest(method, full_uri, nil)
	} else {
		req, err = http.NewRequest(method, full_uri, buffer)

		/* Default of application/json, but allow headers array to overwrite later */
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	if err != nil {
		log.Fatal(err)
		return "", err
	}

	if client.debug {
		log.Printf("api_client.go: Sending HTTP request to %s...\n", req.URL)
	}

	/* Allow for tokens or other pre-created secrets */
	if len(client.headers) > 0 {
		for n, v := range client.headers {
			req.Header.Set(n, v)
		}
	}

	if client.username != "" && client.password != "" {
		/* ... and fall back to basic auth if configured */
		req.SetBasicAuth(client.username, client.password)
	}

	if client.debug {
		log.Printf("api_client.go: Request headers:\n")
		for name, headers := range req.Header {
			for _, h := range headers {
				log.Printf("api_client.go:   %v: %v", name, h)
			}
		}

		log.Printf("api_client.go: BODY:\n")
		body := "<none>"
		if req.Body != nil {
			body = string(data)
		}
		log.Printf("%s\n", body)
	}

	/* Retry only if this is one of the user specified HTTP methods to retry on */
	num_retries := 0
	if len(client.retry_methods) > 0 && sliceContains(method, client.retry_methods) {
		num_retries = 5
	}

	for num_retries >= 0 {
		resp, err := client.http_client.Do(req)

		if err != nil {
			//log.Printf("api_client.go: Error detected: %s\n", err)
			return "", err
		}

		if client.debug {
			log.Printf("api_client.go: Response code: %d\n", resp.StatusCode)
			log.Printf("api_client.go: Response headers:\n")
			for name, headers := range resp.Header {
				for _, h := range headers {
					log.Printf("api_client.go:   %v: %v", name, h)
				}
			}
		}

		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err2 != nil {
			return "", err2
		}
		body := strings.TrimPrefix(string(bodyBytes), client.xssi_prefix)

		if resp.StatusCode == 301 || resp.StatusCode == 302 {
			//Redirecting... decrement num_retries and proceed to the next loop
			//uri = URI.parse(rsp['Location'])
		} else if num_retries != 0 && resp.StatusCode >= 500 && resp.StatusCode < 600 {
			if client.debug {
				log.Printf("Received response code '%d': %s - Retrying", resp.StatusCode, body)
			}
		} else if resp.StatusCode == 404 || resp.StatusCode < 200 || resp.StatusCode >= 303 {
			return "", errors.New(fmt.Sprintf("Unexpected response code '%d': %s", resp.StatusCode, body))
		} else {
			if client.debug {
				log.Printf("api_client.go: BODY:\n%s\n", body)
			}
			return body, nil
		}
		num_retries--
	} //End loop through retry attempts

	return "", errors.New("Error - too many retries!")
}
