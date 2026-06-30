package core

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestImageInfo(t *testing.T) {
	img := createTestImage()
	b64 := encodePNG(t, img)

	res := mustExec(t, imageNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64, MimeType: "image/png"}},
	}}, map[string]any{"action": "info", "binaryProperty": "data"}))
	out := res.Outputs["main"]
	if out[0].JSON["width"] != 10 || out[0].JSON["height"] != 20 {
		t.Fatalf("expected 10x20, got %vx%v", out[0].JSON["width"], out[0].JSON["height"])
	}
	if out[0].JSON["format"] != "png" {
		t.Fatalf("expected png format, got %v", out[0].JSON["format"])
	}
}

func TestImageResize(t *testing.T) {
	img := createTestImage()
	b64 := encodePNG(t, img)

	res := mustExec(t, imageNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64, MimeType: "image/png"}},
	}}, map[string]any{
		"action": "resize", "width": float64(5), "height": float64(5),
		"maintainAspectRatio": false, "binaryProperty": "data", "outputProperty": "data",
	}))
	out := res.Outputs["main"]
	if out[0].JSON["width"] != 5 || out[0].JSON["height"] != 5 {
		t.Fatalf("expected 5x5, got %vx%v", out[0].JSON["width"], out[0].JSON["height"])
	}
	// Verify output binary exists
	if out[0].Binary["data"].Data == "" {
		t.Fatal("expected output binary data")
	}
}

func TestImageConvert(t *testing.T) {
	img := createTestImage()
	b64 := encodePNG(t, img)

	res := mustExec(t, imageNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64, MimeType: "image/png"}},
	}}, map[string]any{
		"action": "convert", "format": "jpeg", "binaryProperty": "data", "outputProperty": "data",
	}))
	out := res.Outputs["main"]
	if out[0].JSON["format"] != "jpg" {
		t.Fatalf("expected jpg format, got %v", out[0].JSON["format"])
	}
}

func TestImageRotate(t *testing.T) {
	img := createTestImage()
	b64 := encodePNG(t, img)

	res := mustExec(t, imageNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64, MimeType: "image/png"}},
	}}, map[string]any{
		"action": "rotate", "degrees": float64(90), "binaryProperty": "data", "outputProperty": "data",
	}))
	out := res.Outputs["main"]
	// 90° rotation swaps dimensions
	if out[0].JSON["width"] != 20 || out[0].JSON["height"] != 10 {
		t.Fatalf("expected 20x10 after 90° rotation, got %vx%v", out[0].JSON["width"], out[0].JSON["height"])
	}
}

func TestImageMissingBinary(t *testing.T) {
	_, err := imageNode.Execute(ctxFor([]schema.Item{{JSON: map[string]any{}}},
		map[string]any{"action": "info", "binaryProperty": "data"}))
	if err == nil {
		t.Fatal("expected error for missing binary property")
	}
}

// helpers

func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 10, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 25), uint8(y * 12), 128, 255})
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
