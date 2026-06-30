package azure

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// DevOps — Azure DevOps REST API (work items, pipelines, repos, pull requests).
// Auth: Personal Access Token (PAT) via azureDevOps credential with Basic auth.
// The base URL is overridable per organization (e.g. https://dev.azure.com/myorg).
func DevOps(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	project := sp("project", "Project name", true)
	wiID := ip("workItemId", "Work Item ID", 0)
	repoID := sp("repositoryId", "Repository ID", true)
	prID := ip("pullRequestId", "PR ID", 0)
	pipelineID := ip("pipelineId", "Pipeline ID", 0)
	branch := schema.ParamSchema{Name: "branch", Label: "Source branch", Type: "string"}
	wiType := schema.ParamSchema{Name: "type", Label: "Work Item Type", Type: "string", Default: "Task"}

	return rest.Node{
		Type: "azure.devops", Label: "Azure DevOps", Group: "integration", Icon: "GitBranch",
		Description:  "Manage Azure DevOps work items, pipelines, repositories, and pull requests.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "azureDevOps",
		Auth:         rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Basic ", ValueField: "accessToken"},
		Ops: []rest.Op{
			{Resource: "workItem", Name: "list", Label: "List Work Items (by query)", Method: "POST",
				Path: "/{project}/_apis/wit/wiql?api-version=7.1", BodyParam: "body",
				ItemsPath: "workItems",
				Params: []schema.ParamSchema{project, body}},
			{Resource: "workItem", Name: "get", Label: "Get Work Item", Method: "GET",
				Path: "/{project}/_apis/wit/workitems/{workItemId}?api-version=7.1",
				Params: []schema.ParamSchema{project, wiID}},
			{Resource: "workItem", Name: "create", Label: "Create Work Item", Method: "POST",
				Path: "/{project}/_apis/wit/workitems/${type}?api-version=7.1", BodyParam: "body",
				Params: []schema.ParamSchema{project, wiType, body}},
			{Resource: "workItem", Name: "update", Label: "Update Work Item", Method: "PATCH",
				Path: "/{project}/_apis/wit/workitems/{workItemId}?api-version=7.1", BodyParam: "body",
				Params: []schema.ParamSchema{project, wiID, body}},
			{Resource: "workItem", Name: "delete", Label: "Delete Work Item", Method: "DELETE",
				Path: "/{project}/_apis/wit/workitems/{workItemId}?api-version=7.1",
				Params: []schema.ParamSchema{project, wiID}},
			{Resource: "pipeline", Name: "list", Label: "List Pipelines", Method: "GET",
				Path: "/{project}/_apis/pipelines?api-version=7.1", ItemsPath: "value",
				Params: []schema.ParamSchema{project}},
			{Resource: "pipeline", Name: "get", Label: "Get Pipeline", Method: "GET",
				Path: "/{project}/_apis/pipelines/{pipelineId}?api-version=7.1",
				Params: []schema.ParamSchema{project, pipelineID}},
			{Resource: "pipeline", Name: "run", Label: "Run Pipeline", Method: "POST",
				Path: "/{project}/_apis/pipelines/{pipelineId}/runs?api-version=7.1", BodyParam: "body",
				Params: []schema.ParamSchema{project, pipelineID, branch, body}},
			{Resource: "repo", Name: "list", Label: "List Repositories", Method: "GET",
				Path: "/{project}/_apis/git/repositories?api-version=7.1", ItemsPath: "value",
				Params: []schema.ParamSchema{project}},
			{Resource: "repo", Name: "get", Label: "Get Repository", Method: "GET",
				Path: "/{project}/_apis/git/repositories/{repositoryId}?api-version=7.1",
				Params: []schema.ParamSchema{project, repoID}},
			{Resource: "pullRequest", Name: "list", Label: "List Pull Requests", Method: "GET",
				Path: "/{project}/_apis/git/pullrequests?api-version=7.1", ItemsPath: "value",
				Params: []schema.ParamSchema{project}},
			{Resource: "pullRequest", Name: "get", Label: "Get Pull Request", Method: "GET",
				Path: "/{project}/_apis/git/repositories/{repositoryId}/pullrequests/{pullRequestId}?api-version=7.1",
				Params: []schema.ParamSchema{project, repoID, prID}},
			{Resource: "pullRequest", Name: "create", Label: "Create Pull Request", Method: "POST",
				Path: "/{project}/_apis/git/repositories/{repositoryId}/pullrequests?api-version=7.1", BodyParam: "body",
				Params: []schema.ParamSchema{project, repoID, body}},
		},
	}
}
