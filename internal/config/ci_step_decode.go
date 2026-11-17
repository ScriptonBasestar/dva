package config

import (
	"encoding/json"
	"fmt"
	"strings"

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
	if value, present := fields["condition"]; present {
		condition, ok := value.(map[string]any)
		if !ok || len(condition) != 2 {
			return fmt.Errorf("condition requires only run and timeout")
		}
		run, runOK := condition["run"].(string)
		timeout, timeoutOK := condition["timeout"].(string)
		if !runOK || !timeoutOK || strings.TrimSpace(run) == "" || strings.TrimSpace(timeout) == "" {
			return fmt.Errorf("condition requires nonempty run and timeout strings")
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
	if value, present := fields["condition"]; present {
		var condition map[string]json.RawMessage
		if err := json.Unmarshal(value, &condition); err != nil || len(condition) != 2 {
			return fmt.Errorf("condition requires only run and timeout")
		}
		var run, timeout string
		if err := json.Unmarshal(condition["run"], &run); err != nil || strings.TrimSpace(run) == "" {
			return fmt.Errorf("condition.run must be a nonempty string")
		}
		if err := json.Unmarshal(condition["timeout"], &timeout); err != nil || strings.TrimSpace(timeout) == "" {
			return fmt.Errorf("condition.timeout must be a nonempty string")
		}
	}
	var decoded ciStepAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = CIStep(decoded)
	return nil
}
