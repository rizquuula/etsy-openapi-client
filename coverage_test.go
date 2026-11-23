package etsy_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	etsy "github.com/rizquuula/etsy-openapi-client"
)

func TestGeneratedClientCoversEveryOperation(t *testing.T) {
	data, err := os.ReadFile("api/openapi.json")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	clientType := reflect.TypeOf((*etsy.Client)(nil))
	methods := make(map[string]struct{}, clientType.NumMethod())
	for i := 0; i < clientType.NumMethod(); i++ {
		methods[clientType.Method(i).Name] = struct{}{}
	}

	operations := 0
	for path, item := range spec.Paths {
		for method, raw := range item {
			switch method {
			case "get", "post", "put", "patch", "delete":
			default:
				continue
			}
			var operation struct {
				OperationID string `json:"operationId"`
			}
			if err := json.Unmarshal(raw, &operation); err != nil {
				t.Fatalf("decode %s %s: %v", method, path, err)
			}
			if operation.OperationID == "" {
				t.Errorf("%s %s has no operationId", method, path)
				continue
			}

			operations++
			name := strings.ToUpper(operation.OperationID[:1]) + operation.OperationID[1:]
			candidates := []string{
				name + "WithResponse",
				name + "WithBodyWithResponse",
				name + "WithFormdataBodyWithResponse",
			}
			found := false
			for _, candidate := range candidates {
				if _, ok := methods[candidate]; ok {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s %s (%s): no generated method", method, path, operation.OperationID)
			}
		}
	}

	if operations == 0 {
		t.Fatal("spec contains no operations")
	}
	t.Logf("covered %d operations", operations)
}

func TestSpecState(t *testing.T) {
	// Guards the vendored spec against accidental truncation.
	data, err := os.ReadFile("api/openapi.json")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec struct {
		OpenAPI string                     `json:"openapi"`
		Paths   map[string]json.RawMessage `json:"paths"`
		Info    struct {
			Title string `json:"title"`
		} `json:"info"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	if spec.OpenAPI != "3.0.2" {
		t.Errorf("openapi = %q, want 3.0.2", spec.OpenAPI)
	}
	if spec.Info.Title != "Etsy Open API v3" {
		t.Errorf("title = %q, want Etsy Open API v3", spec.Info.Title)
	}
	if len(spec.Paths) != 77 {
		t.Errorf("paths = %d, want 77", len(spec.Paths))
	}
}

func TestClientExposesHTTPMethods(t *testing.T) {
	clientType := reflect.TypeOf((*etsy.Client)(nil))
	for _, name := range []string{
		"GetShopWithResponse",
		"GetListingWithResponse",
		"GetMeWithResponse",
		"CreateDraftListingWithBodyWithResponse",
		"UploadListingImageWithBodyWithResponse",
	} {
		if _, ok := clientType.MethodByName(name); !ok {
			t.Errorf("missing generated method %s", name)
		}
	}
}
