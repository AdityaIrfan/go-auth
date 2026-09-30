package docs

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestGeneratedSwaggerContract(t *testing.T) {
	var served map[string]interface{}
	if err := json.Unmarshal([]byte(SwaggerInfo.ReadDoc()), &served); err != nil {
		t.Fatalf("SwaggerInfo.ReadDoc() returned invalid JSON: %v", err)
	}

	jsonFile, err := os.ReadFile("swagger.json")
	if err != nil {
		t.Fatalf("read swagger.json: %v", err)
	}
	var jsonSpec map[string]interface{}
	if err := json.Unmarshal(jsonFile, &jsonSpec); err != nil {
		t.Fatalf("swagger.json is invalid: %v", err)
	}
	// swag injects an empty host at runtime, while the generated JSON omits it.
	if served["host"] == "" {
		delete(served, "host")
	}
	if jsonSpec["host"] == "" {
		delete(jsonSpec, "host")
	}
	if !reflect.DeepEqual(served, jsonSpec) {
		t.Fatal("served Swagger document differs from docs/swagger.json")
	}

	yamlFile, err := os.ReadFile("swagger.yaml")
	if err != nil {
		t.Fatalf("read swagger.yaml: %v", err)
	}
	yamlJSON, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		t.Fatalf("swagger.yaml is invalid: %v", err)
	}
	var yamlSpec map[string]interface{}
	if err := json.Unmarshal(yamlJSON, &yamlSpec); err != nil {
		t.Fatalf("converted swagger.yaml is invalid: %v", err)
	}
	if !reflect.DeepEqual(jsonSpec, yamlSpec) {
		t.Fatal("swagger.json and swagger.yaml are not equivalent")
	}

	paths := object(t, jsonSpec, "paths")
	wantPaths := map[string][]string{
		"/api/v1/auth/refresh":         {"post"},
		"/api/v1/events":               {"post", "get"},
		"/api/v1/events/{id}":          {"put", "delete"},
		"/health":                      {"get"},
		"/ready":                       {"get"},
		"/api/v1/auth/register":        {"post"},
		"/api/v1/auth/login":           {"post"},
		"/api/v1/auth/google/login":    {"get"},
		"/api/v1/auth/google/callback": {"get"},
		"/api/v1/auth/logout":          {"post"},
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("documented paths = %d, want %d: %#v", len(paths), len(wantPaths), paths)
	}
	for path, methods := range wantPaths {
		pathItem := object(t, paths, path)
		for _, method := range methods {
			if _, ok := pathItem[method]; !ok {
				t.Errorf("missing operation %s %s", method, path)
			}
		}
	}

	assertResponseCodes(t, paths, "/api/v1/auth/register", "post", "201", "400", "422")
	assertResponseCodes(t, paths, "/api/v1/auth/login", "post", "200", "400", "401", "422")
	assertResponseCodes(t, paths, "/api/v1/auth/google/login", "get", "307")
	assertResponseCodes(t, paths, "/api/v1/auth/google/callback", "get", "200", "401", "422", "500")
	assertResponseCodes(t, paths, "/api/v1/auth/logout", "post", "200", "401", "422")
	assertResponseCodes(t, paths, "/health", "get", "200")
	assertResponseCodes(t, paths, "/ready", "get", "200", "503")

	assertResponseCodes(t, paths, "/api/v1/auth/refresh", "post", "200", "400", "422", "500")
	assertResponseCodes(t, paths, "/api/v1/events", "post", "201", "400", "401", "422")
	assertResponseCodes(t, paths, "/api/v1/events", "get", "200", "400", "401", "422")
	assertResponseCodes(t, paths, "/api/v1/events/{id}", "put", "200", "400", "401", "404", "422")
	assertResponseCodes(t, paths, "/api/v1/events/{id}", "delete", "200", "400", "401", "404", "422")
	for _, route := range []struct{ path, method string }{
		{"/api/v1/auth/logout", "post"}, {"/api/v1/events", "post"}, {"/api/v1/events", "get"}, {"/api/v1/events/{id}", "put"}, {"/api/v1/events/{id}", "delete"},
	} {
		operation := object(t, object(t, paths, route.path), route.method)
		security, ok := operation["security"].([]interface{})
		if !ok || len(security) != 1 {
			t.Fatalf("missing security for %s %s", route.method, route.path)
		}
		if _, ok := security[0].(map[string]interface{})["BearerAuth"]; !ok {
			t.Fatal("missing BearerAuth")
		}
	}
	definitions := object(t, jsonSpec, "definitions")
	assertExample(t, definitions, "docs.RegisterErrorResponse", "message", "register failed")
	assertExample(t, definitions, "docs.LoginErrorResponse", "message", "login failed")
	assertExample(t, definitions, "docs.LogoutErrorResponse", "message", "logout failed")
	assertExample(t, definitions, "docs.RegisterSuccessResponse", "message", "register success, do login")
	assertExample(t, definitions, "docs.LoginSuccessResponse", "message", "login success")
	assertExample(t, definitions, "docs.LogoutSuccessResponse", "message", "logout success")
}

func object(t *testing.T, parent map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := parent[key].(map[string]interface{})
	if !ok {
		t.Fatalf("%q is missing or is not an object", key)
	}
	return value
}

func assertResponseCodes(t *testing.T, paths map[string]interface{}, path, method string, want ...string) {
	t.Helper()
	operation := object(t, object(t, paths, path), method)
	responses := object(t, operation, "responses")
	if len(responses) != len(want) {
		t.Errorf("%s %s response codes = %#v, want %v", method, path, responses, want)
		return
	}
	for _, code := range want {
		if _, ok := responses[code]; !ok {
			t.Errorf("%s %s is missing response %s", method, path, code)
		}
	}
}

func assertExample(t *testing.T, definitions map[string]interface{}, definition, property string, want interface{}) {
	t.Helper()
	properties := object(t, object(t, definitions, definition), "properties")
	propertySchema := object(t, properties, property)
	if got := propertySchema["example"]; !reflect.DeepEqual(got, want) {
		t.Errorf("%s.%s example = %#v, want %#v", definition, property, got, want)
	}
}

func TestRequestRequiredFields(t *testing.T) {
	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatal(err)
	}
	definitions := object(t, spec, "definitions")
	for name, fields := range map[string][]string{
		"RegisterRequest": {"email", "name", "password"},
		"LoginRequest":    {"email", "password"},
		"RefreshRequest":  {"refresh_token"},
		"EventRequest":    {"end_time", "start_time", "title"},
	} {
		schema := object(t, definitions, "docs."+name)
		got, ok := schema["required"].([]interface{})
		if !ok || len(got) != len(fields) {
			t.Fatalf("%s required=%v", name, schema["required"])
		}
		for i, field := range fields {
			if got[i] != field {
				t.Errorf("%s required=%v", name, got)
			}
		}
	}
	if fields := object(t, definitions, "docs.ListEventRequest")["required"]; fields != nil {
		t.Fatalf("list filters are optional, got required=%v", fields)
	}
	token := object(t, object(t, definitions, "docs.TokenData"), "properties")
	for _, name := range []string{"access_token", "refresh_token", "token_type", "expires_in"} {
		object(t, token, name)
	}
}
