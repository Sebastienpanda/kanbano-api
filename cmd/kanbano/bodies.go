package main

// bodyField décrit un champ attendu dans le body JSON d'une requête.
type bodyField struct {
	name     string
	typ      string
	required bool
}

// requestBodies associe méthode+chemin (tels que renvoyés par chi.Walk) à la
// liste des champs de son body JSON. À tenir à jour si les *Body des
// handlers changent (voir internal/handler/*.handler.go).
//
//nolint:goconst // table déclarative : les noms de champs JSON se répètent légitimement d'une route à l'autre, les remplacer par des constantes nuirait à la lisibilité
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
