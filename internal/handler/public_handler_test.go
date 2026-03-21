package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tinyauth-sidecar/internal/config"
	"tinyauth-sidecar/internal/middleware"
	"tinyauth-sidecar/internal/service"
	"tinyauth-sidecar/internal/store"

	"github.com/gin-gonic/gin"
)

// setupPublicRouter creates a minimal gin engine with only the public handler
// registered, matching how main.go wires it. No session middleware on this group.
func setupPublicRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	st, _ := store.NewStore("/tmp/tinyauth-test-public-handler.toml")
	usersSvc := service.NewUserFileService(cfg)
	dockerSvc := service.NewDockerService(cfg)
	auditSvc := service.NewAuditService("")
	accountSvc := service.NewAccountService(cfg, st, usersSvc, nil, dockerSvc, nil, nil, auditSvc)

	api := r.Group("/manage/api")

	// Only register public handler — no authed group
	rl := middleware.PerMinute(100)
	pub := NewPublicHandler(accountSvc, cfg)
	pub.Register(api, rl, rl, rl)

	return r
}

func TestPublicSignupApproveNotExposed(t *testing.T) {
	cfg := &config.Config{
		DisableSignup: false,
	}
	r := setupPublicRouter(cfg)

	body, _ := json.Marshal(map[string]string{"id": "fake-id"})
	req := httptest.NewRequest(http.MethodPost, "/manage/api/signup/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// /signup/approve should NOT be publicly reachable; expect 404
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for public /signup/approve, got %d", w.Code)
	}
}

func TestPublicSignupDoesNotLeakApprovalID(t *testing.T) {
	cfg := &config.Config{
		DisableSignup:         false,
		SignupRequireApproval: true,
		UsernameIsEmail:       true,
		MinPasswordLength:     8,
		MinPasswordStrength:   0,
		UsersFilePath:         "/tmp/tinyauth-test-users.txt",
	}
	r := setupPublicRouter(cfg)

	body, _ := json.Marshal(map[string]string{
		"username": "test@example.com",
		"password": "CorrectHorseBatteryStaple!",
	})
	req := httptest.NewRequest(http.MethodPost, "/manage/api/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for signup, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// status should be "pending", not a UUID
	status, ok := resp["status"]
	if !ok || status != "pending" {
		t.Fatalf("expected status=pending, got %v", resp)
	}

	// There must be no "id" field exposed
	if _, hasID := resp["id"]; hasID {
		t.Fatalf("response must not contain approval id, got: %v", resp)
	}
}
