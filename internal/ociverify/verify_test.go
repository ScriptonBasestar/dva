package ociverify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func verifyServer(t *testing.T, handler http.HandlerFunc, opts Options) (Result, error) {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return (verifier{client: s.Client(), endpoint: s.URL}).verify(context.Background(), opts)
}

func manifest(config []byte) []byte {
	return fmt.Appendf(nil, `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":%q,"size":%d},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","size":0}]}`, digest(config), len(config))
}

func TestVerifySingleManifestAndPlatform(t *testing.T) {
	config := []byte(`{"os":"linux","architecture":"amd64"}`)
	m := manifest(config)
	result, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/acme/tool/manifests/v1":
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", digest(m))
			_, _ = w.Write(m)
		case "/v2/acme/tool/blobs/" + digest(config):
			_, _ = w.Write(config)
		default:
			http.NotFound(w, r)
		}
	}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: digest(m), Platforms: []string{"linux/amd64"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified || result.ObservedDigest != digest(m) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVerifyIndexRequiredPlatform(t *testing.T) {
	child := manifest([]byte{})
	index := fmt.Appendf(nil, `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":%q,"size":%d,"platform":{"os":"linux","architecture":"arm64"}}]}`, digest(child), len(child))
	result, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/acme/tool/manifests/v1":
			w.Header().Set("Content-Type", "application/vnd.oci.image.index.v1+json")
			_, _ = w.Write(index)
		case "/v2/acme/tool/manifests/" + digest(child):
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			_, _ = w.Write(child)
		default:
			http.NotFound(w, r)
		}
	}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: digest(index), Platforms: []string{"linux/arm64"}})
	if err != nil || !result.Verified {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestVerifyRejectsDigestAndHeaderMismatches(t *testing.T) {
	m := manifest([]byte{})
	for _, tc := range []struct{ name, expected, header, want string }{
		{"expected", digest([]byte("other")), "", "digest mismatch"},
		{"header", digest(m), digest([]byte("other")), "Docker-Content-Digest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Docker-Content-Digest", tc.header)
				_, _ = w.Write(m)
			}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: tc.expected})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestVerifyRejectsMissingPlatform(t *testing.T) {
	index := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","size":1,"digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","platform":{"os":"linux","architecture":"arm64"}}]}`)
	_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.oci.image.index.v1+json")
		_, _ = w.Write(index)
	}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: digest(index), Platforms: []string{"linux/amd64"}})
	if err == nil || !strings.Contains(err.Error(), "does not contain platform") {
		t.Fatalf("err=%v", err)
	}
}

func TestVerifyHTTPFailuresAndLimits(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		_, err := verifyServer(t, http.NotFound, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000"})
		if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("oversize", func(t *testing.T) {
		_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(make([]byte, maxManifestBytes+1)) }, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000"})
		if err == nil || !strings.Contains(err.Error(), "exceeds size limit") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestVerifyRejectsMaliciousBearerRealm(t *testing.T) {
	_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="http://evil.example/token",service="x"`)
		w.WriteHeader(http.StatusUnauthorized)
	}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000"})
	if err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("err=%v", err)
	}
}

func TestVerifyUsesApprovedBearerChallenge(t *testing.T) {
	config := []byte(`{"os":"linux","architecture":"amd64"}`)
	m := manifest(config)
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if r.URL.Query().Get("service") != "registry" {
				t.Errorf("missing token service")
			}
			_, _ = w.Write([]byte(`{"token":"public-token"}`))
		case "/v2/acme/tool/manifests/v1":
			if r.Header.Get("Authorization") != "Bearer public-token" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+server.URL+`/token",service="registry",scope="repository:acme/tool:pull"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			_, _ = w.Write(m)
		case "/v2/acme/tool/blobs/" + digest(config):
			_, _ = w.Write(config)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "https://")
	result, err := (verifier{client: server.Client(), endpoint: server.URL}).verify(context.Background(), Options{Reference: host + "/acme/tool:v1", ExpectedDigest: digest(m)})
	if err != nil || !result.Verified {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestVerifyHonorsCancellation(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (verifier{client: s.Client(), endpoint: s.URL}).verify(ctx, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000"})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestParseReferenceDockerHubAndExplicitSelector(t *testing.T) {
	ref, err := parseReference("docker.io/ubuntu:v1")
	if err != nil || ref.registry != "registry-1.docker.io" || ref.repository != "library/ubuntu" {
		t.Fatalf("ref=%+v err=%v", ref, err)
	}
	if _, err := parseReference("ubuntu"); err == nil {
		t.Fatal("tagless reference accepted")
	}
}

func TestVerifyDigestSelectorMustMatchExpectedBeforeNetwork(t *testing.T) {
	called := false
	selected := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) { called = true }, Options{Reference: "registry.test/acme/tool@" + selected, ExpectedDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err == nil || !strings.Contains(err.Error(), "does not match") || called {
		t.Fatalf("called=%v err=%v", called, err)
	}
}

func TestVerifyRejectsInvalidManifestSchemas(t *testing.T) {
	t.Run("empty index", func(t *testing.T) {
		body := []byte(`{}`)
		_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.oci.image.index.v1+json")
			_, _ = w.Write(body)
		}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: digest(body)})
		if err == nil || !strings.Contains(err.Error(), "invalid OCI index") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("arbitrary child", func(t *testing.T) {
		child := []byte("not a manifest")
		index := fmt.Appendf(nil, `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":%q,"size":%d,"platform":{"os":"linux","architecture":"amd64"}}]}`, digest(child), len(child))
		_, err := verifyServer(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, digest(child)) {
				w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
				_, _ = w.Write(child)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.index.v1+json")
			_, _ = w.Write(index)
		}, Options{Reference: "registry.test/acme/tool:v1", ExpectedDigest: digest(index), Platforms: []string{"linux/amd64"}})
		if err == nil || !strings.Contains(err.Error(), "invalid OCI platform manifest") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestDecodeIndexCapsChildrenBeforeSliceGrowth(t *testing.T) {
	desc := `{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","size":0}`
	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[` + strings.TrimRight(strings.Repeat(desc+",", maxChildren+1), ",") + `]}`)
	if _, err := decodeIndex(body, "application/vnd.oci.image.index.v1+json"); err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("err=%v", err)
	}
}

func TestPublicNetworkPolicy(t *testing.T) {
	for _, value := range []struct {
		address string
		want    bool
	}{
		{"127.0.0.1", false}, {"10.0.0.1", false}, {"100.64.0.1", false}, {"169.254.1.1", false}, {"192.0.2.1", false}, {"::1", false}, {"2001:db8::1", false}, {"8.8.8.8", true}, {"2606:4700:4700::1111", true},
	} {
		if got := publicIP(netip.MustParseAddr(value.address)); got != value.want {
			t.Errorf("publicIP(%s)=%v", value.address, got)
		}
	}
	if transport, ok := publicClient().Transport.(*http.Transport); !ok || transport.Proxy != nil {
		t.Fatal("public client must not use environment proxies")
	}
}

func TestBlobRedirectPolicyStripsAuthorization(t *testing.T) {
	next := httptest.NewRequest(http.MethodGet, "https://cdn.example/blob", nil)
	next.Header.Set("Authorization", "Bearer token")
	previous := httptest.NewRequest(http.MethodGet, "https://registry.example/blob", nil)
	if err := redirectPolicy(true)(next, []*http.Request{previous}); err != nil || next.Header.Get("Authorization") != "" {
		t.Fatalf("err=%v authorization=%q", err, next.Header.Get("Authorization"))
	}
	if err := redirectPolicy(false)(next, []*http.Request{previous}); err == nil {
		t.Fatal("token redirect accepted")
	}
}

func TestVerifyAllowsHTTPSBlobCDNRedirectWithoutCredentialForwarding(t *testing.T) {
	config := []byte(`{"os":"linux","architecture":"amd64"}`)
	m := manifest(config)
	cdn := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("forwarded registry authorization to CDN")
		}
		_, _ = w.Write(config)
	}))
	defer cdn.Close()
	var origin *httptest.Server
	origin = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"token":"registry-token"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer registry-token" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+origin.URL+`/token"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v2/acme/tool/manifests/v1":
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			_, _ = w.Write(m)
		case "/v2/acme/tool/blobs/" + digest(config):
			http.Redirect(w, r, cdn.URL+"/config", http.StatusTemporaryRedirect)
		default:
			http.NotFound(w, r)
		}
	}))
	defer origin.Close()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec // httptest servers only
	host := strings.TrimPrefix(origin.URL, "https://")
	result, err := (verifier{client: client, endpoint: origin.URL}).verify(context.Background(), Options{Reference: host + "/acme/tool:v1", ExpectedDigest: digest(m)})
	if err != nil || !result.Verified {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestPureReferenceValidation(t *testing.T) {
	for _, ref := range []string{"registry.test/a/../admin:v1", "registry.test//image:v1", "registry.test/UPPER:v1", "registry.test/image", "registry.test/image:../v1", "registry.test/image:bad tag"} {
		if err := ValidateReference(ref, nil); err == nil {
			t.Errorf("accepted %q", ref)
		}
	}
	if err := ValidateReference("docker.io/library/alpine:3", []string{"linux"}); err == nil {
		t.Fatal("accepted invalid platform")
	}
	if err := ValidateReference("docker.io/library/alpine:3", []string{"linux/amd64"}); err != nil {
		t.Fatal(err)
	}
	if publicIP(netip.MustParseAddr("::ffff:192.0.2.1")) {
		t.Fatal("accepted mapped reserved IP")
	}
}
