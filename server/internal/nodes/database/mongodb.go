package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClientMu    sync.Mutex
	mongoClientCache = make(map[string]*mongo.Client)
	newMongoClient   = func(ctx context.Context, uri string) (*mongo.Client, error) {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			return nil, err
		}
		if err := client.Ping(ctx, nil); err != nil {
			return nil, fmt.Errorf("mongodb: ping failed: %w", err)
		}
		return client, nil
	}
)

// MongoNode returns the definition for the MongoDB node.
func MongoNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "database.mongodb",
		Label:       "MongoDB",
		Description: "Query and manage documents in a MongoDB database.",
		Group:       "storage",
		Icon:        "Database",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"mongodbApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "mongodbApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "find", Options: []schema.ParamOption{
				{Label: "Find (query documents)", Value: "find"},
				{Label: "Find One (single document)", Value: "findOne"},
				{Label: "Insert One", Value: "insertOne"},
				{Label: "Insert Many", Value: "insertMany"},
				{Label: "Update One", Value: "updateOne"},
				{Label: "Update Many", Value: "updateMany"},
				{Label: "Delete One", Value: "deleteOne"},
				{Label: "Delete Many", Value: "deleteMany"},
				{Label: "Aggregate", Value: "aggregate"},
				{Label: "Trigger: New Document", Value: "trigger:newDocument"},
			}},
			{Name: "collection", Label: "Collection Name", Type: "string", Required: true},
			{Name: "query", Label: "Query / Filter (JSON)", Type: "json",
				Description: "MongoDB filter document, e.g. {\"status\": \"active\"}",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"find", "findOne", "updateOne", "updateMany", "deleteOne", "deleteMany"}}},
			{Name: "document", Label: "Document / Update (JSON)", Type: "json",
				Description: "Document to insert, or update operations (e.g. {\"$set\": {...}})",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"insertOne", "insertMany", "updateOne", "updateMany"}}},
			{Name: "pipeline", Label: "Aggregation Pipeline (JSON array)", Type: "json",
				Description: "Array of aggregation stages, e.g. [{\"$match\": {...}}, {\"$group\": {...}}]",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"aggregate"}}},
			{Name: "options", Label: "Options (JSON)", Type: "json",
				Description: "Additional options: {\"sort\": {...}, \"limit\": 10, \"skip\": 0, \"projection\": {...}}",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"find", "findOne"}}},
			{Name: "upsert", Label: "Upsert", Type: "boolean", Default: false,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"updateOne", "updateMany"}}},
		},
		Execute: executeMongo,
	}
}

// resolveMongoURI builds a MongoDB connection URI from the credential or falls back to env.
func resolveMongoURI(ctx *schema.ExecContext) (string, string, error) {
	dbName := "admin"
	if ctx.Credential != nil {
		cred, err := ctx.Credential("credential")
		if err != nil {
			return "", "", fmt.Errorf("mongodb: failed to get credentials: %w", err)
		}
		if len(cred) > 0 {
			if uri, ok := cred["connectionString"].(string); ok && uri != "" {
				return uri, dbName, nil
			}
			host, _ := cred["host"].(string)
			port, _ := cred["port"].(string)
			user, _ := cred["user"].(string)
			password, _ := cred["password"].(string)
			database, _ := cred["database"].(string)
			if database != "" {
				dbName = database
			}
			if host != "" {
				if port == "" {
					port = "27017"
				}
				authPart := ""
				if user != "" && password != "" {
					authPart = fmt.Sprintf("%s:%s@", user, password)
				}
				return fmt.Sprintf("mongodb://%s%s:%s/%s", authPart, host, port, dbName), dbName, nil
			}
		}
	}
	return "", "", fmt.Errorf("mongodb: no credential configured (provide a connection string or host)")
}

// getOrCreateMongoClient returns a cached MongoDB client for the given URI.
func getOrCreateMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	mongoClientMu.Lock()
	if client, ok := mongoClientCache[uri]; ok {
		mongoClientMu.Unlock()
		return client, nil
	}
	mongoClientMu.Unlock()

	client, err := newMongoClient(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("mongodb: failed to connect: %w", err)
	}

	mongoClientMu.Lock()
	defer mongoClientMu.Unlock()
	if existing, ok := mongoClientCache[uri]; ok {
		return existing, nil
	}
	mongoClientCache[uri] = client
	return client, nil
}

// executeMongo is the execution function for the MongoDB node.
func executeMongo(ctx *schema.ExecContext) (schema.NodeResult, error) {
	uri, dbName, err := resolveMongoURI(ctx)
	if err != nil {
		return schema.NodeResult{}, err
	}

	client, err := getOrCreateMongoClient(context.Background(), uri)
	if err != nil {
		return schema.NodeResult{}, err
	}

	collectionName, _ := ctx.Params["collection"].(string)
	if collectionName == "" {
		return schema.NodeResult{}, fmt.Errorf("mongodb: collection name is required")
	}
	coll := client.Database(dbName).Collection(collectionName)

	operation, _ := ctx.Params["operation"].(string)
	execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch operation {
	case "find":
		return mongoFind(execCtx, coll, ctx)
	case "findOne":
		return mongoFindOne(execCtx, coll, ctx)
	case "insertOne":
		return mongoInsertOne(execCtx, coll, ctx)
	case "insertMany":
		return mongoInsertMany(execCtx, coll, ctx)
	case "updateOne":
		return mongoUpdateOne(execCtx, coll, ctx)
	case "updateMany":
		return mongoUpdateMany(execCtx, coll, ctx)
	case "deleteOne":
		return mongoDeleteOne(execCtx, coll, ctx)
	case "deleteMany":
		return mongoDeleteMany(execCtx, coll, ctx)
	case "aggregate":
		return mongoAggregate(execCtx, coll, ctx)
	case "trigger:newDocument":
		return mongoTriggerNewDocument(execCtx, coll, ctx)
	default:
		return schema.NodeResult{}, fmt.Errorf("mongodb: unknown operation %q", operation)
	}
}

// --- operation helpers ---

func toBsonD(v any) (bson.D, error) {
	m := asObjectValue(v)
	if m == nil {
		return bson.D{}, nil
	}
	b, err := bson.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("mongodb: failed to marshal to BSON: %w", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("mongodb: failed to unmarshal BSON: %w", err)
	}
	return doc, nil
}

func bsonDToItem(doc bson.D) schema.Item {
	m := doc.Map()
	return schema.Item{JSON: m}
}

func mongoFind(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	filter, err := toBsonD(execCtx.RawParam("query"))
	if err != nil {
		return schema.NodeResult{}, err
	}

	findOpts := options.Find()
	optsRaw := asObjectValue(execCtx.RawParam("options"))
	if optsRaw != nil {
		if sortRaw, ok := optsRaw["sort"]; ok {
			if sortDoc, err := toBsonD(sortRaw); err == nil {
				findOpts.SetSort(sortDoc)
			}
		}
		if limit, ok := optsRaw["limit"].(int); ok {
			findOpts.SetLimit(int64(limit))
		}
		if skip, ok := optsRaw["skip"].(int); ok {
			findOpts.SetSkip(int64(skip))
		}
		if proj, ok := optsRaw["projection"]; ok {
			if projDoc, err := toBsonD(proj); err == nil {
				findOpts.SetProjection(projDoc)
			}
		}
	}

	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var out []schema.Item
	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		out = append(out, bsonDToItem(doc))
	}
	if err := cursor.Err(); err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb cursor error: %w", err)
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoFindOne(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	filter, err := toBsonD(execCtx.RawParam("query"))
	if err != nil {
		return schema.NodeResult{}, err
	}

	findOpts := options.FindOne()
	optsRaw := asObjectValue(execCtx.RawParam("options"))
	if optsRaw != nil {
		if sortRaw, ok := optsRaw["sort"]; ok {
			if sortDoc, err := toBsonD(sortRaw); err == nil {
				findOpts.SetSort(sortDoc)
			}
		}
	}

	var doc bson.D
	err = coll.FindOne(ctx, filter, findOpts).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {}}}, nil
		}
		return schema.NodeResult{}, fmt.Errorf("mongodb findOne failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {bsonDToItem(doc)}}}, nil
}

func mongoInsertOne(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	docRaw := asObjectValue(execCtx.RawParam("document"))
	if docRaw == nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb: document is required for insertOne")
	}
	// Include the document as-is plus add the inserted ID
	result, err := coll.InsertOne(ctx, docRaw)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb insertOne failed: %w", err)
	}
	docRaw["_id"] = result.InsertedID
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: docRaw}}}}, nil
}

func mongoInsertMany(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	raw := execCtx.RawParam("document")
	var docs []any
	switch v := raw.(type) {
	case []any:
		docs = v
	case string:
		var arr []any
		if json.Unmarshal([]byte(v), &arr) == nil {
			docs = arr
		}
	}
	if len(docs) == 0 {
		return schema.NodeResult{}, fmt.Errorf("mongodb: document array is required for insertMany")
	}

	result, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb insertMany failed: %w", err)
	}
	out := []schema.Item{{JSON: map[string]any{
		"insertedCount": len(result.InsertedIDs),
		"insertedIds":   result.InsertedIDs,
	}}}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoUpdateOne(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	filter, err := toBsonD(execCtx.RawParam("query"))
	if err != nil {
		return schema.NodeResult{}, err
	}
	update := asObjectValue(execCtx.RawParam("document"))
	if update == nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb: update document is required")
	}
	upsert, _ := execCtx.Params["upsert"].(bool)

	opts := options.Update().SetUpsert(upsert)
	result, err := coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb updateOne failed: %w", err)
	}
	out := []schema.Item{{JSON: map[string]any{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
		"upsertedCount": result.UpsertedCount,
		"upsertedId":    result.UpsertedID,
	}}}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoUpdateMany(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	filter, err := toBsonD(execCtx.RawParam("query"))
	if err != nil {
		return schema.NodeResult{}, err
	}
	update := asObjectValue(execCtx.RawParam("document"))
	if update == nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb: update document is required")
	}
	upsert, _ := execCtx.Params["upsert"].(bool)

	opts := options.Update().SetUpsert(upsert)
	result, err := coll.UpdateMany(ctx, filter, update, opts)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb updateMany failed: %w", err)
	}
	out := []schema.Item{{JSON: map[string]any{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
		"upsertedCount": result.UpsertedCount,
	}}}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoDeleteOne(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	filter, err := toBsonD(execCtx.RawParam("query"))
	if err != nil {
		return schema.NodeResult{}, err
	}
	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb deleteOne failed: %w", err)
	}
	out := []schema.Item{{JSON: map[string]any{
		"deletedCount": result.DeletedCount,
	}}}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoDeleteMany(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	filter, err := toBsonD(execCtx.RawParam("query"))
	if err != nil {
		return schema.NodeResult{}, err
	}
	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb deleteMany failed: %w", err)
	}
	out := []schema.Item{{JSON: map[string]any{
		"deletedCount": result.DeletedCount,
	}}}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoAggregate(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	raw := execCtx.RawParam("pipeline")
	var pipeline []bson.D
	switch v := raw.(type) {
	case []any:
		for _, stage := range v {
			stageDoc, err := toBsonD(stage)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("mongodb: invalid pipeline stage: %w", err)
			}
			pipeline = append(pipeline, stageDoc)
		}
	case string:
		var arr []any
		if json.Unmarshal([]byte(v), &arr) != nil {
			return schema.NodeResult{}, fmt.Errorf("mongodb: pipeline must be a JSON array")
		}
		for _, stage := range arr {
			stageDoc, err := toBsonD(stage)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("mongodb: invalid pipeline stage: %w", err)
			}
			pipeline = append(pipeline, stageDoc)
		}
	default:
		return schema.NodeResult{}, fmt.Errorf("mongodb: pipeline is required and must be a JSON array")
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb aggregate failed: %w", err)
	}
	defer cursor.Close(ctx)

	var out []schema.Item
	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		out = append(out, bsonDToItem(doc))
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func mongoTriggerNewDocument(ctx context.Context, coll *mongo.Collection, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	// Poll for new documents using a timestamp/id cursor from persistent state.
	lastID, _ := execCtx.State["lastId"].(string)

	filter := bson.D{}
	if lastID != "" {
		// Use _id > lastID as a simple cursor (string-based comparison for simplicity)
		filter = bson.D{{Key: "_id", Value: bson.D{{Key: "$gt", Value: lastID}}}}
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(1)
	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("mongodb trigger poll failed: %w", err)
	}
	defer cursor.Close(ctx)

	var out []schema.Item
	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		out = append(out, bsonDToItem(doc))
		// Store the _id for next poll
		if idRaw, ok := doc.Map()["_id"]; ok {
			execCtx.State["lastId"] = fmt.Sprintf("%v", idRaw)
		}
	}

	if len(out) == 0 {
		// No new documents; suspend for polling retry
		return schema.NodeResult{
			Outputs: map[string][]schema.Item{"main": out},
		}, nil
	}

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}
