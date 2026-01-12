package restapi

import (
	"testing"
)

// TestAppendIdToPath tests the appendIdToPath function with various scenarios
func TestAppendIdToPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path without query string",
			path:     "/api/objects",
			expected: "/api/objects/{id}",
		},
		{
			name:     "path with query string",
			path:     "/enrollmentGroups?api-version=2021-10-01",
			expected: "/enrollmentGroups/{id}?api-version=2021-10-01",
		},
		{
			name:     "path already contains {id} placeholder",
			path:     "/api/objects/{id}",
			expected: "/api/objects/{id}",
		},
		{
			name:     "path with {id} and query string",
			path:     "/api/objects/{id}?version=v1",
			expected: "/api/objects/{id}?version=v1",
		},
		{
			name:     "path with multiple query parameters",
			path:     "/api/objects?version=v1&format=json",
			expected: "/api/objects/{id}?version=v1&format=json",
		},
		{
			name:     "path with nested path and query string",
			path:     "/api/v1/objects?expand=true",
			expected: "/api/v1/objects/{id}?expand=true",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "/{id}",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "//{id}",
		},
		{
			name:     "path with trailing slash",
			path:     "/api/objects/",
			expected: "/api/objects//{id}",
		},
		{
			name:     "path with trailing slash and query string",
			path:     "/api/objects/?version=v1",
			expected: "/api/objects//{id}?version=v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendIdToPath(tt.path)
			if result != tt.expected {
				t.Errorf("appendIdToPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// TestNewAPIObject_PathsWithQueryStrings verifies that paths with query strings
// get {id} inserted correctly when default paths are constructed
func TestNewAPIObject_PathsWithQueryStrings(t *testing.T) {
	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://localhost:8080",
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	tests := []struct {
		name                string
		path                string
		expectedReadPath    string
		expectedUpdatePath  string
		expectedDestroyPath string
	}{
		{
			name:                "path without query string",
			path:                "/api/objects",
			expectedReadPath:    "/api/objects/{id}",
			expectedUpdatePath:  "/api/objects/{id}",
			expectedDestroyPath: "/api/objects/{id}",
		},
		{
			name:                "path with query string",
			path:                "/enrollmentGroups?api-version=2021-10-01",
			expectedReadPath:    "/enrollmentGroups/{id}?api-version=2021-10-01",
			expectedUpdatePath:  "/enrollmentGroups/{id}?api-version=2021-10-01",
			expectedDestroyPath: "/enrollmentGroups/{id}?api-version=2021-10-01",
		},
		{
			name:                "path already with {id} and query string",
			path:                "/api/{id}?version=v1",
			expectedReadPath:    "/api/{id}?version=v1",
			expectedUpdatePath:  "/api/{id}?version=v1",
			expectedDestroyPath: "/api/{id}?version=v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &APIObjectOpts{
				Path:        tt.path,
				Data:        `{"name": "test"}`,
				IDAttribute: "id",
			}

			obj, err := NewAPIObject(client, opts)
			if err != nil {
				t.Fatalf("NewAPIObject() error = %v", err)
			}

			if obj.readPath != tt.expectedReadPath {
				t.Errorf("readPath = %q, want %q", obj.readPath, tt.expectedReadPath)
			}
			if obj.updatePath != tt.expectedUpdatePath {
				t.Errorf("updatePath = %q, want %q", obj.updatePath, tt.expectedUpdatePath)
			}
			if obj.deletePath != tt.expectedDestroyPath {
				t.Errorf("deletePath = %q, want %q", obj.deletePath, tt.expectedDestroyPath)
			}
		})
	}
}
