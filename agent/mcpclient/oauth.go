package mcpclient

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"go_binance_futures/agent/security"
	_ "go_binance_futures/bootstrap"
	"go_binance_futures/models"
	"golang.org/x/oauth2"
)

const (
	oauthPublicBaseConfig = "mcp::oauth_public_base_url"
	oauthEncryptionConfig = "mcp::oauth_encryption_key"
	oauthStateTTL         = 10 * time.Minute
	oauthSecretPrefix     = "mcpdb:oauth:"
)

type oauthPending struct {
	ServerID              int64    `json:"server_id"`
	Verifier              string   `json:"verifier"`
	Issuer                string   `json:"issuer"`
	AuthorizationURL      string   `json:"authorization_url"`
	TokenURL              string   `json:"token_url"`
	ClientID              string   `json:"client_id"`
	ClientSecret          string   `json:"client_secret,omitempty"`
	TokenAuthMethod       string   `json:"token_auth_method"`
	RedirectURL           string   `json:"redirect_url"`
	Resource              string   `json:"resource"`
	Scopes                []string `json:"scopes,omitempty"`
	RequireIssuerResponse bool     `json:"require_issuer_response"`
}

type OAuthStartResult struct {
	AuthorizationURL  string `json:"authorization_url"`
	CallbackURL       string `json:"callback_url"`
	ClientMetadataURL string `json:"client_metadata_url"`
	ExpiresAt         int64  `json:"expires_at"`
}

type OAuthCallbackResult struct {
	ServerID int64  `json:"server_id"`
	Status   string `json:"status"`
}

func oauthPublicURLs() (base, metadata, callback string, err error) {
	raw, _ := config.String(oauthPublicBaseConfig)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", fmt.Errorf("mcp.oauth_public_base_url is required for interactive MCP OAuth")
	}
	u, parseErr := url.Parse(raw)
	if parseErr != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", "", "", fmt.Errorf("mcp.oauth_public_base_url must be a valid public base URL")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname())) {
		return "", "", "", fmt.Errorf("mcp.oauth_public_base_url must use HTTPS; HTTP is only allowed for loopback development")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	base = strings.TrimRight(u.String(), "/")
	metadata = base + "/agents/mcp/oauth/client-metadata"
	callback = base + "/agents/mcp/oauth/callback"
	return base, metadata, callback, nil
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func oauthEncryptionKey() ([]byte, error) {
	raw, _ := config.String(oauthEncryptionConfig)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("mcp.oauth_encryption_key is required for interactive MCP OAuth")
	}
	if len(raw) == 64 {
		if key, err := hex.DecodeString(raw); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if key, err := encoding.DecodeString(raw); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	return nil, fmt.Errorf("mcp.oauth_encryption_key must be a 32-byte key encoded as hex or base64")
}

func encryptOAuthPayload(raw []byte, aad string) (string, error) {
	key, err := oauthEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, raw, []byte(aad))
	payload := append(nonce, sealed...)
	return "v1:" + base64.RawURLEncoding.EncodeToString(payload), nil
}

func decryptOAuthPayload(encoded, aad string) ([]byte, error) {
	if !strings.HasPrefix(encoded, "v1:") {
		return nil, fmt.Errorf("unsupported MCP OAuth ciphertext version")
	}
	key, err := oauthEncryptionKey()
	if err != nil {
		return nil, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encoded, "v1:"))
	if err != nil {
		return nil, fmt.Errorf("decode MCP OAuth ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, fmt.Errorf("MCP OAuth ciphertext is truncated")
	}
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("decrypt MCP OAuth credential: %w", err)
	}
	return plain, nil
}

func stateHash(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

func randomOAuthState() (string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func oauthSecretAAD(serverID int64) string {
	return "mcp-secret:" + strconv.FormatInt(serverID, 10) + ":oauth2"
}

func oauthStateAAD(hash string) string { return "mcp-oauth-state:" + hash }

func ResolveManagedSecret(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "env:") || ref == "" {
		return ResolveEnvironmentSecret(ctx, ref)
	}
	if !strings.HasPrefix(ref, oauthSecretPrefix) {
		return "", fmt.Errorf("unsupported managed MCP credential reference")
	}
	serverID, err := strconv.ParseInt(strings.TrimPrefix(ref, oauthSecretPrefix), 10, 64)
	if err != nil || serverID <= 0 {
		return "", fmt.Errorf("managed MCP credential reference is invalid")
	}
	var row models.AgentMCPSecret
	if err := orm.NewOrm().QueryTable(new(models.AgentMCPSecret)).Filter("server_id", serverID).One(&row); err != nil {
		return "", fmt.Errorf("managed MCP credential is unavailable")
	}
	plain, err := decryptOAuthPayload(row.Ciphertext, oauthSecretAAD(serverID))
	if err != nil {
		return "", fmt.Errorf("managed MCP credential is unavailable")
	}
	return string(plain), nil
}

func saveOAuthCredential(ctx context.Context, serverID int64, credential OAuthCredential, issuer string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	ciphertext, err := encryptOAuthPayload(raw, oauthSecretAAD(serverID))
	if err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	o := orm.NewOrm()
	var row models.AgentMCPSecret
	err = o.QueryTable(new(models.AgentMCPSecret)).Filter("server_id", serverID).One(&row)
	if err != nil && err != orm.ErrNoRows {
		return err
	}
	if err == orm.ErrNoRows {
		row = models.AgentMCPSecret{ServerID: serverID, Kind: AuthOAuth2, CreatedAt: now}
	}
	row.Ciphertext, row.UpdatedAt = ciphertext, now
	if row.ID == 0 {
		if _, err := o.Insert(&row); err != nil {
			return err
		}
	} else if _, err := o.Update(&row); err != nil {
		return err
	}
	expiresAt := int64(0)
	if !credential.Expiry.IsZero() {
		expiresAt = credential.Expiry.UTC().UnixMilli()
	}
	_, err = o.QueryTable(new(models.AgentMCPServer)).Filter("id", serverID).Update(orm.Params{
		"secret_ref": oauthSecretPrefix + strconv.FormatInt(serverID, 10),
		"auth_type":  AuthOAuth2, "oauth_status": "authorized", "oauth_issuer": issuer,
		"oauth_expires_at": expiresAt, "updated_at": now,
	})
	return err
}

func deleteOAuthCredential(ctx context.Context, serverID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := orm.NewOrm().QueryTable(new(models.AgentMCPSecret)).Filter("server_id", serverID).Delete()
	return err
}

func (g *Gateway) StartOAuth(ctx context.Context, serverID int64) (OAuthStartResult, error) {
	server, err := g.Store.GetServer(ctx, serverID)
	if err != nil {
		return OAuthStartResult{}, err
	}
	if _, _, err := oauthEncryptionConfigCheck(); err != nil {
		return OAuthStartResult{}, err
	}
	_, metadataURL, callbackURL, err := oauthPublicURLs()
	if err != nil {
		return OAuthStartResult{}, err
	}
	if _, err := oauthEncryptionKey(); err != nil {
		return OAuthStartResult{}, err
	}
	challenge, err := g.discoverOAuthChallenge(ctx, server)
	if err != nil {
		return OAuthStartResult{}, err
	}
	prm, asm, err := g.discoverOAuthMetadata(ctx, server, challenge)
	if err != nil {
		return OAuthStartResult{}, err
	}
	clientID, clientSecret, tokenAuthMethod, err := g.resolveOAuthClient(ctx, server, asm, metadataURL, callbackURL)
	if err != nil {
		return OAuthStartResult{}, err
	}
	verifier := oauth2.GenerateVerifier()
	state, err := randomOAuthState()
	if err != nil {
		return OAuthStartResult{}, err
	}
	scopes := challenge.Scopes
	if len(scopes) == 0 {
		scopes = append([]string(nil), prm.ScopesSupported...)
	}
	cfg := oauth2.Config{
		ClientID: clientID, ClientSecret: clientSecret,
		Endpoint:    oauth2.Endpoint{AuthURL: asm.AuthorizationEndpoint, TokenURL: asm.TokenEndpoint, AuthStyle: tokenAuthStyle(tokenAuthMethod)},
		RedirectURL: callbackURL, Scopes: scopes,
	}
	authURL := cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("resource", prm.Resource))
	pending := oauthPending{
		ServerID: serverID, Verifier: verifier, Issuer: asm.Issuer,
		AuthorizationURL: asm.AuthorizationEndpoint, TokenURL: asm.TokenEndpoint,
		ClientID: clientID, ClientSecret: clientSecret, TokenAuthMethod: tokenAuthMethod,
		RedirectURL: callbackURL, Resource: prm.Resource, Scopes: scopes,
		RequireIssuerResponse: asm.AuthorizationResponseIssParameterSupported,
	}
	expiresAt := time.Now().UTC().Add(oauthStateTTL)
	if err := saveOAuthPending(ctx, state, pending, expiresAt); err != nil {
		return OAuthStartResult{}, err
	}
	now := time.Now().UTC().UnixMilli()
	_, _ = g.Store.orm().QueryTable(new(models.AgentMCPServer)).Filter("id", serverID).Update(orm.Params{
		"auth_type": AuthOAuth2, "oauth_status": "authorization_pending", "oauth_issuer": asm.Issuer,
		"last_error": nil, "updated_at": now,
	})
	return OAuthStartResult{AuthorizationURL: authURL, CallbackURL: callbackURL, ClientMetadataURL: metadataURL, ExpiresAt: expiresAt.UnixMilli()}, nil
}

func oauthEncryptionConfigCheck() (string, string, error) {
	_, metadata, callback, err := oauthPublicURLs()
	return metadata, callback, err
}

type oauthChallenge struct {
	ResourceMetadata string
	Scopes           []string
}

func (g *Gateway) discoverOAuthChallenge(ctx context.Context, server models.AgentMCPServer) (oauthChallenge, error) {
	plain := server
	plain.AuthType, plain.SecretRef, plain.CustomHeader = AuthNone, "", ""
	client, _, err := buildHTTPClient(ctx, plain, g.ResolveSecret)
	if err != nil {
		return oauthChallenge{}, err
	}
	payload := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2026-07-28","capabilities":{},"clientInfo":{"name":"go-binance-futures-agent-host","version":"` + hostVersion + `"}}}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.Endpoint, strings.NewReader(payload))
	if err != nil {
		return oauthChallenge{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		return oauthChallenge{}, fmt.Errorf("probe MCP OAuth challenge: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		return oauthChallenge{}, fmt.Errorf("MCP server did not return an OAuth authorization challenge (status %d)", resp.StatusCode)
	}
	challenges, err := oauthex.ParseWWWAuthenticate(resp.Header.Values("WWW-Authenticate"))
	if err != nil {
		return oauthChallenge{}, fmt.Errorf("parse MCP OAuth challenge: %w", err)
	}
	for _, challenge := range challenges {
		if challenge.Scheme != "bearer" {
			continue
		}
		return oauthChallenge{ResourceMetadata: challenge.Params["resource_metadata"], Scopes: strings.Fields(challenge.Params["scope"])}, nil
	}
	return oauthChallenge{}, fmt.Errorf("MCP server did not provide a Bearer OAuth challenge")
}

func (g *Gateway) discoverOAuthMetadata(ctx context.Context, server models.AgentMCPServer, challenge oauthChallenge) (*oauthex.ProtectedResourceMetadata, *oauthex.AuthServerMeta, error) {
	candidates := protectedResourceCandidates(server.Endpoint, challenge.ResourceMetadata)
	var prm *oauthex.ProtectedResourceMetadata
	var lastErr error
	for _, candidate := range candidates {
		client, err := restrictedHTTPClient(ctx, candidate.MetadataURL, server.AllowPrivate == 1)
		if err != nil {
			lastErr = err
			continue
		}
		prm, err = oauthex.GetProtectedResourceMetadata(ctx, candidate.MetadataURL, candidate.Resource, client)
		if err == nil && prm != nil {
			break
		}
		lastErr = err
	}
	if prm == nil {
		return nil, nil, fmt.Errorf("discover MCP protected resource metadata: %w", lastErr)
	}
	if len(prm.AuthorizationServers) == 0 {
		return nil, nil, fmt.Errorf("MCP protected resource metadata has no authorization_servers")
	}
	issuer := prm.AuthorizationServers[0]
	client, err := restrictedHTTPClient(ctx, issuer, server.AllowPrivate == 1)
	if err != nil {
		return nil, nil, err
	}
	asm, err := auth.GetAuthServerMetadata(ctx, issuer, client)
	if err != nil {
		return nil, nil, fmt.Errorf("discover MCP authorization server metadata: %w", err)
	}
	if asm == nil {
		return nil, nil, fmt.Errorf("MCP authorization server metadata not found")
	}
	if !containsString(asm.CodeChallengeMethodsSupported, "S256") {
		return nil, nil, fmt.Errorf("MCP authorization server does not support PKCE S256")
	}
	return prm, asm, nil
}

type protectedResourceCandidate struct{ MetadataURL, Resource string }

func protectedResourceCandidates(resourceURL, challengeURL string) []protectedResourceCandidate {
	var out []protectedResourceCandidate
	if strings.TrimSpace(challengeURL) != "" {
		out = append(out, protectedResourceCandidate{challengeURL, resourceURL})
	}
	ru, err := url.Parse(resourceURL)
	if err != nil {
		return out
	}
	mu := *ru
	mu.RawQuery, mu.Fragment = "", ""
	mu.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(ru.Path, "/")
	out = append(out, protectedResourceCandidate{mu.String(), resourceURL})
	mu.Path = "/.well-known/oauth-protected-resource"
	root := *ru
	root.Path, root.RawQuery, root.Fragment = "", "", ""
	out = append(out, protectedResourceCandidate{mu.String(), root.String()})
	return out
}

func restrictedHTTPClient(ctx context.Context, endpoint string, allowPrivate bool) (*http.Client, error) {
	client, _, err := buildHTTPClient(ctx, models.AgentMCPServer{
		Endpoint: endpoint, AuthType: AuthNone,
		AllowPrivate: map[bool]int{true: 1, false: 0}[allowPrivate],
	}, nil)
	return client, err
}

func (g *Gateway) resolveOAuthClient(ctx context.Context, server models.AgentMCPServer, asm *oauthex.AuthServerMeta, metadataURL, callbackURL string) (string, string, string, error) {
	if asm.ClientIDMetadataDocumentSupported {
		method := "none"
		if len(asm.TokenEndpointAuthMethodsSupported) > 0 && !containsString(asm.TokenEndpointAuthMethodsSupported, "none") {
			return "", "", "", fmt.Errorf("authorization server supports Client ID Metadata Documents but not public token endpoint auth")
		}
		return metadataURL, "", method, nil
	}
	if asm.RegistrationEndpoint == "" {
		return "", "", "", fmt.Errorf("authorization server supports neither Client ID Metadata Documents nor dynamic client registration")
	}
	method := chooseTokenAuthMethod(asm.TokenEndpointAuthMethodsSupported)
	client, err := restrictedHTTPClient(ctx, asm.RegistrationEndpoint, server.AllowPrivate == 1)
	if err != nil {
		return "", "", "", err
	}
	registration, err := oauthex.RegisterClient(ctx, asm.RegistrationEndpoint, &oauthex.ClientRegistrationMetadata{
		RedirectURIs: []string{callbackURL}, TokenEndpointAuthMethod: method,
		GrantTypes: []string{"authorization_code"}, ResponseTypes: []string{"code"},
		ClientName: "go-binance-futures Agent Host", ApplicationType: "web",
	}, client)
	if err != nil {
		return "", "", "", fmt.Errorf("dynamic MCP OAuth client registration: %w", err)
	}
	return registration.ClientID, registration.ClientSecret, registration.TokenEndpointAuthMethod, nil
}

func chooseTokenAuthMethod(supported []string) string {
	for _, wanted := range []string{"none", "client_secret_post", "client_secret_basic"} {
		if containsString(supported, wanted) {
			return wanted
		}
	}
	if len(supported) == 0 {
		return "none"
	}
	return supported[0]
}

func tokenAuthStyle(method string) oauth2.AuthStyle {
	switch method {
	case "client_secret_basic":
		return oauth2.AuthStyleInHeader
	case "none", "client_secret_post":
		return oauth2.AuthStyleInParams
	default:
		return oauth2.AuthStyleAutoDetect
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func saveOAuthPending(ctx context.Context, state string, pending oauthPending, expiresAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	hash := stateHash(state)
	raw, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	ciphertext, err := encryptOAuthPayload(raw, oauthStateAAD(hash))
	if err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	o := orm.NewOrm()
	_, _ = o.QueryTable(new(models.AgentMCPOAuthState)).Filter("expires_at__lt", now).Delete()
	_, _ = o.QueryTable(new(models.AgentMCPOAuthState)).Filter("server_id", pending.ServerID).Delete()
	_, err = o.Insert(&models.AgentMCPOAuthState{ServerID: pending.ServerID, StateHash: hash, Ciphertext: ciphertext, ExpiresAt: expiresAt.UnixMilli(), CreatedAt: now})
	return err
}

func consumeOAuthPending(ctx context.Context, state string) (oauthPending, error) {
	if err := ctx.Err(); err != nil {
		return oauthPending{}, err
	}
	hash := stateHash(state)
	o := orm.NewOrm()
	var row models.AgentMCPOAuthState
	if err := o.QueryTable(new(models.AgentMCPOAuthState)).Filter("state_hash", hash).One(&row); err != nil {
		return oauthPending{}, fmt.Errorf("MCP OAuth state is invalid or expired")
	}
	_, _ = o.Delete(&row)
	if row.ExpiresAt <= time.Now().UTC().UnixMilli() {
		return oauthPending{}, fmt.Errorf("MCP OAuth state is invalid or expired")
	}
	plain, err := decryptOAuthPayload(row.Ciphertext, oauthStateAAD(hash))
	if err != nil {
		return oauthPending{}, err
	}
	var pending oauthPending
	if err := json.Unmarshal(plain, &pending); err != nil {
		return oauthPending{}, fmt.Errorf("decode MCP OAuth state: %w", err)
	}
	return pending, nil
}

func (g *Gateway) CompleteOAuth(ctx context.Context, state, code, iss, remoteError, remoteDescription string) (OAuthCallbackResult, error) {
	pending, err := consumeOAuthPending(ctx, strings.TrimSpace(state))
	if err != nil {
		return OAuthCallbackResult{}, err
	}
	if remoteError != "" {
		message := "MCP OAuth authorization failed: " + remoteError
		if remoteDescription != "" {
			message += " - " + remoteDescription
		}
		g.markOAuthRequired(pending.ServerID, message)
		return OAuthCallbackResult{ServerID: pending.ServerID, Status: "authorization_required"}, fmt.Errorf("%s", message)
	}
	if strings.TrimSpace(code) == "" {
		g.markOAuthRequired(pending.ServerID, "MCP OAuth callback is missing code")
		return OAuthCallbackResult{}, fmt.Errorf("MCP OAuth callback is missing code")
	}
	if pending.RequireIssuerResponse {
		if iss == "" || iss != pending.Issuer {
			g.markOAuthRequired(pending.ServerID, "MCP OAuth issuer mismatch")
			return OAuthCallbackResult{}, fmt.Errorf("MCP OAuth issuer mismatch")
		}
	} else if iss != "" && iss != pending.Issuer {
		g.markOAuthRequired(pending.ServerID, "MCP OAuth issuer mismatch")
		return OAuthCallbackResult{}, fmt.Errorf("MCP OAuth issuer mismatch")
	}
	server, err := g.Store.GetServer(ctx, pending.ServerID)
	if err != nil {
		return OAuthCallbackResult{}, err
	}
	client, err := restrictedHTTPClient(ctx, pending.TokenURL, server.AllowPrivate == 1)
	if err != nil {
		return OAuthCallbackResult{}, err
	}
	cfg := oauth2.Config{
		ClientID: pending.ClientID, ClientSecret: pending.ClientSecret,
		Endpoint:    oauth2.Endpoint{AuthURL: pending.AuthorizationURL, TokenURL: pending.TokenURL, AuthStyle: tokenAuthStyle(pending.TokenAuthMethod)},
		RedirectURL: pending.RedirectURL, Scopes: pending.Scopes,
	}
	exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, client)
	token, err := cfg.Exchange(exchangeCtx, code, oauth2.VerifierOption(pending.Verifier), oauth2.SetAuthURLParam("resource", pending.Resource))
	if err != nil {
		g.markOAuthRequired(pending.ServerID, "MCP OAuth token exchange failed")
		return OAuthCallbackResult{}, fmt.Errorf("MCP OAuth token exchange failed: %w", err)
	}
	credential := OAuthCredential{
		AccessToken: token.AccessToken, TokenType: token.TokenType, RefreshToken: token.RefreshToken,
		Expiry: token.Expiry, TokenURL: pending.TokenURL, ClientID: pending.ClientID,
		ClientSecret: pending.ClientSecret, TokenAuthMethod: pending.TokenAuthMethod, Scopes: pending.Scopes,
	}
	if err := saveOAuthCredential(ctx, pending.ServerID, credential, pending.Issuer); err != nil {
		return OAuthCallbackResult{}, err
	}
	g.recordSuccess(pending.ServerID)
	return OAuthCallbackResult{ServerID: pending.ServerID, Status: "authorized"}, nil
}

func (g *Gateway) markOAuthRequired(serverID int64, message string) {
	now := time.Now().UTC().UnixMilli()
	message = security.RedactText(message)
	if len(message) > 2000 {
		message = message[:2000]
	}
	_, _ = g.Store.orm().QueryTable(new(models.AgentMCPServer)).Filter("id", serverID).Update(orm.Params{
		"oauth_status": "authorization_required", "last_error": message,
		"last_error_at": now, "updated_at": now,
	})
}

func OAuthClientMetadata() (map[string]any, error) {
	_, metadataURL, callbackURL, err := oauthPublicURLs()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"client_id":                  metadataURL,
		"client_name":                "go-binance-futures Agent Host",
		"client_uri":                 strings.TrimSuffix(metadataURL, "/agents/mcp/oauth/client-metadata"),
		"redirect_uris":              []string{callbackURL},
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"application_type":           "web",
	}, nil
}
