// ABOUTME: OIDC token verification for OAuth-based authentication.
// ABOUTME: Validates Bearer tokens against an Authelia (or any OIDC) issuer.

package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCVerifier validates OIDC tokens from the configured issuer.
type OIDCVerifier struct {
	provider    *oidc.Provider
	userinfoURL string
}

// NewOIDCVerifier creates a verifier using OIDC discovery on the issuer URL.
func NewOIDCVerifier(ctx context.Context, issuerURL, clientID string) (*OIDCVerifier, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	// Extract the userinfo endpoint from the provider's discovery document.
	var claims struct {
		UserinfoEndpoint string `json:"userinfo_endpoint"`
	}
	if err := provider.Claims(&claims); err != nil {
		return nil, fmt.Errorf("oidc claims: %w", err)
	}
	if claims.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("oidc: no userinfo_endpoint in discovery document")
	}

	return &OIDCVerifier{provider: provider, userinfoURL: claims.UserinfoEndpoint}, nil
}

// Verify validates a raw access token via the OIDC userinfo endpoint
// and returns the subject claim. This works for both JWT and opaque access tokens.
func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", v.userinfoURL, nil)
	if err != nil {
		return "", fmt.Errorf("userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo: status %d", resp.StatusCode)
	}

	var claims struct {
		Subject string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return "", fmt.Errorf("userinfo decode: %w", err)
	}
	if claims.Subject == "" {
		return "", fmt.Errorf("userinfo: empty sub claim")
	}
	return claims.Subject, nil
}
