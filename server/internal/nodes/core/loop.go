package core

import (
	"math"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func init() { Nodes = append(Nodes, loopNode) }

// loopNode iterates over items or splits them into batches.
// In the engine, the Loop Over Items / Split In Batches node feeds each
// item (or batch) back into its sub-workflow; here we emit the split
// results so downstream nodes can process them.
var loopNode = schema.NodeDefinition{
	Type: "core.loop", Label: "Loop / Split In Batches", Group: "flow", Icon: "Repeat",
	Description: "Iterate over each input item or split the input array into batches of N items.",
	Inputs:  []schema.Port{{ID: "main"}},
	Outputs: []schema.Port{{ID: "main"}},
	Params: []schema.ParamSchema{
		{Name: "mode", Label: "Mode", Type: "select", Default: "forEach", Options: []schema.ParamOption{
			{Label: "For Each Item", Value: "forEach"},
			{Label: "Split In Batches", Value: "splitBatches"},
		}},
		{Name: "batchSize", Label: "Batch size", Type: "number", Default: float64(10),
			ShowWhen: &schema.ShowWhen{Param: "mode", Equals: []any{"splitBatches"}}},
		{Name: "continueOnFail", Label: "Continue on error", Type: "boolean", Default: true,
			Description: "If true, errors on individual items do not stop the loop."},
	},
	Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
		mode := asString(ctx.Params["mode"], "forEach")
		in := itemsOrEmpty(ctx.Input)

		switch mode {
		case "splitBatches":
			batchSize := int(asFloat(ctx.Params["batchSize"], 10))
			if batchSize < 1 {
				batchSize = 1
			}
			out := make([]schema.Item, 0, (len(in)+batchSize-1)/batchSize)
			for i := 0; i < len(in); i += batchSize {
				end := i + batchSize
				if end > len(in) {
					end = len(in)
				}
				batch := in[i:end]
				arr := make([]any, len(batch))
				for j, item := range batch {
					arr[j] = item.JSON
				}
				out = append(out, schema.Item{JSON: map[string]any{
					"batch":     arr,
					"batchNum":  (i / batchSize) + 1,
					"batchSize": len(batch),
					"totalItems": len(in),
				}})
			}
			return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil

		default: // forEach
			out := make([]schema.Item, len(in))
			for i, item := range in {
				m := map[string]any{"item": i, "totalItems": len(in)}
				for k, v := range item.JSON {
					m[k] = v
				}
				out[i] = schema.Item{JSON: m}
			}
			return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
		}
	},
}

// asFloat extracts a float64 from a param value.
func asFloat(v any, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return def
	}
}

var _ = math.MaxFloat64 // keep math import
