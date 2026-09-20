package models

// AbilityRule is a single CASL rule, in the shape @casl/ability's
// createMongoAbility(rules) expects directly on the Angular side.
type AbilityRule struct {
	Action     string         `json:"action"`
	Subject    string         `json:"subject"`
	Conditions map[string]any `json:"conditions,omitempty"`
	Inverted   bool           `json:"inverted,omitempty"`
}
