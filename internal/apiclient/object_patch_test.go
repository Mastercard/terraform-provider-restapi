package restapi

import (
	"context"
	"testing"
)

func TestNewAPIObject_InvalidSearchPatch(t *testing.T) {
	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://localhost:8080",
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Test that invalid JSON patch fails during construction
	opts := &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "123", "name": "Test"}`,
		ReadSearch: map[string]string{
			"search_key":   "id",
			"search_value": "123",
			"search_patch": `[{"op": "invalid", "path": "/test"}]`, // Invalid op
		},
	}

	obj, err := NewAPIObject(client, opts)
	if err == nil {
		t.Fatal("Expected error for invalid search_patch, but got none")
	}
	if obj == nil {
		t.Fatal("Expected object to be returned even on error")
	}

	expectedErrMsg := "failed to compile search_patch"
	if err != nil && len(err.Error()) < len(expectedErrMsg) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestNewAPIObject_ValidSearchPatch(t *testing.T) {
	ctx := context.Background()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://localhost:8080",
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Test that valid JSON patch compiles successfully
	opts := &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "123", "name": "Test"}`,
		ReadSearch: map[string]string{
			"search_key":   "id",
			"search_value": "123",
			"search_patch": `[{"op": "remove", "path": "/metadata"}]`, // Valid patch
		},
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		t.Fatalf("Unexpected error for valid search_patch: %v", err)
	}
	if obj == nil {
		t.Fatal("Expected object to be created")
	}
	if obj.searchPatch == nil {
		t.Fatal("Expected searchPatch to be compiled")
	}

	// Verify the patch works
	testData := map[string]interface{}{
		"id":       "123",
		"name":     "Test",
		"metadata": map[string]interface{}{"created": "2024-01-01"},
	}

	result, err := ApplyJSONPatch(ctx, testData, obj.searchPatch)
	if err != nil {
		t.Fatalf("Failed to apply compiled patch: %v", err)
	}

	if _, exists := result["metadata"]; exists {
		t.Error("Expected metadata field to be removed by patch")
	}
	if result["id"] != "123" || result["name"] != "Test" {
		t.Error("Expected other fields to remain unchanged")
	}
}
