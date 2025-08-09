package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wtotem-test/internal/config"
	"wtotem-test/internal/httpapi"
	"wtotem-test/internal/mailer"
	"wtotem-test/internal/report"
	"wtotem-test/internal/zipper"
)

func main() {
	// HTTP-only app: request carries cv_url/email; SMTP/env come from environment.
	cfg := config.FromFlagsOrEnv(
		"", "", // cvURL, email — HTTP body
		os.Getenv("SMTP_LOGIN"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_SERVER"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("ADDR"),
		os.Getenv("PROJECT_DIR"),
		os.Getenv("TARGET_EMAIL"),
	)

	// Reasonable defaults
	if cfg.TargetEmail == "" {
		cfg.TargetEmail = "szhaisan@wtotem.com"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = "."
	}
	if cfg.SMTP.Server == "" {
		cfg.SMTP.Server = "smtp.gmail.com"
	}
	if cfg.SMTP.Port == "" {
		cfg.SMTP.Port = "587"
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	// Dependencies
	sender := mailer.New(mailer.SMTP{
		Login:    cfg.SMTP.Login,
		Password: cfg.SMTP.Password,
		Server:   cfg.SMTP.Server,
		Port:     cfg.SMTP.Port,
	})
	rb := report.NewBuilder()
	z := zipper.New()

	// Server
	srv := httpapi.NewServer(cfg, rb, z, sender)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	}
}
