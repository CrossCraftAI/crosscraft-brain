package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func init() { Nodes = append(Nodes, imageNode) }

// imageNode performs basic image operations: resize, convert format, rotate,
// and extract metadata using the Go standard library.
//
// NOTE: For high-quality resizing (Lanczos, bicubic) add github.com/disintegration/imaging.
// The standard library provides only nearest-neighbor via image.NewRGBA + manual sampling,
// which works for simple downscaling but produces pixelated results at large scale factors.
var imageNode = schema.NodeDefinition{
	Type: "core.editImage", Label: "Edit Image", Group: "transform", Icon: "Image",
	Description: "Resize, rotate, convert, or extract info from images (PNG, JPEG, GIF).",
	Inputs:  []schema.Port{{ID: "main"}},
	Outputs: []schema.Port{{ID: "main"}},
	Params: []schema.ParamSchema{
		{Name: "action", Label: "Action", Type: "select", Default: "info", Options: []schema.ParamOption{
			{Label: "Get Info", Value: "info"},
			{Label: "Resize", Value: "resize"},
			{Label: "Convert Format", Value: "convert"},
			{Label: "Rotate", Value: "rotate"},
		}},
		{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"},
		{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"},
		{Name: "width", Label: "Width (px)", Type: "number", Default: float64(800),
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"resize"}}},
		{Name: "height", Label: "Height (px)", Type: "number", Default: float64(600),
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"resize"}}},
		{Name: "maintainAspectRatio", Label: "Maintain aspect ratio", Type: "boolean", Default: true,
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"resize"}}},
		{Name: "format", Label: "Target format", Type: "select", Default: "png", Options: []schema.ParamOption{
			{Label: "PNG", Value: "png"}, {Label: "JPEG", Value: "jpeg"}, {Label: "GIF", Value: "gif"},
		}, ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"convert"}}},
		{Name: "quality", Label: "JPEG quality (1-100)", Type: "number", Default: float64(85),
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"convert"}}},
		{Name: "degrees", Label: "Degrees (multiple of 90)", Type: "number", Default: float64(90),
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"rotate"}}},
	},
	Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
		action := asString(ctx.Params["action"], "info")
		inProp := asString(ctx.Params["binaryProperty"], "data")
		outProp := asString(ctx.Params["outputProperty"], "data")

		out := make([]schema.Item, 0, len(itemsOrEmpty(ctx.Input)))
		for _, item := range itemsOrEmpty(ctx.Input) {
			ref, ok := item.Binary[inProp]
			if !ok {
				return schema.NodeResult{}, fmt.Errorf("image: binary property %q not found", inProp)
			}
			data, err := base64.StdEncoding.DecodeString(ref.Data)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("image: decode binary: %w", err)
			}

			img, format, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				// Try to get info from config even if decode fails
				cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(data))
				if cfgErr != nil {
					return schema.NodeResult{}, fmt.Errorf("image: decode: %w", err)
				}
				// Return metadata only
				out = append(out, imageInfoItem(item, cfg.Width, cfg.Height, format, inProp, outProp))
				continue
			}

			switch action {
			case "info":
				bounds := img.Bounds()
				out = append(out, imageInfoItem(item, bounds.Dx(), bounds.Dy(), format, inProp, outProp))
			case "resize":
				newW := int(asFloat(ctx.Params["width"], 800))
				newH := int(asFloat(ctx.Params["height"], 600))
				maintain := true
				if v, ok := ctx.Params["maintainAspectRatio"]; ok {
					maintain = isTruthy(v)
				}
				bounds := img.Bounds()
				srcW, srcH := bounds.Dx(), bounds.Dy()
				if maintain && (newW != srcW || newH != srcH) {
					ratio := float64(srcW) / float64(srcH)
					if float64(newW)/float64(newH) > ratio {
						newW = int(float64(newH) * ratio)
					} else {
						newH = int(float64(newW) / ratio)
					}
					if newW < 1 {
						newW = 1
					}
					if newH < 1 {
						newH = 1
					}
				}
				resized := resizeNearest(img, newW, newH)
				outItem, imgErr := encodeImageItem(item, resized, format, 85, inProp, outProp)
				if imgErr != nil {
					return schema.NodeResult{}, imgErr
				}
				out = append(out, outItem)

			case "convert":
				targetFormat := asString(ctx.Params["format"], "png")
				quality := int(asFloat(ctx.Params["quality"], 85))
				outItem, imgErr := encodeImageItem(item, img, targetFormat, quality, inProp, outProp)
				if imgErr != nil {
					return schema.NodeResult{}, imgErr
				}
				out = append(out, outItem)

			case "rotate":
				deg := int(asFloat(ctx.Params["degrees"], 90))
				// Normalise to 0/90/180/270
				deg = ((deg % 360) + 360) % 360
				rotated := rotateImage(img, deg)
				outItem, imgErr := encodeImageItem(item, rotated, format, 85, inProp, outProp)
				if imgErr != nil {
					return schema.NodeResult{}, imgErr
				}
				out = append(out, outItem)
			}
		}
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
	},
}

// --- helpers ---------------------------------------------------------------

func imageInfoItem(item schema.Item, w, h int, format, inProp, outProp string) schema.Item {
	m := copyJSON(item.JSON)
	m["width"] = w
	m["height"] = h
	m["format"] = format
	return schema.Item{JSON: m, Binary: item.Binary}
}

func encodeImageItem(item schema.Item, img image.Image, format string, quality int, inProp, outProp string) (schema.Item, error) {
	var buf bytes.Buffer
	outFormat := format
	var mimeType string
	switch strings.ToLower(outFormat) {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return schema.Item{}, fmt.Errorf("image encode jpeg: %w", err)
		}
		mimeType = "image/jpeg"
		outFormat = "jpg"
	case "gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return schema.Item{}, fmt.Errorf("image encode gif: %w", err)
		}
		mimeType = "image/gif"
	default: // png
		if err := png.Encode(&buf, img); err != nil {
			return schema.Item{}, fmt.Errorf("image encode png: %w", err)
		}
		mimeType = "image/png"
		outFormat = "png"
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	m := copyJSON(item.JSON)
	m["width"] = img.Bounds().Dx()
	m["height"] = img.Bounds().Dy()
	m["format"] = outFormat
	m["size"] = buf.Len()

	binMap := map[string]schema.BinaryRef{}
	for k, v := range item.Binary {
		binMap[k] = v
	}
	binMap[outProp] = schema.BinaryRef{
		Data:     b64,
		MimeType: mimeType,
		FileName: "image." + outFormat,
	}
	return schema.Item{JSON: m, Binary: binMap}, nil
}

// resizeNearest performs nearest-neighbor scaling.
func resizeNearest(img image.Image, newW, newH int) image.Image {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == newW && srcH == newH {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		srcY := y * srcH / newH
		for x := 0; x < newW; x++ {
			srcX := x * srcW / newW
			dst.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}
	return dst
}

// rotateImage rotates by multiples of 90 degrees.
func rotateImage(img image.Image, deg int) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	var nw, nh int
	switch deg {
	case 90, 270:
		nw, nh = h, w
	default:
		nw, nh = w, h
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			var sx, sy int
			switch deg {
			case 90:
				sx = y
				sy = w - 1 - x
			case 180:
				sx = w - 1 - x
				sy = h - 1 - y
			case 270:
				sx = h - 1 - y
				sy = x
			default:
				sx, sy = x, y
			}
			dst.Set(x, y, img.At(bounds.Min.X+sx, bounds.Min.Y+sy))
		}
	}
	return dst
}

// copyJSON shallow-copies a JSON map.
func copyJSON(src map[string]any) map[string]any {
	m := make(map[string]any, len(src))
	for k, v := range src {
		m[k] = v
	}
	return m
}
