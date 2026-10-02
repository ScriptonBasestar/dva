package ociverify

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strings"
)

type reference struct{ registry, repository, selector string }

// ValidateReference checks declaration syntax without network access or an
// expected digest (which is supplied later by the exact workflow's artifact).
func ValidateReference(reference string, platforms []string) error {
	if _, err := parseReference(reference); err != nil {
		return err
	}
	if len(platforms) > 16 {
		return errors.New("too many OCI platforms")
	}
	_, err := parsePlatforms(platforms)
	return err
}

var repositoryComponent = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*$`)
var imageTag = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

func parseReference(s string) (reference, error) {
	if s == "" || len(s) > 1024 || strings.ContainsAny(s, "?#\\ \t\r\n\x00") || strings.Contains(s, "://") {
		return reference{}, errors.New("invalid OCI reference")
	}
	name, selector, hasDigest := strings.Cut(s, "@")
	if hasDigest {
		if !isSHA256(selector) {
			return reference{}, errors.New("OCI reference digest must be sha256")
		}
	} else {
		i := strings.LastIndex(name, ":")
		if i < 0 || i < strings.LastIndex(name, "/") {
			return reference{}, errors.New("OCI reference requires an explicit tag or digest")
		}
		selector = name[i+1:]
		name = name[:i]
		if !imageTag.MatchString(selector) || name == "" {
			return reference{}, errors.New("OCI reference requires an explicit tag or digest")
		}
	}
	parts := strings.Split(name, "/")
	registry := "docker.io"
	if strings.ContainsAny(parts[0], ".:") || parts[0] == "localhost" {
		registry, parts = parts[0], parts[1:]
	}
	if len(parts) == 0 || strings.Join(parts, "/") == "" {
		return reference{}, errors.New("OCI reference has no repository")
	}
	for _, component := range parts {
		if !repositoryComponent.MatchString(component) {
			return reference{}, errors.New("invalid OCI repository component")
		}
	}
	u, err := url.Parse("https://" + registry)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return reference{}, errors.New("invalid OCI registry host")
	}
	if registry == "docker.io" {
		registry = "registry-1.docker.io"
		if len(parts) == 1 {
			parts = append([]string{"library"}, parts...)
		}
	}
	return reference{registry: registry, repository: strings.Join(parts, "/"), selector: selector}, nil
}

func readBounded(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("OCI response exceeds size limit")
	}
	return b, nil
}
func digest(b []byte) string { sum := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(sum[:]) }
func isSHA256(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(s, "sha256:"))
	return err == nil
}
