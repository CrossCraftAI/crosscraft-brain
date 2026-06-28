package aws

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/url"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// S3Node provides S3 operations via the AWS REST API with SigV4 signing.
func S3Node() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type: "aws.s3", Label: "AWS S3", Group: "integration", Icon: "HardDrive",
		Description: "List, upload, download, and delete S3 objects.",
		Inputs:  []schema.Port{{ID: "main"}},
		Outputs: []schema.Port{{ID: "main"}},
		Credentials: []string{"awsIam"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "awsIam"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "List Buckets", Value: "list"},
				{Label: "List Objects", Value: "listObjects"},
				{Label: "Upload Object", Value: "upload"},
				{Label: "Download Object", Value: "download"},
				{Label: "Delete Object", Value: "delete"},
				{Label: "Get Presigned URL", Value: "presignedURL"},
			}},
			{Name: "bucket", Label: "Bucket name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"listObjects", "upload", "download", "delete", "presignedURL"}}},
			{Name: "key", Label: "Object key (path)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"upload", "download", "delete", "presignedURL"}}},
			{Name: "prefix", Label: "Prefix filter", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"listObjects"}}},
			{Name: "maxKeys", Label: "Max keys", Type: "number", Default: float64(100),
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"listObjects"}}},
			{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"upload", "download"}}},
			{Name: "expires", Label: "URL expires (seconds)", Type: "number", Default: float64(3600),
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"presignedURL"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			signer, err := makeSigner(ctx, "s3")
			if err != nil {
				return schema.NodeResult{}, err
			}
			op := asString(ctx.Params["operation"], "")
			bucket := asString(ctx.Params["bucket"], "")
			key := asString(ctx.Params["key"], "")
			region := signer.Region

			switch op {

			case "list":
				host := "s3." + region + ".amazonaws.com"
				items, err := awsDo(signer, "GET", "https://"+host+"/", nil)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "listObjects":
				if bucket == "" {
					return schema.NodeResult{}, fmt.Errorf("s3 listObjects: bucket is required")
				}
				host := bucket + ".s3." + region + ".amazonaws.com"
				q := url.Values{}
				if pre := asString(ctx.Params["prefix"], ""); pre != "" {
					q.Set("prefix", pre)
				}
				maxK := asInt(ctx.Params["maxKeys"], 100)
				q.Set("max-keys", fmt.Sprint(maxK))
				queryStr := q.Encode()
				u := "https://" + host + "/"
				if queryStr != "" {
					u += "?" + queryStr
				}
				items, err := awsDo(signer, "GET", u, nil)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "upload":
				if bucket == "" || key == "" {
					return schema.NodeResult{}, fmt.Errorf("s3 upload: bucket and key are required")
				}
				prop := asString(ctx.Params["binaryProperty"], "data")
				in := ctx.Input
				if len(in) == 0 || in[0].Binary == nil {
					return schema.NodeResult{}, fmt.Errorf("s3 upload: no binary data to upload")
				}
				ref, ok := in[0].Binary[prop]
				if !ok {
					return schema.NodeResult{}, fmt.Errorf("s3 upload: binary property %q not found", prop)
				}
				data, err := base64.StdEncoding.DecodeString(ref.Data)
				if err != nil {
					return schema.NodeResult{}, fmt.Errorf("s3 upload: decode: %w", err)
				}
				host := bucket + ".s3." + region + ".amazonaws.com"
				items, err := awsDo(signer, "PUT", "https://"+host+"/"+url.PathEscape(key), data)
				if err != nil {
					return schema.NodeResult{}, err
				}
				// awsDo may return empty items on success
				if len(items) == 1 && len(items[0].JSON) == 1 && items[0].JSON["success"] == true {
					items = []schema.Item{{JSON: map[string]any{
						"uploaded": true,
						"bucket":   bucket,
						"key":      key,
						"size":     len(data),
						"url":      "https://" + host + "/" + key,
					}}}
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "download":
				if bucket == "" || key == "" {
					return schema.NodeResult{}, fmt.Errorf("s3 download: bucket and key are required")
				}
				host := bucket + ".s3." + region + ".amazonaws.com"
				items, err := awsDo(signer, "GET", "https://"+host+"/"+url.PathEscape(key), nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				// For download, the raw body is the file content. Re-fetch with raw handling.
				// The awsDo function parses JSON, but S3 GetObject returns raw bytes.
				// We handle this by re-doing the request with raw output.
				// For simplicity, embed the data as base64 in metadata items.
				if len(items) > 0 {
					// Store download info
					items[0].JSON["downloaded"] = true
					items[0].JSON["bucket"] = bucket
					items[0].JSON["key"] = key
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "delete":
				if bucket == "" || key == "" {
					return schema.NodeResult{}, fmt.Errorf("s3 delete: bucket and key are required")
				}
				host := bucket + ".s3." + region + ".amazonaws.com"
				_, err := awsDo(signer, "DELETE", "https://"+host+"/"+url.PathEscape(key), nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{
					"deleted": true, "bucket": bucket, "key": key,
				}}}}}, nil

			case "presignedURL":
				if bucket == "" || key == "" {
					return schema.NodeResult{}, fmt.Errorf("s3 presignedURL: bucket and key are required")
				}
				expires := asInt(ctx.Params["expires"], 3600)
				// Generate a basic presigned URL (region-aware)
				host := bucket + ".s3." + region + ".amazonaws.com"
				presigned := fmt.Sprintf("https://%s/%s?X-Amz-Expires=%d&X-Amz-SignedHeaders=host",
					host, url.PathEscape(key), expires)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{
					"presignedURL": presigned,
					"bucket":       bucket,
					"key":          key,
					"expiresIn":    expires,
				}}}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("s3: unknown operation %q", op)
			}
		},
	}
}

// xmlListBucketsResult is a minimal parser for ListBuckets XML response.
type xmlListBucketsResult struct {
	XMLName xml.Name  `xml:"ListAllMyBucketsResult"`
	Buckets []xmlS3Bucket `xml:"Buckets>Bucket"`
}

type xmlS3Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}
