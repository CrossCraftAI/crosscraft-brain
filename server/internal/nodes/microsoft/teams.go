package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Teams — channels, messages, chats, and apps via Graph. Enhanced with channel
// CRUD and chat message operations.
func Teams(base string) rest.Node {
	teamID := sp("teamId", "Team ID", true)
	channelID := sp("channelId", "Channel ID", true)
	body := jp("body", "Body (JSON)")
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(50)}
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	channelName := schema.ParamSchema{Name: "displayName", Label: "Channel name", Type: "string"}
	channelDesc := schema.ParamSchema{Name: "description", Label: "Description", Type: "string"}
	chatID := sp("chatId", "Chat ID", true)

	return node(base, "microsoft.teams", "Microsoft Teams", "Users", "Teams channels, messages, chats, and apps.", []rest.Op{
		// Teams
		{Resource: "team", Name: "listJoined", Label: "List Joined Teams", Method: "GET", Path: "/me/joinedTeams", ItemsPath: "value",
			Params: []schema.ParamSchema{top}},
		{Resource: "team", Name: "get", Label: "Get Team", Method: "GET", Path: "/teams/{teamId}", Params: []schema.ParamSchema{teamID}},
		{Resource: "team", Name: "listMembers", Label: "List Team Members", Method: "GET", Path: "/teams/{teamId}/members", ItemsPath: "value",
			Params: []schema.ParamSchema{teamID, top}},
		// Channels
		{Resource: "channel", Name: "list", Label: "List Channels", Method: "GET", Path: "/teams/{teamId}/channels", ItemsPath: "value",
			Params: []schema.ParamSchema{teamID, top, filter}},
		{Resource: "channel", Name: "get", Label: "Get Channel", Method: "GET", Path: "/teams/{teamId}/channels/{channelId}",
			Params: []schema.ParamSchema{teamID, channelID}},
		{Resource: "channel", Name: "create", Label: "Create Channel", Method: "POST", Path: "/teams/{teamId}/channels", BodyParam: "body",
			Params: []schema.ParamSchema{teamID, channelName, channelDesc, body}},
		{Resource: "channel", Name: "update", Label: "Update Channel", Method: "PATCH", Path: "/teams/{teamId}/channels/{channelId}", BodyParam: "body",
			Params: []schema.ParamSchema{teamID, channelID, channelName, body}},
		{Resource: "channel", Name: "delete", Label: "Delete Channel", Method: "DELETE", Path: "/teams/{teamId}/channels/{channelId}",
			Params: []schema.ParamSchema{teamID, channelID}},
		// Channel messages
		{Resource: "channelMessage", Name: "list", Label: "List Channel Messages", Method: "GET", Path: "/teams/{teamId}/channels/{channelId}/messages", ItemsPath: "value",
			Params: []schema.ParamSchema{teamID, channelID, top}},
		{Resource: "channelMessage", Name: "get", Label: "Get Message", Method: "GET", Path: "/teams/{teamId}/channels/{channelId}/messages/{messageId}",
			Params: []schema.ParamSchema{teamID, channelID, sp("messageId", "Message ID", true)}},
		{Resource: "channelMessage", Name: "send", Label: "Send Channel Message", Method: "POST", Path: "/teams/{teamId}/channels/{channelId}/messages", BodyParam: "body",
			Params: []schema.ParamSchema{teamID, channelID, body}},
		{Resource: "channelMessage", Name: "reply", Label: "Reply to Message", Method: "POST", Path: "/teams/{teamId}/channels/{channelId}/messages/{messageId}/replies", BodyParam: "body",
			Params: []schema.ParamSchema{teamID, channelID, sp("messageId", "Message ID", true), body}},
		// Chats
		{Resource: "chat", Name: "list", Label: "List Chats", Method: "GET", Path: "/me/chats", ItemsPath: "value",
			Params: []schema.ParamSchema{top}},
		{Resource: "chat", Name: "listMessages", Label: "List Chat Messages", Method: "GET", Path: "/me/chats/{chatId}/messages", ItemsPath: "value",
			Params: []schema.ParamSchema{chatID, top}},
		{Resource: "chat", Name: "sendMessage", Label: "Send Chat Message", Method: "POST", Path: "/me/chats/{chatId}/messages", BodyParam: "body",
			Params: []schema.ParamSchema{chatID, body}},
	})
}
