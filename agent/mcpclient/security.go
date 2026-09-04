package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"go_binance_futures/models"
	"golang.org/x/oauth2"
)

const defaultMaxResponseBytes int64 = 4 << 20

type SecretResolver func(context.Context, string) (string, error)

func ResolveEnvironmentSecret(_ context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if !strings.HasPrefix(ref, "env:") {
		return "", fmt.Errorf("unsupported managed MCP credential reference")
	}
	name := strings.TrimSpace(strings.TrimPrefix(ref, "env:"))
	if name == "" {
		return "", fmt.Errorf("managed MCP credential reference is invalid")
	}
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("managed MCP credential is unavailable")
	}
	return value, nil
}

func ValidateEndpoint(raw string, allowPrivate bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" && !(allowPrivate && parsed.Scheme == "http") {
		return nil, fmt.Errorf("MCP endpoint must use HTTPS; HTTP requires allow_private")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("MCP endpoint must not contain userinfo")
	}
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("MCP endpoint hostname is required")
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("MCP endpoint fragment is not allowed")
	}
	return parsed, nil
}

func validateResolvedIP(ip net.IP, allowPrivate bool) error {
	if ip == nil {
		return fmt.Errorf("empty resolved IP")
	}
	if allowPrivate {
		return nil
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return fmt.Errorf("MCP endpoint resolved to blocked address %s", ip.String())
	}
	return nil
}

type maxReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func (r *maxReadCloser) Read(p []byte) (int, error) { return r.reader.Read(p) }
func (r *maxReadCloser) Close() error               { return r.closer.Close() }

type authRoundTripper struct {
	base                     http.RoundTripper
	authType, secret, header string
	maxBytes                 int64
}

func (rt authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	switch rt.authType {
	case AuthBearer:
		clone.Header.Set("Authorization", "Bearer "+rt.secret)
	case AuthCustomHeader:
		clone.Header.Set(rt.header, rt.secret)
	}
	resp, err := rt.base.RoundTrip(clone)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil && rt.maxBytes > 0 {
		resp.Body = &maxReadCloser{reader: io.LimitReader(resp.Body, rt.maxBytes+1), closer: resp.Body}
	}
	return resp, nil
}

type staticOAuthHandler struct{ source oauth2.TokenSource }

func (h staticOAuthHandler) TokenSource(context.Context) (oauth2.TokenSource, error) {
	return h.source, nil
}
func (h staticOAuthHandler) Authorize(context.Context, *http.Request, *http.Response) error {
	return errors.New("MCP OAuth authorization is required; refresh the managed credential")
}

func oauthSource(ctx context.Context, raw string, allowPrivate bool) (oauth2.TokenSource, error) {
	var credential OAuthCredential
	if err := json.Unmarshal([]byte(raw), &credential); err != nil {
		return nil, fmt.Errorf("decode oauth2 managed credential: %w", err)
	}
	if strings.TrimSpace(credential.AccessToken) == "" {
		return nil, fmt.Errorf("oauth2 managed credential requires access_token")
	}
	token := &oauth2.Token{AccessToken: credential.AccessToken, TokenType: credential.TokenType, RefreshToken: credential.RefreshToken, Expiry: credential.Expiry}
	if credential.RefreshToken == "" || credential.TokenURL == "" || credential.ClientID == "" {
		return oauth2.StaticTokenSource(token), nil
	}
	// OAuth refresh is another server-side outbound HTTP path, so it must use
	// the same endpoint, DNS/IP and redirect restrictions as the MCP endpoint.
	oauthClient, _, err := buildHTTPClient(ctx, models.AgentMCPServer{
		Endpoint: credential.TokenURL, AuthType: AuthNone,
		AllowPrivate: map[bool]int{true: 1, false: 0}[allowPrivate],
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth2 token endpoint: %w", err)
	}
	oauthCtx := context.WithValue(ctx, oauth2.HTTPClient, oauthClient)
	cfg := oauth2.Config{ClientID: credential.ClientID, ClientSecret: credential.ClientSecret, Endpoint: oauth2.Endpoint{TokenURL: credential.TokenURL}, Scopes: credential.Scopes}
	return cfg.TokenSource(oauthCtx, token), nil
}

func buildHTTPClient(ctx context.Context, server models.AgentMCPServer, resolver SecretResolver) (*http.Client, auth.OAuthHandler, error) {
	parsed, err := ValidateEndpoint(server.Endpoint, server.AllowPrivate == 1)
	if err != nil {
		return nil, nil, err
	}
	if resolver == nil {
		resolver = ResolveEnvironmentSecret
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Do not inherit HTTP_PROXY/HTTPS_PROXY. A proxy would move DialContext away
	// from the configured MCP host and undermine the target-IP SSRF check below.
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("MCP endpoint %q resolved no addresses", host)
		}
		dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		for _, item := range ips {
			if err := validateResolvedIP(item.IP, server.AllowPrivate == 1); err != nil {
				return nil, err
			}
		}
		var lastErr error
		for _, item := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(item.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("MCP endpoint has no allowed addresses")
		}
		return nil, lastErr
	}
	transport.ResponseHeaderTimeout = 15 * time.Second
	secret := ""
	if server.AuthType != AuthNone {
		secret, err = resolver(ctx, server.SecretRef)
		if err != nil {
			return nil, nil, err
		}
	}
	authType := server.AuthType
	header := strings.TrimSpace(server.CustomHeader)
	if authType == AuthCustomHeader && !validHeaderName(header) {
		return nil, nil, fmt.Errorf("invalid custom header name")
	}
	rt := http.RoundTripper(transport)
	if authType == AuthBearer || authType == AuthCustomHeader {
		rt = authRoundTripper{base: rt, authType: authType, secret: secret, header: header, maxBytes: defaultMaxResponseBytes}
	} else {
		rt = authRoundTripper{base: rt, maxBytes: defaultMaxResponseBytes}
	}
	client := &http.Client{Transport: rt, Timeout: 30 * time.Second}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many MCP redirects")
		}
		if req.URL.Scheme != parsed.Scheme || !strings.EqualFold(req.URL.Hostname(), parsed.Hostname()) {
			return http.ErrUseLastResponse
		}
		_, err := ValidateEndpoint(req.URL.String(), server.AllowPrivate == 1)
		return err
	}
	var oauth auth.OAuthHandler
	if authType == AuthOAuth2 {
		source, err := oauthSource(ctx, secret, server.AllowPrivate == 1)
		if err != nil {
			return nil, nil, err
		}
		oauth = staticOAuthHandler{source: oauth2.ReuseTokenSource(nil, source)}
	}
	return client, oauth, nil
}

func validHeaderName(value string) bool {
	if value == "" || strings.EqualFold(value, "Host") || strings.EqualFold(value, "Content-Length") {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("!#$%&'*+-.^_`|~", r) {
			return false
		}
	}
	return true
}
