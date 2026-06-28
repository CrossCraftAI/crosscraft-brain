package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Photoshop — Adobe Photoshop API (apply edits, smart object replacement, run
// actions, create renditions) via server-to-server IMS OAuth2.
func Photoshop(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	binaryProp := schema.ParamSchema{Name: "binaryProperty", Label: "Input binary property", Type: "string", Default: "data"}
	outputProp := schema.ParamSchema{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"}
	jobID := sp("jobId", "Job ID", true)

	return oauth2Node(base,
		"adobe.photoshop", "Adobe Photoshop", "Image",
		"Apply Photoshop edits, replace smart objects, run actions, and create renditions.",
		[]rest.Op{
			{Resource: "image", Name: "applyEdits", Label: "Apply Edits", Method: "POST",
				Path: "/pie/psdService/edit", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp, outputProp}},
			{Resource: "image", Name: "smartObject", Label: "Replace Smart Object", Method: "POST",
				Path: "/pie/psdService/smartObject", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp, outputProp}},
			{Resource: "image", Name: "runAction", Label: "Run Photoshop Action", Method: "POST",
				Path: "/pie/psdService/photoshopActions", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp, outputProp}},
			{Resource: "image", Name: "createRendition", Label: "Create Rendition", Method: "POST",
				Path: "/pie/psdService/rendition", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp, outputProp}},
			// Job management
			{Resource: "job", Name: "getStatus", Label: "Get Job Status", Method: "GET",
				Path: "/pie/psdService/status/{jobId}",
				Params: []schema.ParamSchema{jobID}},
			{Resource: "job", Name: "getDocumentManifest", Label: "Get Document Manifest", Method: "GET",
				Path: "/pie/psdService/documentManifest", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
		})
}
