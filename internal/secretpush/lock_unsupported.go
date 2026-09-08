//go:build !darwin && !linux

package secretpush

func lockTarget(_, _ string) (func(), error) { return nil, codeError("state_platform_unsupported") }
