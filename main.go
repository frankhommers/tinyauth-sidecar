package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"

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
	passwordTargets := provider.NewPasswordTargetProvider()
	passwordHook := provider.NewWebhookPasswordHook(fileCfg.PasswordHook)

	// SMS: config.toml takes precedence, fall back to env vars
	smsProvider := provider.NewWebhookSMSProviderFromConfig(fileCfg.SMS)
	if smsProvider == nil {
		smsProvider = provider.NewWebhookSMSProvider()
	}

	usersSvc := service.NewUserFileService(cfg)
	mailSvc := service.NewMailService(cfg)
	dockerSvc := service.NewDockerService(cfg)
	authSvc := service.NewAuthService(cfg, st, usersSvc)
	accountSvc := service.NewAccountService(cfg, st, usersSvc, mailSvc, dockerSvc, passwordTargets, smsProvider, passwordHook)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		authHandler := handler.NewAuthHandler(cfg, authSvc)
		authHandler.Register(api)

		public := handler.NewPublicHandler(accountSvc)
		public.Register(api)

		authed := api.Group("")
		authed.Use(middleware.SessionMiddleware(cfg, st))
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
	fs fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
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

	r.NoRoute(gin.WrapH(spaHandler{fs: distFS}))
}
