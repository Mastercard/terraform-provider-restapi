// Simple test server that returns additional fields beyond what was sent
// Simulates PATCH-based API like IBM Cloud Resource Controller
// Run with: go run server.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

var objects = make(map[string]map[string]any)

func main() {
	http.HandleFunc("/api/objects/", handleObject)

	fmt.Println("Starting server on :8080...")
	fmt.Println("This server simulates a PATCH-based API that returns additional fields")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleObject(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/objects/")
	if id == "" {
		http.Error(w, "ID required in path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "PATCH":
		var data map[string]any
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Merge with existing or create new
		existing, exists := objects[id]
		if !exists {
			existing = make(map[string]any)
		}

		// Merge user data
		for k, v := range data {
			existing[k] = v
		}

		// Server adds MANY additional fields (simulating IBM Cloud API behavior)
		existing["id"] = id
		existing["guid"] = id
		existing["crn"] = fmt.Sprintf("crn:v1:bluemix:public:service:region:account:%s::", id)
		existing["created_at"] = time.Now().UTC().Format(time.RFC3339)
		existing["updated_at"] = time.Now().UTC().Format(time.RFC3339)
		existing["created_by"] = "user@example.com"
		existing["updated_by"] = "user@example.com"
		existing["account_id"] = "abc123def456"
		existing["resource_group_id"] = "rg-12345"
		existing["resource_group_crn"] = "crn:v1:bluemix:public:resource-group:global:account:abc123::"
		existing["name"] = "test-resource"
		existing["state"] = "active"
		existing["type"] = "service_instance"
		existing["sub_type"] = "enterprise"
		existing["allow_cleanup"] = false
		existing["locked"] = false
		existing["deleted_at"] = nil
		existing["deleted_by"] = ""
		existing["scheduled_reclaim_at"] = nil
		existing["restored_at"] = nil
		existing["restored_by"] = ""
		existing["resource_plan_id"] = "plan-12345"
		existing["resource_bindings_url"] = fmt.Sprintf("/v2/resource_instances/%s/bindings", id)
		existing["resource_keys_url"] = fmt.Sprintf("/v2/resource_instances/%s/keys", id)
		existing["dashboard_url"] = fmt.Sprintf("https://cloud.example.com/%s", id)
		existing["last_operation"] = map[string]any{
			"type":        "update",
			"state":       "succeeded",
			"async":       false,
			"description": "Operation completed",
		}
		existing["extensions"] = map[string]any{
			"external_dashboard": fmt.Sprintf("https://external.example.com/%s", id),
			"virtual_private_endpoints": map[string]any{
				"dns_domain": "service.cloud.example.com",
				"dns_hosts":  []string{"private"},
				"endpoints": []map[string]any{
					{"ip_address": "10.0.0.1", "zone": "zone-1"},
					{"ip_address": "10.0.0.2", "zone": "zone-2"},
				},
			},
		}

		objects[id] = existing

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)

	case "GET":
		obj, ok := objects[id]
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj)

	case "DELETE":
		delete(objects, id)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
