package provider

import (
	"context"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeAPIObject_ReadSearch(t *testing.T) {
	ctx := context.Background()

	client, err := apiclient.NewAPIClient(&apiclient.APIClientOpt{
		URI:     "http://localhost:8080",
		Timeout: 2,
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		model          *RestAPIObjectResourceModel
		expectedSearch map[string]string
	}{
		{
			name: "nil_read_search",
			model: &RestAPIObjectResourceModel{
				Path:       types.StringValue("/api/objects"),
				Data:       jsontypes.NewNormalizedValue(`{"id":"123"}`),
				ReadSearch: nil,
			},
			expectedSearch: nil,
		},
		{
			name: "read_search_with_required_fields_only",
			model: &RestAPIObjectResourceModel{
				Path: types.StringValue("/api/objects"),
				Data: jsontypes.NewNormalizedValue(`{"id":"123"}`),
				ReadSearch: &ReadSearchModel{
					SearchKey:   types.StringValue("email"),
					SearchValue: types.StringValue("test@example.com"),
				},
			},
			expectedSearch: map[string]string{
				"search_key":   "email",
				"search_value": "test@example.com",
			},
		},
		{
			name: "read_search_with_all_fields",
			model: &RestAPIObjectResourceModel{
				Path: types.StringValue("/api/objects"),
				Data: jsontypes.NewNormalizedValue(`{"id":"123"}`),
				ReadSearch: &ReadSearchModel{
					SearchKey:   types.StringValue("email"),
					SearchValue: types.StringValue("test@example.com"),
					ResultsKey:  types.StringValue("data/users"),
					QueryString: types.StringValue("status=active"),
					SearchData:  jsontypes.NewNormalizedValue(`{"filter":"active"}`),
				},
			},
			expectedSearch: map[string]string{
				"search_key":   "email",
				"search_value": "test@example.com",
				"results_key":  "data/users",
				"query_string": "status=active",
				"search_data":  `{"filter":"active"}`,
			},
		},
		{
			name: "read_search_with_partial_optional_fields",
			model: &RestAPIObjectResourceModel{
				Path: types.StringValue("/api/objects"),
				Data: jsontypes.NewNormalizedValue(`{"id":"123"}`),
				ReadSearch: &ReadSearchModel{
					SearchKey:   types.StringValue("name"),
					SearchValue: types.StringValue("John Doe"),
					ResultsKey:  types.StringValue("results"),
				},
			},
			expectedSearch: map[string]string{
				"search_key":   "name",
				"search_value": "John Doe",
				"results_key":  "results",
			},
		},
		{
			name: "read_search_with_id_placeholder",
			model: &RestAPIObjectResourceModel{
				Path: types.StringValue("/api/objects"),
				Data: jsontypes.NewNormalizedValue(`{"id":"test-123"}`),
				ReadSearch: &ReadSearchModel{
					SearchKey:   types.StringValue("objectId"),
					SearchValue: types.StringValue("{id}"),
				},
			},
			expectedSearch: map[string]string{
				"search_key":   "objectId",
				"search_value": "{id}", // Not yet substituted at this stage
			},
		},
		{
			name: "read_search_with_search_patch",
			model: &RestAPIObjectResourceModel{
				Path: types.StringValue("/api/objects"),
				Data: jsontypes.NewNormalizedValue(`{"id":"123"}`),
				ReadSearch: &ReadSearchModel{
					SearchKey:   types.StringValue("email"),
					SearchValue: types.StringValue("test@example.com"),
					SearchPatch: jsontypes.NewNormalizedValue(`[{"op":"move","from":"/data","path":"/"}]`),
				},
			},
			expectedSearch: map[string]string{
				"search_key":   "email",
				"search_value": "test@example.com",
				"search_patch": `[{"op":"move","from":"/data","path":"/"}]`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := makeAPIObject(ctx, client, "test-id", tt.model)
			require.NoError(t, err, "makeAPIObject should not return an error")
			require.NotNil(t, obj, "makeAPIObject should return an object")

			if tt.expectedSearch == nil {
				assert.Nil(t, obj.GetReadSearch(), "ReadSearch should be nil when not configured")
			} else {
				readSearch := obj.GetReadSearch()
				require.NotNil(t, readSearch, "ReadSearch should not be nil")
				assert.Equal(t, len(tt.expectedSearch), len(readSearch), "ReadSearch should have correct number of fields")

				for key, expectedValue := range tt.expectedSearch {
					actualValue, exists := readSearch[key]
					assert.True(t, exists, "ReadSearch should contain key: %s", key)
					assert.Equal(t, expectedValue, actualValue, "ReadSearch[%s] should match", key)
				}
			}
		})
	}
}
