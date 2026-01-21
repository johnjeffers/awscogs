package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johnjeffers/awscogs/backend/internal/api"
	"github.com/johnjeffers/awscogs/backend/internal/aws"
	"github.com/johnjeffers/awscogs/backend/internal/config"
	"github.com/johnjeffers/awscogs/backend/internal/pricing"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create pricing provider
	ctx := context.Background()
	pricingProvider, err := pricing.NewAWSProvider(ctx, cfg.Pricing.RefreshIntervalMinutes, cfg.Pricing.RateLimitPerSecond)
	if err != nil {
		logger.Error("failed to initialize AWS pricing provider", "error", err)
		os.Exit(1)
	}
	logger.Info("pricing provider initialized", "rateLimitPerSecond", cfg.Pricing.RateLimitPerSecond)

	// Create discovery service
	discovery := aws.NewDiscovery(pricingProvider, logger, cfg.Cache.ResourceTTLMinutes, cfg.Cache.AccountTTLMinutes)
	logger.Info("discovery service initialized", "resourceCacheTTL", cfg.Cache.ResourceTTLMinutes, "accountCacheTTL", cfg.Cache.AccountTTLMinutes)

	// Create and start server
	server := api.NewServer(cfg, discovery, logger)

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			logger.Error("server error", "error", err)
		}
	}()

	logger.Info("awscogs started", "port", cfg.Server.Port)

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("awscogs stopped")
}
