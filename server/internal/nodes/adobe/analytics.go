package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Analytics — Adobe Analytics API (run reports, list segments/metrics/dimensions)
// via server-to-server IMS OAuth2. Uses the Analytics 2.0 API.
func Analytics(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	rsID := sp("rsid", "Report Suite ID", true)
	segmentID := sp("segmentId", "Segment ID", true)

	return oauth2Node(base,
		"adobe.analytics", "Adobe Analytics", "BarChart3",
		"Run reports, and list segments, metrics, and dimensions from Adobe Analytics.",
		[]rest.Op{
			{Resource: "report", Name: "run", Label: "Run Report", Method: "POST",
				Path: "/reports", BodyParam: "body",
				Params: []schema.ParamSchema{rsID, body}},
			{Resource: "report", Name: "topItems", Label: "Get Top Items", Method: "GET",
				Path: "/reports/topItems",
				Query: map[string]string{"rsid": "rsid"},
				Params: []schema.ParamSchema{rsID}},
			{Resource: "segment", Name: "list", Label: "List Segments", Method: "GET",
				Path: "/segments",
				Query: map[string]string{"rsid": "rsid"},
				Params: []schema.ParamSchema{rsID}},
			{Resource: "segment", Name: "get", Label: "Get Segment", Method: "GET",
				Path: "/segments/{segmentId}",
				Params: []schema.ParamSchema{rsID, segmentID}},
			{Resource: "metric", Name: "list", Label: "List Metrics", Method: "GET",
				Path: "/metrics",
				Query: map[string]string{"rsid": "rsid"},
				Params: []schema.ParamSchema{rsID}},
			{Resource: "dimension", Name: "list", Label: "List Dimensions", Method: "GET",
				Path: "/dimensions",
				Query: map[string]string{"rsid": "rsid"},
				Params: []schema.ParamSchema{rsID}},
			{Resource: "dimension", Name: "getItems", Label: "Get Dimension Items", Method: "GET",
				Path: "/dimensions/{dimensionId}",
				Params: []schema.ParamSchema{rsID, sp("dimensionId", "Dimension ID", true)}},
		})
}
