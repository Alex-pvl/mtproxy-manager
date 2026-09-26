package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
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

	var xuiClient *xui.Client
	if cfg.XUIURL != "" {
		xuiClient, err = xui.NewClient(cfg.XUIURL, cfg.XUIPathPrefix, cfg.XUIAPIToken, cfg.XUISubURL, cfg.XUIInboundID)
		if err != nil {
			log.Printf("WARNING: VPN disabled, x-ui unavailable: %v", err)
			xuiClient = nil
		} else {
			log.Printf("x-ui connected (inbound id=%d)", cfg.XUIInboundID)
		}
	}

	jwtSvc := auth.NewJWTService(cfg.JWTSecret)
	vpn := handlers.NewVPN(db, xuiClient)
	bot := handlers.NewBot(cfg, db)

	authHandler := handlers.NewAuthHandler(db, jwtSvc)
	oidcHandler := handlers.NewOIDCHandler(db, jwtSvc, cfg)
	webAppHandler := handlers.NewWebAppHandler(db, jwtSvc, cfg)
	proxyHandler := handlers.NewProxyHandler(db, vpn)
	adminHandler := handlers.NewAdminHandler(db, vpn)
	paymentHandler := handlers.NewPaymentHandler(db, cfg, vpn, bot)
	referralHandler := handlers.NewReferralHandler(db, cfg)

	r := chi.NewRouter()
	r.Use(chimw.Logger, chimw.Recoverer, chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:         300,
	}))

	r.Route("/api", func(r chi.Router) {
		// Public
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/webapp", webAppHandler.Auth)
		r.Get("/auth/oidc/init", oidcHandler.Init)
		r.Get("/auth/oidc/callback", oidcHandler.Callback)
		r.Get("/plans", paymentHandler.ListPlans)

		// Provider webhooks (each verifies its own authenticity)
		r.Post("/payments/webhook", paymentHandler.Webhook)
		r.Post("/payments/sbp/webhook", paymentHandler.DigitalPayWebhook)
		r.Post("/webhook/bot", paymentHandler.BotWebhook)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired(jwtSvc))
			r.Get("/auth/me", authHandler.Me)
			r.Get("/subscription", paymentHandler.GetSubscription)
			r.Get("/referral", referralHandler.Get)

			r.Get("/proxies", proxyHandler.List)
			r.Post("/proxies", proxyHandler.Create)
			r.Delete("/proxies/{id}", proxyHandler.Delete)

			r.Post("/payments/create", paymentHandler.CreatePayment)
			r.Post("/payments/sbp/create", paymentHandler.CreateSBPPayment)
			r.Post("/payments/stars/create", paymentHandler.CreateStarsPayment)
			r.Post("/payments/ton/create", paymentHandler.CreateTonPayment)
			r.Post("/payments/check-pending", paymentHandler.CheckPendingPayments)

			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.AdminRequired)
				r.Get("/users", adminHandler.ListUsers)
				r.Put("/users/{id}", adminHandler.UpdateUser)
				r.Delete("/users/{id}", adminHandler.DeleteUser)
				r.Get("/proxies", adminHandler.ListAllProxies)
				r.Delete("/proxies/{id}", adminHandler.DeleteProxy)
			})
		})
	})

	if dir := "./frontend/dist"; dirExists(dir) {
		serveSPA(r, dir)
	}

	srv := &http.Server{Addr: ":" + cfg.ServerPort, Handler: r}
	go func() {
		log.Printf("listening on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()
	go every(6*time.Hour, 2*time.Minute, bot.RunExpiryReminders)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// every runs job after delay and then on each interval, surviving panics.
func every(interval, delay time.Duration, job func()) {
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("background job panic: %v", r)
			}
		}()
		job()
	}
	time.Sleep(delay)
	for {
		run()
		time.Sleep(interval)
	}
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// serveSPA serves built frontend files and falls back to index.html for client routes.
func serveSPA(r chi.Router, dir string) {
	files := http.FileServer(http.Dir(dir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		if _, err := os.Stat(filepath.Join(dir, filepath.Clean("/"+req.URL.Path))); os.IsNotExist(err) {
			http.ServeFile(w, req, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, req)
	})
}
