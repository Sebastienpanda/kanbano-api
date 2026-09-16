package main

// bodyField describes an expected field in a request's JSON body.
type bodyField struct {
	name     string
	typ      string
	required bool
}

// requestBodies maps method+path (as returned by chi.Walk) to the list of
// fields in its JSON body. Keep in sync if the handlers' *Body types change
// (see internal/handler/*.handler.go).
//
//nolint:goconst // declarative table: JSON field names legitimately repeat across routes, replacing them with constants would hurt readability
var requestBodies = map[string][]bodyField{
	"PATCH /api/v1/me/": {
		{"name", "string", true},
	},
	"POST /api/v1/organisation/invitations/": {
		{"email", "string", true},
		{"role", "string", false},
		{"workspace_id", "uuid", false},
		{"column_id", "uuid", false},
		{"task_id", "uuid", false},
	},
	"PATCH /api/v1/organisation/invitations/{id}": {
		{"status", "string", true},
	},
	"POST /api/v1/tags/": {
		{"name", "string", true},
		{"color", "string", false},
	},
	"PATCH /api/v1/tags/{id}": {
		{"name", "string", false},
		{"color", "string", false},
	},
	"POST /api/v1/workspaces/": {
		{"name", "string", true},
		{"description", "string", false},
	},
	"PATCH /api/v1/workspaces/{id}/": {
		{"name", "string", false},
		{"description", "string", false},
	},
	"POST /api/v1/workspaces/{id}/columns": {
		{"name", "string", true},
	},
	"PATCH /api/v1/workspaces/{id}/columns/{columnId}": {
		{"name", "string", false},
		{"position", "int", false},
	},
	"POST /api/v1/workspaces/{id}/columns/{columnId}/tasks": {
		{"name", "string", true},
		{"description", "string", false},
		{"tag_id", "uuid", false},
		{"tag_name", "string", false},
		{"status", "string", false},
	},
	"PATCH /api/v1/workspaces/{id}/columns/{columnId}/tasks/{taskId}": {
		{"name", "string", false},
		{"description", "string", false},
		{"tag_id", "uuid", false},
		{"tag_name", "string", false},
		{"status", "string", false},
		{"position", "int", false},
		{"targetColumnId", "uuid", false},
	},
	"POST /api/v1/workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees": {
		{"member_id", "uuid", true},
		{"role", "string", true},
	},
}

func bodyFieldsFor(method, path string) []bodyField {
	return requestBodies[method+" "+path]
}
