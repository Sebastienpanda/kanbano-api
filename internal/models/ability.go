package models

// AbilityRule is a single CASL rule, in the shape @casl/ability's
// createMongoAbility(rules) expects directly on the Angular side.
type AbilityRule struct {
	Action     string         `json:"action" enums:"read,create,update,delete,share,assign,invite"`
	Subject    string         `json:"subject" enums:"Workspace,Column,Task"`
	Fields     []string       `json:"fields,omitempty"`
	Conditions map[string]any `json:"conditions,omitempty"`
}
