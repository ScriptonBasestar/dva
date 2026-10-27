package config

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Validate presence before decoding to a string: null/empty must not become omission.
func (s *CIStep) UnmarshalYAML(node *yaml.Node) error {
	var fields map[string]any
	if err := node.Decode(&fields); err != nil {
		return err
	}
	if value, present := fields["child_ci"]; present {
		if mode, ok := value.(string); !ok || mode != CIChildIndependentRoot {
			return fmt.Errorf("child_ci must be independent-root when present")
		}
	}
	var decoded ciStepAlias
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	*s = CIStep(decoded)
	return nil
}

func (s *CIStep) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if value, present := fields["child_ci"]; present {
		var mode string
		if err := json.Unmarshal(value, &mode); err != nil || mode != CIChildIndependentRoot {
			return fmt.Errorf("child_ci must be independent-root when present")
		}
	}
	var decoded ciStepAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = CIStep(decoded)
	return nil
}
