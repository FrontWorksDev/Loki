package api

import (
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

// newTestServer は OpenAPI スペック検証用の Server を生成する。
// 実際にはネットワーク listen はしない。
func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(DefaultConfig())
}

func TestOpenAPI_InfoMetadata(t *testing.T) {
	srv := newTestServer(t)
	spec := srv.API().OpenAPI()

	if spec.Info == nil {
		t.Fatal("OpenAPI Info is nil")
	}

	if got, want := spec.Info.Title, "Loki Image API"; got != want {
		t.Errorf("Info.Title = %q, want %q", got, want)
	}

	if spec.Info.Version == "" {
		t.Error("Info.Version is empty")
	}

	desc := spec.Info.Description
	if desc == "" {
		t.Fatal("Info.Description is empty")
	}
	for _, kw := range []string{"画像", "JPEG", "PNG", "WebP"} {
		if !strings.Contains(desc, kw) {
			t.Errorf("Info.Description missing keyword %q", kw)
		}
	}

	if spec.Info.Contact == nil {
		t.Fatal("Info.Contact is nil")
	}
	if spec.Info.Contact.Name == "" || spec.Info.Contact.URL == "" {
		t.Errorf("Info.Contact.Name=%q URL=%q both must be non-empty",
			spec.Info.Contact.Name, spec.Info.Contact.URL)
	}

	if spec.Info.License == nil {
		t.Fatal("Info.License is nil")
	}
	if spec.Info.License.Name == "" || spec.Info.License.URL == "" {
		t.Errorf("Info.License.Name=%q URL=%q both must be non-empty",
			spec.Info.License.Name, spec.Info.License.URL)
	}
}

func TestOpenAPI_ImageEndpointsHaveCommonErrorResponses(t *testing.T) {
	srv := newTestServer(t)
	spec := srv.API().OpenAPI()

	endpoints := []string{"/api/v1/compress", "/api/v1/convert"}
	codes := []string{"400", "413", "422", "429", "500"}

	for _, path := range endpoints {
		t.Run(path, func(t *testing.T) {
			pi, ok := spec.Paths[path]
			if !ok || pi == nil || pi.Post == nil {
				t.Fatalf("path %q POST operation not found", path)
			}
			op := pi.Post
			if op.Responses == nil {
				t.Fatalf("path %q POST has nil Responses", path)
			}
			for _, code := range codes {
				resp, ok := op.Responses[code]
				if !ok || resp == nil {
					t.Errorf("path %q: missing response for status %s", path, code)
					continue
				}
				if _, ok := resp.Content["application/problem+json"]; !ok {
					var keys []string
					for k := range resp.Content {
						keys = append(keys, k)
					}
					t.Errorf("path %q response %s: missing application/problem+json (got %v)",
						path, code, keys)
				}
			}
		})
	}
}

func TestOpenAPI_HealthExcludedFromCommonErrors(t *testing.T) {
	srv := newTestServer(t)
	spec := srv.API().OpenAPI()

	pi, ok := spec.Paths["/api/v1/health"]
	if !ok || pi == nil || pi.Get == nil {
		t.Fatal("/api/v1/health GET operation not found")
	}
	op := pi.Get

	for _, code := range []string{"400", "413", "422", "429", "500"} {
		if _, exists := op.Responses[code]; exists {
			t.Errorf("/api/v1/health should not declare common error response %s", code)
		}
	}
}

func TestOpenAPI_CompressRequestExamples(t *testing.T) {
	srv := newTestServer(t)
	spec := srv.API().OpenAPI()

	schema := requireMultipartSchema(t, spec, "/api/v1/compress")
	requireFieldExample(t, schema, "quality")
	requireFieldExample(t, schema, "level")
}

func TestOpenAPI_ConvertRequestExamples(t *testing.T) {
	srv := newTestServer(t)
	spec := srv.API().OpenAPI()

	schema := requireMultipartSchema(t, spec, "/api/v1/convert")
	for _, field := range []string{"format", "quality", "level"} {
		requireFieldExample(t, schema, field)
	}
}

// requireMultipartSchema は multipart/form-data リクエストボディのスキーマを取得する。
// $ref で参照されている場合は components/schemas から解決する。
func requireMultipartSchema(t *testing.T, spec *huma.OpenAPI, path string) *huma.Schema {
	t.Helper()
	pi, ok := spec.Paths[path]
	if !ok || pi == nil || pi.Post == nil {
		t.Fatalf("path %q POST not found", path)
	}
	body := pi.Post.RequestBody
	if body == nil || body.Content == nil {
		t.Fatalf("path %q POST: nil RequestBody/Content", path)
	}
	mt, ok := body.Content["multipart/form-data"]
	if !ok || mt.Schema == nil {
		t.Fatalf("path %q POST: missing multipart/form-data schema", path)
	}
	return resolveSchema(t, spec, mt.Schema)
}

// resolveSchema は schema が $ref を含む場合 components から解決する。
func resolveSchema(t *testing.T, spec *huma.OpenAPI, s *huma.Schema) *huma.Schema {
	t.Helper()
	if s.Ref == "" {
		return s
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(s.Ref, prefix) {
		t.Fatalf("unsupported $ref %q", s.Ref)
	}
	name := strings.TrimPrefix(s.Ref, prefix)
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Fatalf("components.schemas not initialized for $ref %q", s.Ref)
	}
	resolved := spec.Components.Schemas.Map()[name]
	if resolved == nil {
		t.Fatalf("schema %q not found in components", name)
	}
	return resolved
}

// requireFieldExample は schema.Properties[field] に example が設定されていることを検証する。
// Huma は struct タグ `example:"…"` を Schema.Examples ([]any) に格納する。
func requireFieldExample(t *testing.T, schema *huma.Schema, field string) {
	t.Helper()
	if schema.Properties == nil {
		t.Fatalf("schema has no Properties")
	}
	prop, ok := schema.Properties[field]
	if !ok {
		t.Fatalf("field %q not found in schema properties", field)
	}
	if len(prop.Examples) == 0 {
		t.Errorf("field %q missing example (Schema.Examples is empty)", field)
	}
}
