// Package ociverify verifies public OCI registry references by digest.
package ociverify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	maxManifestBytes = 4 << 20
	maxConfigBytes   = 1 << 20
	maxTokenBytes    = 1 << 20
	maxChildren      = 128
)

// Options specifies a public image reference and its required digest.
type Options struct {
	Reference      string
	ExpectedDigest string
	Platforms      []string
}

// Result is suitable for JSON output and deliberately contains no credentials.
type Result struct {
	Reference      string   `json:"reference"`
	ObservedDigest string   `json:"observed_digest"`
	ExpectedDigest string   `json:"expected_digest"`
	Platforms      []string `json:"platforms,omitempty"`
	Verified       bool     `json:"verified"`
}

// Verify fetches and cryptographically verifies a public OCI image manifest.
func Verify(ctx context.Context, opts Options) (Result, error) {
	client := publicClient()
	defer client.CloseIdleConnections()
	return verifier{client: client}.verify(ctx, opts)
}

func (v verifier) verify(ctx context.Context, opts Options) (Result, error) {
	result := Result{Reference: opts.Reference, ExpectedDigest: opts.ExpectedDigest, Platforms: append([]string(nil), opts.Platforms...)}
	if opts.ExpectedDigest == "" {
		return result, errors.New("OCI expected digest is required")
	}
	if !isSHA256(opts.ExpectedDigest) {
		return result, fmt.Errorf("OCI expected digest must be sha256: %q", opts.ExpectedDigest)
	}
	ref, err := parseReference(opts.Reference)
	if err != nil {
		return result, err
	}
	if isSHA256(ref.selector) && ref.selector != opts.ExpectedDigest {
		return result, errors.New("OCI reference digest does not match expected digest")
	}
	platforms, err := parsePlatforms(opts.Platforms)
	if err != nil {
		return result, err
	}

	body, mediaType, err := v.fetch(ctx, ref, "manifests", ref.selector, maxManifestBytes)
	if err != nil {
		return result, err
	}
	observed := digest(body)
	result.ObservedDigest = observed
	if observed != opts.ExpectedDigest {
		return result, fmt.Errorf("OCI manifest digest mismatch: got %s", observed)
	}

	if isIndexMediaType(mediaType) {
		index, err := decodeIndex(body, mediaType)
		if err != nil {
			return result, err
		}
		for _, required := range platforms {
			found := false
			for _, child := range index.Manifests {
				if child.Platform != nil && samePlatform(required, *child.Platform) {
					found = true
					if !isSHA256(child.Digest) {
						return result, fmt.Errorf("OCI child descriptor has unsupported digest: %q", child.Digest)
					}
					childBody, childType, err := v.fetch(ctx, ref, "manifests", child.Digest, maxManifestBytes)
					if err != nil {
						return result, fmt.Errorf("fetch OCI platform manifest %s: %w", required.text, err)
					}
					if digest(childBody) != child.Digest {
						return result, fmt.Errorf("OCI platform manifest digest mismatch for %s", required.text)
					}
					if _, err := decodeManifest(childBody, childType); err != nil {
						return result, fmt.Errorf("invalid OCI platform manifest %s: %w", required.text, err)
					}
					break
				}
			}
			if !found {
				return result, fmt.Errorf("OCI index does not contain platform %s", required.text)
			}
		}
	} else {
		manifest, err := decodeManifest(body, mediaType)
		if err != nil {
			return result, err
		}
		config, _, err := v.fetch(ctx, ref, "blobs", manifest.Config.Digest, maxConfigBytes)
		if err != nil {
			return result, fmt.Errorf("fetch OCI config: %w", err)
		}
		if digest(config) != manifest.Config.Digest {
			return result, errors.New("OCI config digest mismatch")
		}
		var cfg imageConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return result, fmt.Errorf("decode OCI config: %w", err)
		}
		if cfg.OS == "" || cfg.Architecture == "" {
			return result, errors.New("OCI config is missing platform")
		}
		actual := platform{OS: cfg.OS, Architecture: cfg.Architecture, Variant: cfg.Variant, text: platformText(cfg.OS, cfg.Architecture, cfg.Variant)}
		for _, required := range platforms {
			if !samePlatform(required, actual) {
				return result, fmt.Errorf("OCI manifest platform is %s, not %s", actual.text, required.text)
			}
		}
	}
	result.Verified = true
	return result, nil
}

type descriptor struct {
	MediaType string    `json:"mediaType"`
	Digest    string    `json:"digest"`
	Size      *int64    `json:"size"`
	Platform  *platform `json:"platform"`
}
type imageIndex struct {
	Manifests []descriptor `json:"manifests"`
}
type imageManifest struct {
	SchemaVersion int           `json:"schemaVersion"`
	MediaType     string        `json:"mediaType"`
	Config        descriptor    `json:"config"`
	Layers        *[]descriptor `json:"layers"`
}
type imageConfig struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant"`
}
type platform struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant"`
	text         string
}

func parsePlatforms(values []string) ([]platform, error) {
	out := make([]platform, 0, len(values))
	for _, value := range values {
		p := strings.Split(value, "/")
		if len(p) < 2 || len(p) > 3 || p[0] == "" || p[1] == "" {
			return nil, fmt.Errorf("invalid OCI platform %q", value)
		}
		item := platform{OS: p[0], Architecture: p[1], text: value}
		if len(p) == 3 {
			item.Variant = p[2]
			if item.Variant == "" {
				return nil, fmt.Errorf("invalid OCI platform %q", value)
			}
		}
		out = append(out, item)
	}
	return out, nil
}
func samePlatform(a, b platform) bool {
	return a.OS == b.OS && a.Architecture == b.Architecture && (a.Variant == "" || a.Variant == b.Variant)
}

func platformText(os, architecture, variant string) string {
	if variant == "" {
		return os + "/" + architecture
	}
	return os + "/" + architecture + "/" + variant
}
func isIndexMediaType(mediaType string) bool {
	return mediaType == "application/vnd.oci.image.index.v1+json" || mediaType == "application/vnd.docker.distribution.manifest.list.v2+json"
}

func isManifestMediaType(mediaType string) bool {
	return mediaType == "application/vnd.oci.image.manifest.v1+json" || mediaType == "application/vnd.docker.distribution.manifest.v2+json"
}

func decodeIndex(body []byte, mediaType string) (imageIndex, error) {
	if !isIndexMediaType(mediaType) {
		return imageIndex{}, fmt.Errorf("unsupported OCI manifest media type %q", mediaType)
	}
	var root struct {
		SchemaVersion int             `json:"schemaVersion"`
		MediaType     string          `json:"mediaType"`
		Manifests     json.RawMessage `json:"manifests"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return imageIndex{}, fmt.Errorf("decode OCI index: %w", err)
	}
	if root.SchemaVersion != 2 || root.Manifests == nil || (root.MediaType != "" && root.MediaType != mediaType) {
		return imageIndex{}, errors.New("invalid OCI index schema")
	}
	dec := json.NewDecoder(strings.NewReader(string(root.Manifests)))
	token, err := dec.Token()
	if err != nil || token != json.Delim('[') {
		return imageIndex{}, errors.New("invalid OCI index manifests")
	}
	var out imageIndex
	for dec.More() {
		if len(out.Manifests) == maxChildren {
			return imageIndex{}, fmt.Errorf("OCI index has too many manifests: %d", len(out.Manifests)+1)
		}
		var item descriptor
		if err := dec.Decode(&item); err != nil {
			return imageIndex{}, fmt.Errorf("decode OCI index descriptor: %w", err)
		}
		if err := validateDescriptor(item); err != nil {
			return imageIndex{}, fmt.Errorf("invalid OCI index descriptor: %w", err)
		}
		out.Manifests = append(out.Manifests, item)
	}
	if _, err := dec.Token(); err != nil {
		return imageIndex{}, errors.New("invalid OCI index manifests")
	}
	if len(out.Manifests) == 0 {
		return imageIndex{}, errors.New("OCI image index contains no manifests")
	}
	return out, nil
}

func decodeManifest(body []byte, mediaType string) (imageManifest, error) {
	if !isManifestMediaType(mediaType) {
		return imageManifest{}, fmt.Errorf("unsupported OCI manifest media type %q", mediaType)
	}
	var manifest imageManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return imageManifest{}, fmt.Errorf("decode OCI manifest: %w", err)
	}
	if manifest.SchemaVersion != 2 || manifest.Layers == nil || (manifest.MediaType != "" && manifest.MediaType != mediaType) {
		return imageManifest{}, errors.New("invalid OCI manifest schema")
	}
	if err := validateDescriptor(manifest.Config); err != nil {
		return imageManifest{}, fmt.Errorf("invalid OCI config descriptor: %w", err)
	}
	for _, layer := range *manifest.Layers {
		if err := validateDescriptor(layer); err != nil {
			return imageManifest{}, fmt.Errorf("invalid OCI layer descriptor: %w", err)
		}
	}
	return manifest, nil
}

func validateDescriptor(desc descriptor) error {
	if desc.MediaType == "" || desc.Size == nil || *desc.Size < 0 || !isSHA256(desc.Digest) {
		return errors.New("missing or invalid descriptor fields")
	}
	return nil
}

func bearerParams(challenge string) (map[string]string, bool) {
	if !strings.HasPrefix(strings.ToLower(challenge), "bearer ") {
		return nil, false
	}
	out := map[string]string{}
	for part := range strings.SplitSeq(strings.TrimSpace(challenge[7:]), ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			return nil, false
		}
		v = strings.Trim(strings.TrimSpace(v), "\"")
		if k == "realm" && v == "" {
			return nil, false
		}
		out[strings.ToLower(k)] = v
	}
	return out, out["realm"] != ""
}
