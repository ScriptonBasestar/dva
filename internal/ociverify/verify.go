// Package ociverify verifies public OCI registry references by digest.
package ociverify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	maxManifestBytes = 4 << 20
	maxConfigBytes   = 1 << 20
	maxTokenBytes    = 1 << 20
	maxChildren      = 128
)

var nonPublicRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

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

type reference struct{ registry, repository, selector string }
type verifier struct {
	client   *http.Client
	endpoint string // test-only override; public calls always use HTTPS.
}

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

func (v verifier) fetch(ctx context.Context, ref reference, kind, value string, limit int64) ([]byte, string, error) {
	base := v.endpoint
	if base == "" {
		base = "https://" + ref.registry
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, "", errors.New("invalid OCI registry endpoint")
	}
	u.Path = path.Join(u.Path, "v2", ref.repository, kind, value)
	client := *v.client
	client.CheckRedirect = redirectPolicy(kind == "blobs")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", strings.Join([]string{"application/vnd.oci.image.index.v1+json", "application/vnd.oci.image.manifest.v1+json", "application/vnd.docker.distribution.manifest.list.v2+json", "application/vnd.docker.distribution.manifest.v2+json"}, ", "))
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		if err := resp.Body.Close(); err != nil {
			return nil, "", err
		}
		token, err := v.token(ctx, &client, ref.registry, resp.Header.Get("WWW-Authenticate"))
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err != nil {
			return nil, "", err
		}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("OCI %s request returned HTTP %d", kind, resp.StatusCode)
	}
	body, err := readBounded(resp.Body, limit)
	if err != nil {
		return nil, "", err
	}
	actual := digest(body)
	if header := resp.Header.Get("Docker-Content-Digest"); header != "" && header != actual {
		return nil, "", errors.New("OCI Docker-Content-Digest does not match response")
	}
	mediaType, _, _ := strings.Cut(resp.Header.Get("Content-Type"), ";")
	return body, mediaType, nil
}

func (v verifier) token(ctx context.Context, client *http.Client, registry, challenge string) (string, error) {
	params, ok := bearerParams(challenge)
	if !ok {
		return "", errors.New("OCI registry requires unsupported authentication")
	}
	realm, err := url.Parse(params["realm"])
	if err != nil || realm.Scheme != "https" || realm.Host == "" || realm.User != nil {
		return "", errors.New("OCI bearer token realm is not approved")
	}
	if realm.Host != registry && (registry != "registry-1.docker.io" || realm.Host != "auth.docker.io") {
		return "", errors.New("OCI bearer token realm is not approved")
	}
	q := realm.Query()
	if params["service"] != "" {
		q.Set("service", params["service"])
	}
	if params["scope"] != "" {
		q.Set("scope", params["scope"])
	}
	realm.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, realm.String(), nil)
	if err != nil {
		return "", err
	}
	tokenClient := *client
	tokenClient.CheckRedirect = redirectPolicy(false)
	resp, err := tokenClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OCI token request returned HTTP %d", resp.StatusCode)
	}
	body, err := readBounded(resp.Body, maxTokenBytes)
	if err != nil {
		return "", err
	}
	var token struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return "", errors.New("invalid OCI token response")
	}
	if token.Token != "" {
		return token.Token, nil
	}
	if token.AccessToken != "" {
		return token.AccessToken, nil
	}
	return "", errors.New("OCI token response has no token")
}

func redirectPolicy(allowBlobs bool) func(*http.Request, []*http.Request) error {
	return func(next *http.Request, previous []*http.Request) error {
		if !allowBlobs || next.URL.Scheme != "https" {
			return errors.New("OCI redirects are not allowed")
		}
		if len(previous) != 0 && next.URL.Host != previous[0].URL.Host {
			next.Header.Del("Authorization")
		}
		return nil
	}
}

func publicClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = publicDialContext
	return &http.Client{Timeout: 30 * time.Second, Transport: transport}
}

func publicDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("OCI registry host did not resolve")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return nil, fmt.Errorf("OCI registry host resolves to a non-public address")
		}
	}
	// Dial the checked address directly so a second resolver lookup cannot rebind it.
	dialer := net.Dialer{}
	var last error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicRanges {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
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
