package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Calendar — Outlook calendar events via Graph. Enhanced with update, calendar
// list, and free/busy scheduling.
func Calendar(base string) rest.Node {
	evID := sp("eventId", "Event ID", true)
	body := jp("body", "Body (JSON)")
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(25)}
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	calID := sp("calendarId", "Calendar ID", true)
	tMin := schema.ParamSchema{Name: "timeMin", Label: "Start (ISO 8601)", Type: "string", Placeholder: "2025-01-01T00:00:00Z"}
	tMax := schema.ParamSchema{Name: "timeMax", Label: "End (ISO 8601)", Type: "string", Placeholder: "2025-01-07T00:00:00Z"}
	schedules := schema.ParamSchema{Name: "schedules", Label: "Email addresses (comma-sep)", Type: "string", Placeholder: "alice@example.com,bob@example.com"}
	availabilityViewInterval := schema.ParamSchema{Name: "interval", Label: "Interval (minutes)", Type: "number", Default: float64(30)}

	return node(base, "microsoft.outlookCalendar", "Outlook Calendar", "Calendar", "Manage Outlook calendar events, list calendars, and query free/busy availability.", []rest.Op{
		// Events
		{Resource: "event", Name: "list", Label: "List Events", Method: "GET", Path: "/me/events", ItemsPath: "value",
			Query: map[string]string{"$top": "$top", "$filter": "$filter", "startDateTime": "timeMin", "endDateTime": "timeMax"},
			Params: []schema.ParamSchema{top, filter, tMin, tMax}},
		{Resource: "event", Name: "get", Label: "Get Event", Method: "GET", Path: "/me/events/{eventId}", Params: []schema.ParamSchema{evID}},
		{Resource: "event", Name: "create", Label: "Create Event", Method: "POST", Path: "/me/events", BodyParam: "body", Params: []schema.ParamSchema{body}},
		{Resource: "event", Name: "update", Label: "Update Event", Method: "PATCH", Path: "/me/events/{eventId}", BodyParam: "body", Params: []schema.ParamSchema{evID, body}},
		{Resource: "event", Name: "delete", Label: "Delete Event", Method: "DELETE", Path: "/me/events/{eventId}", Params: []schema.ParamSchema{evID}},
		// Calendar management
		{Resource: "calendar", Name: "list", Label: "List Calendars", Method: "GET", Path: "/me/calendars", ItemsPath: "value",
			Params: []schema.ParamSchema{top}},
		{Resource: "calendar", Name: "get", Label: "Get Calendar", Method: "GET", Path: "/me/calendars/{calendarId}", Params: []schema.ParamSchema{calID}},
		{Resource: "calendar", Name: "listEvents", Label: "List Calendar Events", Method: "GET", Path: "/me/calendars/{calendarId}/events", ItemsPath: "value",
			Params: []schema.ParamSchema{calID, top, tMin, tMax}},
		// Free/busy scheduling
		{Resource: "freebusy", Name: "query", Label: "Get Schedule (Free/Busy)", Method: "POST", Path: "/me/calendar/getSchedule", BodyParam: "body",
			Params: []schema.ParamSchema{schedules, tMin, tMax, availabilityViewInterval, body}},
	})
}
