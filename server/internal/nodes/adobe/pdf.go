package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// PDFServices — Adobe PDF Services API (create, export, OCR, compress, combine,
// split, extract, and document generation via server-to-server IMS OAuth2).
func PDFServices(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	assetID := sp("assetId", "Asset ID", true)
	jobID := sp("jobId", "Job ID", true)
	binaryProp := schema.ParamSchema{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"}
	outputProp := schema.ParamSchema{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"}

	return oauth2Node(base,
		"adobe.pdfServices", "Adobe PDF Services", "FileText",
		"Create, export, OCR, compress, combine, split, and extract PDFs.",
		[]rest.Op{
			{Resource: "pdf", Name: "create", Label: "Create PDF (from Office/HTML)", Method: "POST",
				Path: "/operation/createpdf", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp}},
			{Resource: "pdf", Name: "export", Label: "Export PDF (to Office/JPEG)", Method: "POST",
				Path: "/operation/exportpdf", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp}},
			{Resource: "pdf", Name: "ocr", Label: "OCR PDF", Method: "POST",
				Path: "/operation/ocr", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp}},
			{Resource: "pdf", Name: "compress", Label: "Compress PDF", Method: "POST",
				Path: "/operation/compresspdf", BodyParam: "body",
				Params: []schema.ParamSchema{body, binaryProp}},
			{Resource: "pdf", Name: "combine", Label: "Combine PDFs", Method: "POST",
				Path: "/operation/combinepdf", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "pdf", Name: "split", Label: "Split PDF", Method: "POST",
				Path: "/operation/splitpdf", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "pdf", Name: "extract", Label: "Extract Info (text/tables)", Method: "POST",
				Path: "/operation/extractpdf", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "pdf", Name: "documentGeneration", Label: "Document Generation", Method: "POST",
				Path: "/operation/documentgeneration", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Job management
			{Resource: "job", Name: "getStatus", Label: "Get Job Status", Method: "GET",
				Path: "/operation/status/{jobId}", Params: []schema.ParamSchema{jobID}},
			{Resource: "job", Name: "download", Label: "Download Result", Method: "GET",
				Path: "/operation/download/{assetId}",
				Params: []schema.ParamSchema{assetID, outputProp}},
		})
}
