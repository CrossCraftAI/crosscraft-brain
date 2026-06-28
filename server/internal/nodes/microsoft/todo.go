package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// ToDo — Microsoft To Do task lists and tasks via Graph. Enhanced with list CRUD
// and task get/update.
func ToDo(base string) rest.Node {
	listID := sp("listId", "List ID", true)
	taskID := sp("taskId", "Task ID", true)
	body := jp("body", "Body (JSON)")
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(50)}
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	listName := schema.ParamSchema{Name: "displayName", Label: "Name", Type: "string"}
	taskTitle := schema.ParamSchema{Name: "title", Label: "Title", Type: "string"}
	importance := schema.ParamSchema{Name: "importance", Label: "Importance", Type: "select", Default: "normal",
		Options: []schema.ParamOption{{Label: "Low", Value: "low"}, {Label: "Normal", Value: "normal"}, {Label: "High", Value: "high"}}}
	status := schema.ParamSchema{Name: "status", Label: "Status", Type: "select", Default: "notStarted",
		Options: []schema.ParamOption{{Label: "Not Started", Value: "notStarted"}, {Label: "In Progress", Value: "inProgress"}, {Label: "Completed", Value: "completed"}, {Label: "Waiting", Value: "waitingOnOthers"}, {Label: "Deferred", Value: "deferred"}}}
	dueDate := schema.ParamSchema{Name: "dueDateTime", Label: "Due date (ISO 8601)", Type: "string", Placeholder: "2025-06-01T00:00:00Z"}
	reminder := schema.ParamSchema{Name: "reminderDateTime", Label: "Reminder (ISO 8601)", Type: "string", Placeholder: "2025-05-31T18:00:00Z"}
	note := schema.ParamSchema{Name: "notes", Label: "Notes", Type: "code"}

	return node(base, "microsoft.todo", "Microsoft To Do", "CheckSquare", "Manage Microsoft To Do task lists and tasks with full CRUD.", []rest.Op{
		// Task lists
		{Resource: "list", Name: "list", Label: "List Task Lists", Method: "GET", Path: "/me/todo/lists", ItemsPath: "value",
			Params: []schema.ParamSchema{top, filter}},
		{Resource: "list", Name: "get", Label: "Get Task List", Method: "GET", Path: "/me/todo/lists/{listId}", Params: []schema.ParamSchema{listID}},
		{Resource: "list", Name: "create", Label: "Create Task List", Method: "POST", Path: "/me/todo/lists", BodyParam: "body",
			Params: []schema.ParamSchema{listName, body}},
		{Resource: "list", Name: "update", Label: "Update Task List", Method: "PATCH", Path: "/me/todo/lists/{listId}", BodyParam: "body",
			Params: []schema.ParamSchema{listID, listName, body}},
		{Resource: "list", Name: "delete", Label: "Delete Task List", Method: "DELETE", Path: "/me/todo/lists/{listId}", Params: []schema.ParamSchema{listID}},
		// Tasks
		{Resource: "task", Name: "list", Label: "List Tasks", Method: "GET", Path: "/me/todo/lists/{listId}/tasks", ItemsPath: "value",
			Params: []schema.ParamSchema{listID, top, filter}},
		{Resource: "task", Name: "get", Label: "Get Task", Method: "GET", Path: "/me/todo/lists/{listId}/tasks/{taskId}", Params: []schema.ParamSchema{listID, taskID}},
		{Resource: "task", Name: "create", Label: "Create Task", Method: "POST", Path: "/me/todo/lists/{listId}/tasks", BodyParam: "body",
			Params: []schema.ParamSchema{listID, taskTitle, importance, status, dueDate, reminder, note, body}},
		{Resource: "task", Name: "update", Label: "Update Task", Method: "PATCH", Path: "/me/todo/lists/{listId}/tasks/{taskId}", BodyParam: "body",
			Params: []schema.ParamSchema{listID, taskID, taskTitle, status, dueDate, importance, body}},
		{Resource: "task", Name: "delete", Label: "Delete Task", Method: "DELETE", Path: "/me/todo/lists/{listId}/tasks/{taskId}", Params: []schema.ParamSchema{listID, taskID}},
	})
}
