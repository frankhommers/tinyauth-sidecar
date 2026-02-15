package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/handler"
	"tinyauth-usermanagement/internal/middleware"
	"tinyauth-usermanagement/internal/provider"
	"tinyauth-usermanagement/internal/service"
	"tinyauth-usermanagement/internal/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist frontend/dist/*
var frontendFS embed.FS

func main() {
	cfg := config.Load()

	st, err := store.NewStore("")
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}
	defer st.Close()

	// Initialize providers
	fileCfg := config.LoadFileConfig()

	// Password policy: config.toml takes precedence over env vars
	if fileCfg.PasswordPolicy.MinLength > 0 {
		cfg.MinPasswordLength = fileCfg.PasswordPolicy.MinLength
	}
	if fileCfg.PasswordPolicy.MinStrength > 0 {
		cfg.MinPasswordStrength = fileCfg.PasswordPolicy.MinStrength
	}

	// Username is email: config.toml takes precedence over env var
	if fileCfg.Users.UsernameIsEmail != nil {
		cfg.UsernameIsEmail = *fileCfg.Users.UsernameIsEmail
	}

	passwordTargets := provider.NewPasswordTargetProvider()
	var passwordHooks []provider.PasswordChangeHook
	for _, hookCfg := range fileCfg.PasswordHooks {
		if h := provider.NewWebhookPasswordHook(hookCfg); h != nil {
			passwordHooks = append(passwordHooks, h)
		}
	}

	// SMS: config.toml takes precedence, fall back to env vars
	smsProvider := provider.NewWebhookSMSProviderFromConfig(fileCfg.SMS)
	if smsProvider == nil {
		smsProvider = provider.NewWebhookSMSProvider()
	}

	usersSvc := service.NewUserFileService(cfg)
	mailSvc := service.NewMailService(cfg)
	dockerSvc := service.NewDockerService(cfg)
	accountSvc := service.NewAccountService(cfg, st, usersSvc, mailSvc, dockerSvc, passwordTargets, smsProvider, passwordHooks...)

	r := gin.Default()

	// Security headers on all responses
	r.Use(middleware.SecurityHeaders())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	// Rate limiters for sensitive public endpoints
	resetEmailRL := middleware.PerMinute(3)
	forgotSmsRL := middleware.PerMinute(3)
	resetSmsRL := middleware.PerMinute(5)

	api := r.Group("/manage/api")
	{
		// CSRF protection on all API endpoints (validates POST/PUT/DELETE)
		api.Use(middleware.CSRFMiddleware())

		// Public endpoints (no auth required)
		public := handler.NewPublicHandler(accountSvc, cfg)
		public.Register(api, resetEmailRL, forgotSmsRL, resetSmsRL)

		// Auth check and logout (behind tinyauth middleware)
		authed := api.Group("")
		authed.Use(middleware.SessionMiddleware(cfg))

		// Auth endpoints
		authHandler := handler.NewAuthHandler(cfg)
		authHandler.Register(authed)

		// Account management endpoints
		accountHandler := handler.NewAccountHandler(accountSvc)
		accountHandler.Register(authed)
	}

	serveSPA(r)

	log.Printf("tinyauth-usermanagement listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

type spaHandler struct {
	fs       fs.FS
	basePath string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if h.basePath != "" && strings.HasPrefix(path, h.basePath) {
		path = strings.TrimPrefix(path, h.basePath)
	}
	if path == "" || path == "/" {
		path = "index.html"
	} else if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	f, err := h.fs.Open(path)
	if err != nil {
		// SPA fallback
		path = "index.html"
		f, err = h.fs.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		path = "index.html"
		f2, err := h.fs.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		f.Close()
		f = f2
		stat, _ = f.Stat()
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f.(io.ReadSeeker))
}

func serveSPA(r *gin.Engine) {
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Printf("frontend dist not embedded yet: %v", err)
		return
	}

	spa := spaHandler{fs: distFS, basePath: "/manage"}
	r.NoRoute(gin.WrapH(spa))
}
