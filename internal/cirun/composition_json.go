package cirun

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Composition grants are bounded runtime messages, not arbitrary JSON documents.
func decodeCompositionJSON(value string, target any) error {
	if len(value) == 0 || len(value) > 64<<10 {
		return fmt.Errorf("composition message must contain 1..65536 bytes")
	}
	scan := json.NewDecoder(strings.NewReader(value))
	if err := scanCompositionJSON(scan, 0); err != nil {
		return err
	}
	if _, err := scan.Token(); err != io.EOF {
		return fmt.Errorf("composition message contains trailing data")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func scanCompositionJSON(decoder *json.Decoder, depth int) error {
	if depth > 16 {
		return fmt.Errorf("composition JSON nesting exceeds 16")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, container := token.(json.Delim)
	if !container {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("composition JSON contains a duplicate or invalid key")
			}
			seen[name] = true
			if err := scanCompositionJSON(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := scanCompositionJSON(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected composition JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}
