package apiclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInjectIDIntoBody covers the pure helper: empty body, numeric coercion,
// non-numeric id, merging into an existing body, and invalid JSON.
func TestInjectIDIntoBody(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		key     string
		id      string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "empty body, numeric id -> number",
			body: "", key: "id", id: "5",
			want: map[string]interface{}{"id": float64(5)},
		},
		{
			name: "non-numeric id -> string",
			body: "", key: "id", id: "abc-123",
			want: map[string]interface{}{"id": "abc-123"},
		},
		{
			name: "merge into existing body, id wins",
			body: `{"name":"x","id":0}`, key: "id", id: "7",
			want: map[string]interface{}{"name": "x", "id": float64(7)},
		},
		{
			name: "custom key",
			body: `{"name":"x"}`, key: "pk", id: "9",
			want: map[string]interface{}{"name": "x", "pk": float64(9)},
		},
		{
			name: "invalid body errors",
			body: `{not json`, key: "id", id: "1",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := injectIDIntoBody(tc.body, tc.key, tc.id)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var m map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(got), &m))
			assert.Equal(t, tc.want, m)
		})
	}
}

// TestDeleteObject_BodyIDAttribute proves the id is sent in the DELETE body
// (the pfrest requirement) as a JSON number, not just in the URL.
func TestDeleteObject_BodyIDAttribute(t *testing.T) {
	ctx := context.Background()

	var gotBody string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := NewAPIClient(&APIClientOpt{URI: srv.URL, Timeout: 2})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:            "/api/objects",
		ID:              "5",
		IDAttribute:     "id",
		BodyIDAttribute: "id",
		Data:            `{"name":"x"}`,
	})
	require.NoError(t, err)

	require.NoError(t, obj.DeleteObject(ctx))
	assert.Equal(t, http.MethodDelete, gotMethod)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(gotBody), &m), "delete body must be JSON")
	assert.Equal(t, float64(5), m["id"], "id must be injected into the DELETE body as a number")
}

// TestUpdateObject_BodyIDAttribute proves the id is injected into the PATCH/PUT
// body alongside the managed fields.
func TestUpdateObject_BodyIDAttribute(t *testing.T) {
	ctx := context.Background()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":5,"name":"updated"}`))
	}))
	defer srv.Close()

	client, err := NewAPIClient(&APIClientOpt{URI: srv.URL, Timeout: 2, WriteReturnsObject: true})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:            "/api/objects",
		ID:              "5",
		IDAttribute:     "id",
		BodyIDAttribute: "id",
		Data:            `{"name":"updated"}`,
	})
	require.NoError(t, err)

	require.NoError(t, obj.UpdateObject(ctx))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(gotBody), &m), "update body must be JSON")
	assert.Equal(t, float64(5), m["id"], "id must be injected into the UPDATE body")
	assert.Equal(t, "updated", m["name"], "managed fields must be preserved")
}

// TestDeleteObject_ResolveBeforeWrite proves the id is re-resolved from the live
// collection right before delete (positional-id APIs): state says id=5 but the live
// list has id=4, so the DELETE must target 4.
func TestDeleteObject_ResolveBeforeWrite(t *testing.T) {
	ctx := context.Background()

	var delPath string
	var sawDelete bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"id":4,"host":"matrix"}]}`))
		case http.MethodDelete:
			sawDelete = true
			delPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	client, err := NewAPIClient(&APIClientOpt{URI: srv.URL, Timeout: 2})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:               "/api/objects",
		ID:                 "5", // stale id from state
		IDAttribute:        "id",
		Data:               `{"host":"matrix"}`,
		ResolveBeforeWrite: true,
		ReadSearch: map[string]string{
			"search_key":   "host",
			"search_value": "matrix",
			"results_key":  "data",
			"id_attribute": "id",
		},
	})
	require.NoError(t, err)

	require.NoError(t, obj.DeleteObject(ctx))
	assert.True(t, sawDelete, "a DELETE should have been issued")
	assert.Equal(t, "/api/objects/4", delPath, "delete must target the re-resolved live id (4), not the stale id (5)")
}

// TestDeleteObject_ResolveBeforeWrite_NotFound: if the object is gone from the live
// collection, delete is a no-op (no DELETE issued).
func TestDeleteObject_ResolveBeforeWrite_NotFound(t *testing.T) {
	ctx := context.Background()

	var sawDelete bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			sawDelete = true
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	client, err := NewAPIClient(&APIClientOpt{URI: srv.URL, Timeout: 2})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:               "/api/objects",
		ID:                 "5",
		IDAttribute:        "id",
		Data:               `{"host":"matrix"}`,
		ResolveBeforeWrite: true,
		ReadSearch: map[string]string{
			"search_key":   "host",
			"search_value": "matrix",
			"results_key":  "data",
			"id_attribute": "id",
		},
	})
	require.NoError(t, err)

	require.NoError(t, obj.DeleteObject(ctx))
	assert.False(t, sawDelete, "no DELETE should be issued when the object is already gone")
}

// TestDeleteObject_NoBodyIDAttribute confirms the default behavior is unchanged:
// without body_id_attribute, no body is sent on a plain delete.
func TestDeleteObject_NoBodyIDAttribute(t *testing.T) {
	ctx := context.Background()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := NewAPIClient(&APIClientOpt{URI: srv.URL, Timeout: 2})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		ID:          "5",
		IDAttribute: "id",
		Data:        `{"name":"x"}`,
	})
	require.NoError(t, err)

	require.NoError(t, obj.DeleteObject(ctx))
	assert.Empty(t, gotBody, "no body should be sent when body_id_attribute is unset")
}

// TestUpdateObject_ReadSearchUnwrapsEnvelope is the bug #4 regression test: a pfrest-style
// API returns a {code,status,...,data:{...}} ENVELOPE from the PUT, but the bare object from
// the read_search GET. With write_returns_object=true AND read_search configured, UpdateObject
// must RE-READ via read_search (storing the unwrapped object) instead of parsing the PUT
// envelope into apiData (which poisoned it with envelope keys and forced force_new everywhere).
func TestUpdateObject_ReadSearchUnwrapsEnvelope(t *testing.T) {
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut, http.MethodPost, http.MethodPatch:
			// pfrest-style envelope; "comment":"poison" proves a direct-parse path was wrongly taken
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":200,"status":"ok","return":0,"data":{"id":5,"host":"matrix","comment":"poison"}}`))
		case http.MethodGet:
			// read_search GET returns the clean collection; "comment":"updated" is the truth
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"id":5,"host":"matrix","comment":"updated"}]}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	client, err := NewAPIClient(&APIClientOpt{URI: srv.URL, Timeout: 2, WriteReturnsObject: true})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		ID:          "5",
		IDAttribute: "id",
		Data:        `{"host":"matrix","comment":"updated"}`,
		ReadSearch: map[string]string{
			"search_key":   "host",
			"search_value": "matrix",
			"results_key":  "data",
			"id_attribute": "id",
		},
	})
	require.NoError(t, err)

	require.NoError(t, obj.UpdateObject(ctx))

	got := obj.GetApiData()
	// the unwrapped object must be present...
	assert.Equal(t, "updated", got["comment"], "apiData must hold the re-read object, not the PUT envelope")
	assert.Equal(t, "matrix", got["host"])
	// ...and the envelope keys must NOT have leaked into apiData
	assert.NotContains(t, got, "code", "envelope key 'code' must not poison apiData")
	assert.NotContains(t, got, "status", "envelope key 'status' must not poison apiData")
	assert.NotContains(t, got, "return", "envelope key 'return' must not poison apiData")
	assert.NotContains(t, got, "data", "envelope key 'data' must not poison apiData")
}
