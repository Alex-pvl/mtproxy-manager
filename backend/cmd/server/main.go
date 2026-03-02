package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/docker"
	"mtproxy-manager/internal/handlers"
	"mtproxy-manager/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	dockerMgr, err := docker.NewManager(cfg, db)
	if err != nil {
		log.Fatalf("docker: %v", err)
	}
	defer dockerMgr.Close()

	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	authHandler := handlers.NewAuthHandler(db, jwtSvc)
	telegramHandler := handlers.NewTelegramHandler(db, jwtSvc, cfg)
	oidcHandler := handlers.NewOIDCHandler(db, jwtSvc, cfg)
	proxyHandler := handlers.NewProxyHandler(db, dockerMgr)
	adminHandler := handlers.NewAdminHandler(db, dockerMgr)
	paymentHandler := handlers.NewPaymentHandler(db, cfg)
	referralHandler := handlers.NewReferralHandler(db, cfg)

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/telegram", telegramHandler.Auth)
			r.Get("/oidc/init", oidcHandler.Init)
			r.Get("/oidc/callback", oidcHandler.Callback)
			r.With(middleware.AuthRequired(jwtSvc)).Get("/me", authHandler.Me)
		})

		r.Route("/proxies", func(r chi.Router) {
			r.Use(middleware.AuthRequired(jwtSvc))
			r.Get("/", proxyHandler.List)
			r.Post("/", proxyHandler.Create)
			r.Post("/{id}/stop", proxyHandler.Stop)
			r.Post("/{id}/start", proxyHandler.Start)
			r.Delete("/{id}", proxyHandler.Delete)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.AuthRequired(jwtSvc))
			r.Use(middleware.AdminRequired)
			r.Get("/users", adminHandler.ListUsers)
			r.Put("/users/{id}", adminHandler.UpdateUser)
			r.Delete("/users/{id}", adminHandler.DeleteUser)
			r.Get("/proxies", adminHandler.ListAllProxies)
			r.Delete("/proxies/{id}", adminHandler.DeleteProxy)
		})

		r.Get("/plans", paymentHandler.ListPlans)
		r.Post("/payments/webhook", paymentHandler.Webhook)

		r.Route("/payments", func(r chi.Router) {
			r.Use(middleware.AuthRequired(jwtSvc))
			r.Post("/create", paymentHandler.CreatePayment)
		})

		r.With(middleware.AuthRequired(jwtSvc)).Get("/subscription", paymentHandler.GetSubscription)
		r.With(middleware.AuthRequired(jwtSvc)).Get("/referral", referralHandler.Get)
	})

	// Serve frontend static files (embedded or from disk)
	staticDir := "./frontend/dist"
	if _, err := os.Stat(staticDir); err == nil {
		fileServer(r, staticDir)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("Starting server on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func fileServer(r chi.Router, dir string) {
	fs := http.FileServer(http.Dir(dir))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try serving the file directly; fall back to index.html for SPA routing
		path := dir + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, dir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})
}
