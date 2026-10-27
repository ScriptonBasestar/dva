package config

import (
	"encoding/json"
	"fmt"
)

type ciStepAlias CIStep

type ciOCIStepWire struct {
	Name      string   `yaml:"name" json:"name"`
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Timeout   string   `yaml:"timeout" json:"timeout"`
	OCI       *CIOCI   `yaml:"oci" json:"oci"`
}

func (s CIStep) MarshalJSON() ([]byte, error) {
	if s.OCI != nil && s.ChildCI != "" {
		return nil, fmt.Errorf("child_ci is only supported by native run steps")
	}
	if s.OCI == nil {
		return json.Marshal(ciStepAlias(s))
	}
	return json.Marshal(ciOCIStepWire{Name: s.Name, DependsOn: s.DependsOn, Timeout: s.Timeout, OCI: s.OCI})
}

func (s CIStep) MarshalYAML() (any, error) {
	if s.OCI != nil && s.ChildCI != "" {
		return nil, fmt.Errorf("child_ci is only supported by native run steps")
	}
	if s.OCI == nil {
		return ciStepAlias(s), nil
	}
	return ciOCIStepWire{Name: s.Name, DependsOn: s.DependsOn, Timeout: s.Timeout, OCI: s.OCI}, nil
}
