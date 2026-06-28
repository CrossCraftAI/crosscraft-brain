package azure

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// PowerBI — Azure Power BI REST API (datasets, reports, dashboards).
// Auth: OAuth2 Bearer token via azurePowerBI credential.
func PowerBI(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	datasetID := sp("datasetId", "Dataset ID", true)
	tableName := schema.ParamSchema{Name: "tableName", Label: "Table name", Type: "string"}

	return rest.Node{
		Type: "azure.powerbi", Label: "Power BI", Group: "integration", Icon: "BarChart3",
		Description:  "Manage Power BI datasets, push rows, refresh datasets, and list reports/dashboards.",
		BaseURL:      base,
		CredType:     "azurePowerBI",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			{Resource: "dataset", Name: "list", Label: "List Datasets", Method: "GET",
				Path: "/datasets", ItemsPath: "value"},
			{Resource: "dataset", Name: "get", Label: "Get Dataset", Method: "GET",
				Path: "/datasets/{datasetId}",
				Params: []schema.ParamSchema{datasetID}},
			{Resource: "dataset", Name: "pushRows", Label: "Push Rows to Dataset Table", Method: "POST",
				Path: "/datasets/{datasetId}/tables/{tableName}/rows", BodyParam: "body",
				Params: []schema.ParamSchema{datasetID, tableName, body}},
			{Resource: "dataset", Name: "refresh", Label: "Refresh Dataset", Method: "POST",
				Path: "/datasets/{datasetId}/refreshes",
				Params: []schema.ParamSchema{datasetID}},
			{Resource: "report", Name: "list", Label: "List Reports", Method: "GET",
				Path: "/reports", ItemsPath: "value"},
			{Resource: "report", Name: "get", Label: "Get Report", Method: "GET",
				Path: "/reports/{reportId}",
				Params: []schema.ParamSchema{sp("reportId", "Report ID", true)}},
			{Resource: "dashboard", Name: "list", Label: "List Dashboards", Method: "GET",
				Path: "/dashboards", ItemsPath: "value"},
			{Resource: "dashboard", Name: "get", Label: "Get Dashboard", Method: "GET",
				Path: "/dashboards/{dashboardId}",
				Params: []schema.ParamSchema{sp("dashboardId", "Dashboard ID", true)}},
			{Resource: "group", Name: "list", Label: "List Groups (Workspaces)", Method: "GET",
				Path: "/groups", ItemsPath: "value"},
		},
	}
}
