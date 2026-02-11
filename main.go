package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/handler"
	"tinyauth-usermanagement/internal/middleware"
	"tinyauth-usermanagement/internal/service"
	"tinyauth-usermanagement/internal/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist frontend/dist/*
var frontendFS embed.FS

func main() {
	cfg := config.Load()

	sqliteStore, err := store.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to init sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	usersSvc := service.NewUserFileService(cfg)
	mailSvc := service.NewMailService(cfg)
	dockerSvc := service.NewDockerService(cfg)
	authSvc := service.NewAuthService(cfg, sqliteStore, usersSvc)
	accountSvc := service.NewAccountService(cfg, sqliteStore, usersSvc, mailSvc, dockerSvc)

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
		authed.Use(middleware.SessionMiddleware(cfg, sqliteStore))
		accountHandler := handler.NewAccountHandler(accountSvc)
		accountHandler.Register(authed)
	}

	serveSPA(r)

	log.Printf("tinyauth-usermanagement listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

func serveSPA(r *gin.Engine) {
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Printf("frontend dist not embedded yet: %v", err)
		return
	}

	fileServer := http.FileServer(http.FS(distFS))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			c.Request.URL.Path = "/index.html"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
