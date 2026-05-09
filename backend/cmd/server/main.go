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
	"mtproxy-manager/internal/xui"

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

	// Initialize x-ui client only when XUI_URL is configured
	var xuiClient *xui.Client
	if cfg.XUIEnabled {
		xuiClient, err = xui.NewClient(cfg.XUIURL, cfg.XUIPathPrefix, cfg.XUIUsername, cfg.XUIPassword, cfg.XUIInboundID)
		if err != nil {
			log.Printf("WARNING: x-ui integration disabled — %v", err)
			xuiClient = nil
		} else {
			log.Printf("x-ui integration enabled (inbound id=%d)", cfg.XUIInboundID)
		}
	}

	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	authHandler := handlers.NewAuthHandler(db, jwtSvc)
	telegramHandler := handlers.NewTelegramHandler(db, jwtSvc, cfg)
	oidcHandler := handlers.NewOIDCHandler(db, jwtSvc, cfg)
	webAppHandler := handlers.NewWebAppHandler(db, jwtSvc, cfg)
	proxyHandler := handlers.NewProxyHandler(db, dockerMgr, xuiClient)
	adminHandler := handlers.NewAdminHandler(db, dockerMgr)
	paymentHandler := handlers.NewPaymentHandler(db, cfg, xuiClient)
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
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/telegram", telegramHandler.Auth)
			r.Post("/webapp", webAppHandler.Auth)
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
		r.Post("/webhook/bot", paymentHandler.BotWebhook)
		r.Post("/payments/sbp/webhook", paymentHandler.DigitalPayWebhook)

		r.Route("/payments", func(r chi.Router) {
			r.Use(middleware.AuthRequired(jwtSvc))
			r.Post("/create", paymentHandler.CreatePayment)
			r.Post("/sbp/create", paymentHandler.CreateSBPPayment)
			r.Post("/stars/create", paymentHandler.CreateStarsPayment)
			r.Post("/ton/create", paymentHandler.CreateTonPayment)
			r.Post("/check-pending", paymentHandler.CheckPendingPayments)
		})

		// Telegram photo proxy (no auth — proxies bot avatars without exposing token)
		r.Get("/tg/photo", paymentHandler.ServeTelegramPhoto)

		// Stars / Premium via Fragment
		r.Route("/products", func(r chi.Router) {
			r.Use(middleware.AuthRequired(jwtSvc))
			r.Get("/stars/quote", paymentHandler.GetStarsQuote)
			r.Get("/premium/quote", paymentHandler.GetPremiumQuote)
			r.Get("/username/check", paymentHandler.CheckUsername)
			r.Post("/order", paymentHandler.CreateProductOrder)
			r.Get("/orders/{id}", paymentHandler.GetOrderStatus)
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

	// Subscription expiry reminders (7d / 1d) via Telegram; UTC calendar-day match, idempotent flags on subscriptions.
	go func() {
		run := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("RunSubscriptionExpiryReminders panic: %v", r)
				}
			}()
			paymentHandler.RunSubscriptionExpiryReminders()
		}
		time.Sleep(2 * time.Minute)
		run()
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			run()
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
