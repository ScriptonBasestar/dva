package ociverify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strings"
	"time"
)

type verifier struct {
	client   *http.Client
	endpoint string // test-only override; public calls always use HTTPS.
}

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
