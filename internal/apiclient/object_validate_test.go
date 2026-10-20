package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newValidateTestClient creates an APIClient pointed at svr.URL.
func newValidateTestClient(t *testing.T, svrURL string) *APIClient {
	t.Helper()
	client, err := NewAPIClient(&APIClientOpt{
		URI:     svrURL,
		Timeout: 5,
	})
	require.NoError(t, err)
	return client
}

// TestValidateObject_NoValidatePath verifies ValidateObject is a no-op when validate_path is empty.
func TestValidateObject_NoValidatePath(t *testing.T) {
	ctx := context.Background()

	called := false
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	client := newValidateTestClient(t, svr.URL)
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/objects",
		Data: `{"id":"1","name":"test"}`,
	})
	require.NoError(t, err)

	err = obj.ValidateObject(ctx)
	assert.NoError(t, err, "ValidateObject should be a no-op when validate_path is empty")
	assert.False(t, called, "No HTTP request should be made when validate_path is empty")
}

// TestValidateObject_Success verifies a 200 response from validate_path passes.
func TestValidateObject_Success(t *testing.T) {
	ctx := context.Background()

	var receivedMethod, receivedPath, receivedBody string
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		receivedBody = string(buf)
		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	client := newValidateTestClient(t, svr.URL)
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:         "/objects",
		Data:         `{"id":"1","name":"test"}`,
		ValidatePath: "/objects/validate",
	})
	require.NoError(t, err)

	err = obj.ValidateObject(ctx)
	assert.NoError(t, err, "ValidateObject should succeed on 200")
	assert.Equal(t, "POST", receivedMethod, "Default method should be POST")
	assert.Equal(t, "/objects/validate", receivedPath)
	assert.Contains(t, receivedBody, `"name":"test"`, "Data should be sent in the request body")
}

// TestValidateObject_CustomMethod verifies that validate_method overrides the default POST.
func TestValidateObject_CustomMethod(t *testing.T) {
	ctx := context.Background()

	var receivedMethod string
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	client := newValidateTestClient(t, svr.URL)
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:           "/objects",
		Data:           `{"id":"1"}`,
		ValidatePath:   "/objects/validate",
		ValidateMethod: "PUT",
	})
	require.NoError(t, err)

	err = obj.ValidateObject(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "PUT", receivedMethod, "ValidateMethod should override default POST")
}

// TestValidateObject_APIRejectsConfig verifies a 4xx response fails the validation with an error.
func TestValidateObject_APIRejectsConfig(t *testing.T) {
	ctx := context.Background()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errorMessage":"tableName is required"}`, http.StatusBadRequest)
	}))
	defer svr.Close()

	client := newValidateTestClient(t, svr.URL)
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:         "/objects",
		Data:         `{"name":"bad"}`,
		ValidatePath: "/objects/validate",
	})
	require.NoError(t, err)

	err = obj.ValidateObject(ctx)
	assert.Error(t, err, "ValidateObject should return an error on 4xx response")
	assert.Contains(t, err.Error(), "/objects/validate", "Error should mention the validate path")
}

// TestValidateObject_ServerError verifies a 5xx response also fails validation.
func TestValidateObject_ServerError(t *testing.T) {
	ctx := context.Background()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer svr.Close()

	client := newValidateTestClient(t, svr.URL)
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:         "/objects",
		Data:         `{"id":"1"}`,
		ValidatePath: "/objects/validate",
	})
	require.NoError(t, err)

	err = obj.ValidateObject(ctx)
	assert.Error(t, err, "ValidateObject should return an error on 5xx response")
}

// TestValidateObject_GettersReturnConfiguredValues verifies accessor methods.
func TestValidateObject_GettersReturnConfiguredValues(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	client := newValidateTestClient(t, svr.URL)
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:           "/objects",
		Data:           `{"id":"1"}`,
		ValidatePath:   "/objects/validate",
		ValidateMethod: "POST",
	})
	require.NoError(t, err)

	assert.Equal(t, "/objects/validate", obj.GetValidatePath())
	assert.Equal(t, "POST", obj.GetValidateMethod())
}
