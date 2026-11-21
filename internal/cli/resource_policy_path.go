package cli

import (
	"errors"
	"os"
)

// resolveResourcePolicyPath returns the --policy value, else
// DVA_RESOURCE_POLICY. A missing policy is named as missing rather than
// surfacing later as a misleading "must be absolute" on the empty path.
func resolveResourcePolicyPath(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	path, declared := os.LookupEnv("DVA_RESOURCE_POLICY")
	if !declared {
		return "", errors.New("no resource policy: pass --policy ABSOLUTE_PATH or set DVA_RESOURCE_POLICY")
	}
	if path == "" {
		return "", errors.New("DVA_RESOURCE_POLICY is explicitly empty: set it to an absolute policy path, or unset it and pass --policy ABSOLUTE_PATH")
	}
	return path, nil
}
