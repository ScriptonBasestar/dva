package config

// DiagnosticSeverity controls validation policy independently of message text.
// The zero value remains actionable; only an explicit info classification is exempt.
type DiagnosticSeverity uint8

const (
	SeverityWarning DiagnosticSeverity = iota
	SeverityInfo
)

// ValidationDiagnostic describes a non-fatal semantic finding. This is an internal
// validation API, not a replacement for the CLI's existing JSON warning format.
type ValidationDiagnostic struct {
	Severity DiagnosticSeverity
	Message  string
}

// Actionable reports whether strict validation must reject this diagnostic.
// Unknown severities stay actionable rather than silently bypassing strict mode.
func (d ValidationDiagnostic) Actionable() bool {
	return d.Severity != SeverityInfo
}

// ValidateWarnings preserves the legacy message-only API, including order and
// wording. Validation policy should use ValidateDiagnostics instead.
func (c *Config) ValidateWarnings() []string {
	var messages []string
	for _, diagnostic := range c.ValidateDiagnostics() {
		messages = append(messages, diagnostic.Message)
	}
	return messages
}
