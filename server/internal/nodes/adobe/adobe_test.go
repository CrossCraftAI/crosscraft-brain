package adobe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// TestAcrobatSignListWithBaseOverride checks header (Bearer) auth, the ItemsPath
// mapping, and the per-node baseUrl override.
func TestAcrobatSignListWithBaseOverride(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.Method == http.MethodGet && r.URL.Path == "/agreements" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"userAgreementList": []any{map[string]any{"id": "a1"}, map[string]any{"id": "a2"}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := AcrobatSign("https://api.na1.adobesign.com/api/rest/v6").Build()
	ctx := &schema.ExecContext{
		Params:     map[string]any{"operation": "agreement:list", "baseUrl": srv.URL, "credential": "c1"},
		RawParam:   func(n string) any { return nil },
		Credential: func(string) (map[string]any, error) { return map[string]any{"accessToken": "KEY123"}, nil },
		Log:        func(string, any) {},
	}
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer KEY123" {
		t.Fatalf("auth header: %q", gotAuth)
	}
	if out := res.Outputs["main"]; len(out) != 2 || out[0].JSON["id"] != "a1" {
		t.Fatalf("agreements: %+v", res.Outputs["main"])
	}
}

func TestAcrobatSignEnhanced(t *testing.T) {
	node := AcrobatSign("https://api.test.sign.com/api/rest/v6")
	def := node.Build()
	ops := collectOps(def)
	if len(ops) < 10 {
		t.Fatalf("expected at least 10 ops, got %d", len(ops))
	}
	for _, want := range []string{"agreement:send", "agreement:cancel", "agreement:getDocuments", "reminder:send", "webhook:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q in enhanced Sign node", want)
		}
	}
}

func TestPDFServicesNode(t *testing.T) {
	def := PDFServices("https://pdf.services.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"pdf:create", "pdf:export", "pdf:ocr", "pdf:compress", "pdf:combine", "pdf:split", "pdf:extract", "pdf:documentGeneration"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestFireflyNode(t *testing.T) {
	def := Firefly("https://firefly.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"image:generate", "image:fill", "image:expand", "image:upscale"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestPhotoshopNode(t *testing.T) {
	def := Photoshop("https://image.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"image:applyEdits", "image:smartObject", "image:runAction", "image:createRendition"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestLightroomNode(t *testing.T) {
	def := Lightroom("https://lr.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"image:autoTone", "image:applyPreset", "image:edit", "image:getRendition"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestAEMAssetsNode(t *testing.T) {
	def := AEMAssets("https://aem.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"asset:upload", "asset:get", "asset:updateMetadata", "asset:delete", "folder:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestAnalyticsNode(t *testing.T) {
	def := Analytics("https://analytics.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"report:run", "segment:list", "metric:list", "dimension:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestStockNode(t *testing.T) {
	def := Stock("https://stock.test.io").Build()
	ops := collectOps(def)
	for _, want := range []string{"asset:search", "asset:get", "asset:license", "asset:download"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestCommerceNode(t *testing.T) {
	def := Commerce("https://magento.test.io").Build()
	ops := collectOps(def)
	if len(ops) < 12 {
		t.Fatalf("expected at least 12 ops, got %d", len(ops))
	}
	for _, want := range []string{"customer:list", "customer:create", "product:list", "order:get", "invoice:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestNodesReturnsAll(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 9 {
		t.Fatalf("expected 9 Adobe nodes, got %d", len(nodes))
	}
	names := map[string]bool{}
	for _, n := range nodes {
		names[n.Type] = true
	}
	for _, want := range []string{
		"adobe.acrobatSign", "adobe.pdfServices", "adobe.firefly",
		"adobe.photoshop", "adobe.lightroom", "adobe.aemAssets",
		"adobe.analytics", "adobe.stock", "adobe.commerce",
	} {
		if !names[want] {
			t.Fatalf("expected node %q to be registered", want)
		}
	}
}

// collectOps extracts operation keys from a built NodeDefinition.
func collectOps(def schema.NodeDefinition) map[string]bool {
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}
	return ops
}
