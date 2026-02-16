// Package oidc provides a lightweight OIDC provider that delegates
// authentication to tinyauth via forwardauth.
package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ClientConfig represents a registered OIDC client.
type ClientConfig struct {
	ID           string   `toml:"id"`
	Secret       string   `toml:"secret"`
	RedirectURIs []string `toml:"redirect_uris"`
}

// Config holds OIDC provider settings (from [oidc] in config.toml).
type Config struct {
	Enabled   bool           `toml:"enabled"`
	IssuerURL string         `toml:"issuer_url"`
	LoginURL  string         `toml:"login_url"`
	KeyPath   string         `toml:"key_path"`
	Clients   []ClientConfig `toml:"clients"`
}

// Provider is the OIDC provider that registers endpoints on a gin RouterGroup.
type Provider struct {
	cfg        Config
	verifyURL  string
	loginURL   string
	keyManager *keyManager
	codeStore  *codeStore
}

// New creates a new OIDC provider. verifyURL is tinyauth's forwardauth endpoint,
// loginURL is the public tinyauth login page.
func New(cfg Config, verifyURL string, loginURL string) (*Provider, error) {
	if cfg.KeyPath == "" {
		cfg.KeyPath = "/data/oidc-keys"
	}
	// If login_url is set in OIDC config, use it (public URL for browser redirects).
	// Otherwise fall back to the internal TINYAUTH_BASEURL.
	if cfg.LoginURL != "" {
		loginURL = cfg.LoginURL
	}

	km, err := newKeyManager(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("oidc key init: %w", err)
	}

	return &Provider{
		cfg:        cfg,
		verifyURL:  verifyURL,
		loginURL:   loginURL,
		keyManager: km,
		codeStore:  newCodeStore(60 * time.Second),
	}, nil
}

// Register mounts OIDC endpoints on the given router group (e.g. /oidc).
func (p *Provider) Register(group *gin.RouterGroup) {
	group.GET("/.well-known/openid-configuration", p.discovery)
	group.GET("/jwks", p.jwks)
	group.GET("/authorize", p.authorize)
	group.POST("/token", p.token)
	group.GET("/userinfo", p.userinfo)
}

// --- Discovery ---

func (p *Provider) discovery(c *gin.Context) {
	iss := p.cfg.IssuerURL
	c.JSON(200, gin.H{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/authorize",
		"token_endpoint":                        iss + "/token",
		"userinfo_endpoint":                     iss + "/userinfo",
		"jwks_uri":                              iss + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
	})
}

func (p *Provider) jwks(c *gin.Context) {
	c.JSON(200, p.keyManager.jwks())
}

// --- Authorize ---

func (p *Provider) authorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	scope := c.Query("scope")
	state := c.Query("state")
	nonce := c.Query("nonce")

	if clientID == "" || redirectURI == "" || responseType == "" || scope == "" {
		c.JSON(400, gin.H{"error": "missing required parameters"})
		return
	}
	if responseType != "code" {
		c.JSON(400, gin.H{"error": "unsupported response_type"})
		return
	}
	if !strings.Contains(scope, "openid") {
		c.JSON(400, gin.H{"error": "scope must include openid"})
		return
	}

	client := p.findClient(clientID)
	if client == nil {
		c.JSON(400, gin.H{"error": "unknown client_id"})
		return
	}
	if !validateRedirectURI(client, redirectURI) {
		c.JSON(400, gin.H{"error": "invalid redirect_uri"})
		return
	}

	// Check tinyauth session
	user, email, name, ok := p.validateSession(c.Request)
	if !ok {
		// Build the full absolute URL so tinyauth can redirect back after login.
		// c.Request.URL only contains the path+query; prepend the issuer base.
		issuerBase := strings.TrimSuffix(p.cfg.IssuerURL, "/")
		// IssuerURL is e.g. https://auth.hommers.nl/oidc, request path is /oidc/authorize?...
		// We need the base domain part only.
		if idx := strings.Index(issuerBase, "/oidc"); idx > 0 {
			issuerBase = issuerBase[:idx]
		}
		fullURL := issuerBase + c.Request.URL.String()
		loginURL := p.loginURL + "?redirect_uri=" + url.QueryEscape(fullURL)
		c.Redirect(302, loginURL)
		return
	}

	code := p.codeStore.store(&codeData{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		UserID:      user,
		Email:       email,
		Name:        name,
		Nonce:       nonce,
	})

	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	c.Redirect(302, u.String())
}

// --- Token ---

func (p *Provider) token(c *gin.Context) {
	if c.PostForm("grant_type") != "authorization_code" {
		c.JSON(400, gin.H{"error": "unsupported_grant_type"})
		return
	}

	code := c.PostForm("code")
	redirectURI := c.PostForm("redirect_uri")

	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	if clientID == "" {
		var ok bool
		clientID, clientSecret, ok = c.Request.BasicAuth()
		if !ok {
			c.JSON(401, gin.H{"error": "invalid_client"})
			return
		}
	}

	client := p.findClient(clientID)
	if client == nil || client.Secret != clientSecret {
		c.JSON(401, gin.H{"error": "invalid_client"})
		return
	}

	data, ok := p.codeStore.retrieve(code)
	if !ok || data.ClientID != clientID || data.RedirectURI != redirectURI {
		c.JSON(400, gin.H{"error": "invalid_grant"})
		return
	}

	now := time.Now()
	exp := now.Add(time.Hour)

	idClaims := jwt.MapClaims{
		"iss":                p.cfg.IssuerURL,
		"sub":                data.UserID,
		"aud":                clientID,
		"exp":                exp.Unix(),
		"iat":                now.Unix(),
		"email":              data.Email,
		"name":               data.Name,
		"preferred_username": data.UserID,
	}
	if data.Nonce != "" {
		idClaims["nonce"] = data.Nonce
	}

	idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, idClaims)
	idToken.Header["kid"] = p.keyManager.kid
	idTokenStr, err := idToken.SignedString(p.keyManager.privateKey)
	if err != nil {
		c.JSON(500, gin.H{"error": "server_error"})
		return
	}

	accessClaims := jwt.MapClaims{
		"iss":                p.cfg.IssuerURL,
		"sub":                data.UserID,
		"exp":                exp.Unix(),
		"iat":                now.Unix(),
		"email":              data.Email,
		"name":               data.Name,
		"preferred_username": data.UserID,
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessToken.Header["kid"] = p.keyManager.kid
	accessTokenStr, err := accessToken.SignedString(p.keyManager.privateKey)
	if err != nil {
		c.JSON(500, gin.H{"error": "server_error"})
		return
	}

	c.JSON(200, gin.H{
		"access_token": accessTokenStr,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     idTokenStr,
	})
}

// --- Userinfo ---

func (p *Provider) userinfo(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(401, gin.H{"error": "missing token"})
		return
	}

	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return p.keyManager.publicKey(), nil
	})
	if err != nil || !token.Valid {
		c.JSON(401, gin.H{"error": "invalid token"})
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	c.JSON(200, gin.H{
		"sub":                claims["sub"],
		"email":              claims["email"],
		"name":               claims["name"],
		"preferred_username": claims["preferred_username"],
	})
}

// --- Session validation (forwardauth) ---

func (p *Provider) validateSession(r *http.Request) (string, string, string, bool) {
	req, err := http.NewRequest("GET", p.verifyURL, nil)
	if err != nil {
		return "", "", "", false
	}
	req.Header.Set("Cookie", r.Header.Get("Cookie"))
	req.Header.Set("X-Forwarded-Host", r.Host)
	req.Header.Set("X-Forwarded-Uri", r.URL.Path)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", "", false
	}

	user := resp.Header.Get("Remote-User")
	if user == "" {
		return "", "", "", false
	}
	return user, resp.Header.Get("Remote-Email"), resp.Header.Get("Remote-Name"), true
}

// --- Helpers ---

func (p *Provider) findClient(id string) *ClientConfig {
	for i := range p.cfg.Clients {
		if p.cfg.Clients[i].ID == id {
			return &p.cfg.Clients[i]
		}
	}
	return nil
}

func validateRedirectURI(client *ClientConfig, uri string) bool {
	for _, allowed := range client.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// --- Key Manager ---

type jwkKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwkSet struct {
	Keys []jwkKey `json:"keys"`
}

type keyManager struct {
	privateKey *rsa.PrivateKey
	kid        string
}

func newKeyManager(keyPath string) (*keyManager, error) {
	if err := os.MkdirAll(keyPath, 0700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}

	privPath := filepath.Join(keyPath, "private.pem")
	pubPath := filepath.Join(keyPath, "public.pem")

	var key *rsa.PrivateKey

	if data, err := os.ReadFile(privPath); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("failed to decode private key PEM")
		}
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
	} else {
		key, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}

		privPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})
		if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
			return nil, fmt.Errorf("write private key: %w", err)
		}

		pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("marshal public key: %w", err)
		}
		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubDER,
		})
		if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
			return nil, fmt.Errorf("write public key: %w", err)
		}
	}

	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	hash := sha256.Sum256(pubDER)
	kid := base64.RawURLEncoding.EncodeToString(hash[:8])

	return &keyManager{privateKey: key, kid: kid}, nil
}

func (km *keyManager) publicKey() *rsa.PublicKey {
	return &km.privateKey.PublicKey
}

func (km *keyManager) jwks() jwkSet {
	pub := km.publicKey()
	return jwkSet{
		Keys: []jwkKey{
			{
				Kty: "RSA",
				Use: "sig",
				Alg: "RS256",
				Kid: km.kid,
				N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}

// --- Code Store ---

type codeData struct {
	ClientID    string
	RedirectURI string
	UserID      string
	Email       string
	Name        string
	Nonce       string
	ExpiresAt   time.Time
}

type codeStore struct {
	mu    sync.Mutex
	codes map[string]*codeData
	ttl   time.Duration
}

func newCodeStore(ttl time.Duration) *codeStore {
	return &codeStore{
		codes: make(map[string]*codeData),
		ttl:   ttl,
	}
}

func (s *codeStore) store(data *codeData) string {
	b := make([]byte, 32)
	rand.Read(b)
	code := hex.EncodeToString(b)
	data.ExpiresAt = time.Now().Add(s.ttl)

	s.mu.Lock()
	s.codes[code] = data
	s.mu.Unlock()
	return code
}

func (s *codeStore) retrieve(code string) (*codeData, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.codes[code]
	if !ok {
		return nil, false
	}
	delete(s.codes, code)

	if time.Now().After(data.ExpiresAt) {
		return nil, false
	}
	return data, true
}
