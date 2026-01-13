package apiclient

import (
	"context"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateObject_Success tests successful object creation
func TestCreateObject_Success(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8082, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                 "http://127.0.0.1:8082",
		Timeout:             2,
		WriteReturnsObject:  true,
		CreateReturnsObject: true,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "test1", "name": "Test Object"}`,
	})
	require.NoError(t, err)

	err = obj.CreateObject(ctx)
	assert.NoError(t, err, "CreateObject should succeed")
	assert.Equal(t, "test1", obj.ID, "Object ID should be set")
}

// TestCreateObject_NoIDNoWriteReturns tests error when ID is missing and write_returns_object is false
func TestCreateObject_NoIDNoWriteReturns(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8083, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                 "http://127.0.0.1:8083",
		Timeout:             2,
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"name": "Test Object"}`, // No ID
	})
	require.NoError(t, err)

	err = obj.CreateObject(ctx)
	assert.Error(t, err, "CreateObject should fail when ID is missing and write_returns_object is false")
	assert.Contains(t, err.Error(), "does not have an id set", "Error should mention missing ID")
}

// TestCreateObject_WithQueryString tests object creation with query string
func TestCreateObject_WithQueryString(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8084, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                 "http://127.0.0.1:8084",
		Timeout:             2,
		WriteReturnsObject:  true,
		CreateReturnsObject: true,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "test2", "name": "Test Object"}`,
		QueryString: "force=true",
	})
	require.NoError(t, err)

	err = obj.CreateObject(ctx)
	assert.NoError(t, err, "CreateObject with query string should succeed")
}

// TestCreateObject_WithoutWriteReturns tests creation when write_returns_object is false
func TestCreateObject_WithoutWriteReturns(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"test3": {
			"id":   "test3",
			"name": "Test Object",
		},
	}
	svr := fakeserver.NewFakeServer(8085, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                 "http://127.0.0.1:8085",
		Timeout:             2,
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "test3", "name": "Test Object"}`,
	})
	require.NoError(t, err)

	err = obj.CreateObject(ctx)
	assert.NoError(t, err, "CreateObject should succeed and fetch object")
	assert.Equal(t, "test3", obj.ID)
}

// TestUpdateObject_Success tests successful object update
func TestUpdateObject_Success(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"update1": {
			"id":   "update1",
			"name": "Original Name",
		},
	}
	svr := fakeserver.NewFakeServer(8086, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                "http://127.0.0.1:8086",
		Timeout:            2,
		WriteReturnsObject: true,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "update1", "name": "Updated Name"}`,
	})
	require.NoError(t, err)

	err = obj.UpdateObject(ctx)
	assert.NoError(t, err, "UpdateObject should succeed")
}

// TestUpdateObject_NoID tests error when updating object without ID
func TestUpdateObject_NoID(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8087, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8087",
		Timeout: 2,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"name": "No ID Object"}`,
	})
	require.NoError(t, err)

	err = obj.UpdateObject(ctx)
	assert.Error(t, err, "UpdateObject should fail when ID is missing")
	assert.Contains(t, err.Error(), "cannot update an object unless the ID has been set", "Error should mention missing ID")
}

// TestUpdateObject_WithUpdateData tests update using update_data instead of full data
func TestUpdateObject_WithUpdateData(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"update2": {
			"id":          "update2",
			"name":        "Original Name",
			"description": "Original Description",
		},
	}
	svr := fakeserver.NewFakeServer(8088, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                "http://127.0.0.1:8088",
		Timeout:            2,
		WriteReturnsObject: true,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:       "/api/objects",
		Data:       `{"id": "update2", "name": "Original Name", "description": "Original Description"}`,
		UpdateData: `{"name": "Partial Update"}`,
	})
	require.NoError(t, err)

	err = obj.UpdateObject(ctx)
	assert.NoError(t, err, "UpdateObject with update_data should succeed")
}

// TestUpdateObject_WithQueryString tests update with query string
func TestUpdateObject_WithQueryString(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"update3": {
			"id":   "update3",
			"name": "Original Name",
		},
	}
	svr := fakeserver.NewFakeServer(8089, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                "http://127.0.0.1:8089",
		Timeout:            2,
		WriteReturnsObject: true,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "update3", "name": "Updated Name"}`,
		QueryString: "version=2",
	})
	require.NoError(t, err)

	err = obj.UpdateObject(ctx)
	assert.NoError(t, err, "UpdateObject with query string should succeed")
}

// TestUpdateObject_WithoutWriteReturns tests update when write_returns_object is false
func TestUpdateObject_WithoutWriteReturns(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"update4": {
			"id":   "update4",
			"name": "Updated Name",
		},
	}
	svr := fakeserver.NewFakeServer(8090, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                "http://127.0.0.1:8090",
		Timeout:            2,
		WriteReturnsObject: false,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "update4", "name": "Updated Name"}`,
	})
	require.NoError(t, err)

	err = obj.UpdateObject(ctx)
	assert.NoError(t, err, "UpdateObject should succeed and fetch object")
}

// TestDeleteObject_Success tests successful object deletion
func TestDeleteObject_Success(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"delete1": {
			"id":   "delete1",
			"name": "To Be Deleted",
		},
	}
	svr := fakeserver.NewFakeServer(8091, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8091",
		Timeout: 2,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "delete1"}`,
	})
	require.NoError(t, err)

	err = obj.DeleteObject(ctx)
	assert.NoError(t, err, "DeleteObject should succeed")
}

// TestDeleteObject_NoID tests deletion when ID is empty (should be allowed)
func TestDeleteObject_NoID(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8092, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8092",
		Timeout: 2,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"name": "No ID"}`,
	})
	require.NoError(t, err)

	err = obj.DeleteObject(ctx)
	assert.NoError(t, err, "DeleteObject with no ID should return nil (no-op)")
}

// TestDeleteObject_WithQueryString tests deletion with query string
func TestDeleteObject_WithQueryString(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"delete2": {
			"id":   "delete2",
			"name": "To Be Deleted",
		},
	}
	svr := fakeserver.NewFakeServer(8093, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8093",
		Timeout: 2,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "delete2"}`,
		QueryString: "cascade=true",
	})
	require.NoError(t, err)

	err = obj.DeleteObject(ctx)
	assert.NoError(t, err, "DeleteObject with query string should succeed")
}

// TestDeleteObject_WithDestroyData tests deletion with destroy_data payload
func TestDeleteObject_WithDestroyData(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"delete3": {
			"id":   "delete3",
			"name": "To Be Deleted",
		},
	}
	svr := fakeserver.NewFakeServer(8094, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8094",
		Timeout: 2,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "delete3"}`,
		DestroyData: `{"reason": "test cleanup"}`,
	})
	require.NoError(t, err)

	err = obj.DeleteObject(ctx)
	assert.NoError(t, err, "DeleteObject with destroy_data should succeed")
}

// TestDeleteObject_NotFound tests deletion when object is already gone (404)
func TestDeleteObject_NotFound(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8095, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8095",
		Timeout: 2,
	})
	require.NoError(t, err)

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "nonexistent"}`,
	})
	require.NoError(t, err)

	err = obj.DeleteObject(ctx)
	assert.NoError(t, err, "DeleteObject should succeed when object not found (404)")
}
