package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SuggestionIgnoreRule is one declaration in suggestion_ignore. The legacy scalar
// spelling remains supported; object entries record why an intentional suppression
// exists and, for globs, bound the number of uncovered suggestions it may hide.
type SuggestionIgnoreRule struct {
	Pattern    string `yaml:"pattern" json:"pattern"`
	Reason     string `yaml:"reason,omitempty" json:"reason,omitempty"`
	MaxMatches *int   `yaml:"max_matches,omitempty" json:"max_matches,omitempty"`
}

func (r *SuggestionIgnoreRule) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var pattern string
		if err := node.Decode(&pattern); err != nil {
			return err
		}
		r.Pattern = pattern
		r.Reason = ""
		r.MaxMatches = nil
		return nil
	case yaml.MappingNode:
		var value struct {
			Pattern    string `yaml:"pattern"`
			Reason     string `yaml:"reason"`
			MaxMatches *int   `yaml:"max_matches"`
		}
		if err := node.Decode(&value); err != nil {
			return err
		}
		r.Pattern = value.Pattern
		r.Reason = value.Reason
		r.MaxMatches = value.MaxMatches
		return r.validateObject()
	default:
		return fmt.Errorf("suggestion_ignore entry must be a string or object")
	}
}

func (r SuggestionIgnoreRule) MarshalYAML() (any, error) {
	if r.legacy() {
		return r.Pattern, nil
	}
	return struct {
		Pattern    string `yaml:"pattern"`
		Reason     string `yaml:"reason"`
		MaxMatches *int   `yaml:"max_matches,omitempty"`
	}{r.Pattern, r.Reason, r.MaxMatches}, nil
}

func (r *SuggestionIgnoreRule) UnmarshalJSON(data []byte) error {
	var pattern string
	if err := json.Unmarshal(data, &pattern); err == nil {
		r.Pattern = pattern
		r.Reason = ""
		r.MaxMatches = nil
		return nil
	}

	var value struct {
		Pattern    string `json:"pattern"`
		Reason     string `json:"reason"`
		MaxMatches *int   `json:"max_matches"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("suggestion_ignore entry must be a string or object: %w", err)
	}
	r.Pattern = value.Pattern
	r.Reason = value.Reason
	r.MaxMatches = value.MaxMatches
	return r.validateObject()
}

func (r SuggestionIgnoreRule) MarshalJSON() ([]byte, error) {
	if r.legacy() {
		return json.Marshal(r.Pattern)
	}
	return json.Marshal(struct {
		Pattern    string `json:"pattern"`
		Reason     string `json:"reason"`
		MaxMatches *int   `json:"max_matches,omitempty"`
	}{r.Pattern, r.Reason, r.MaxMatches})
}

func (r SuggestionIgnoreRule) legacy() bool {
	return r.Reason == "" && r.MaxMatches == nil
}

func (r SuggestionIgnoreRule) validateObject() error {
	if strings.TrimSpace(r.Pattern) == "" {
		return fmt.Errorf("suggestion_ignore object pattern is required")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("suggestion_ignore object reason is required")
	}
	if hasGlob(r.Pattern) && (r.MaxMatches == nil || *r.MaxMatches <= 0) {
		return fmt.Errorf("suggestion_ignore object glob %q requires a positive max_matches", r.Pattern)
	}
	if r.MaxMatches != nil && *r.MaxMatches <= 0 {
		return fmt.Errorf("suggestion_ignore object max_matches must be positive")
	}
	return nil
}

func hasGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// Matches reports whether the rule matches name using the same path.Match
// semantics suggestion_ignore has always used. Invalid legacy patterns remain a
// non-match, preserving the existing validator behaviour.
func (r SuggestionIgnoreRule) Matches(name string) bool {
	matched, err := filepath.Match(r.Pattern, name)
	return err == nil && matched
}
