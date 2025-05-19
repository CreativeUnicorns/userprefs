// Package main is the entry point for the userprefs-server application.
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/api"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	// Basic flag for listen address
	listenAddr := flag.String("listen-addr", ":8080", "HTTP listen address")
	// TODO: Add flags for storage type (postgres, sqlite, memory), DSNs, cache type (redis, memory), etc.
	flag.Parse()

	// Setup logger
	// TODO: Make log level configurable via flag
	logger := userprefs.NewDefaultLogger()
	logger.Info("Userprefs server starting up...")

	// Setup storage (example: in-memory for now)
	// In a real scenario, this would be configured via flags/env vars
	s := storage.NewMemoryStorage()
	var store userprefs.Storage = s

	// Setup cache (example: in-memory for now)
	ca := cache.NewMemoryCache()
	var cacher userprefs.Cache = ca

	// Setup manager
	mgr := userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(cacher),
		userprefs.WithLogger(logger),
	)

	// Define some sample preferences (for testing/demonstration)
	if err := mgr.DefinePreference(userprefs.PreferenceDefinition{Key: "theme", Type: userprefs.StringType, DefaultValue: "dark", Category: "appearance"}); err != nil {
		logger.Error("Failed to define preference 'theme'", "error", err)
		os.Exit(1)
	}
	if err := mgr.DefinePreference(userprefs.PreferenceDefinition{Key: "notifications.enabled", Type: userprefs.BoolType, DefaultValue: true, Category: "notifications"}); err != nil {
		logger.Error("Failed to define preference 'notifications.enabled'", "error", err)
		os.Exit(1)
	}

	// Setup API server
	apiCfg := api.Config{
		ListenAddress: *listenAddr,
		Manager:       mgr,
		Logger:        logger,
	}
	apiServer, err := api.NewServer(apiCfg)
	if err != nil {
		logger.Error("Failed to create API server", "error", err)
		os.Exit(1)
	}

	// Start server in a goroutine
	go func() {
		if err := apiServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API server error", "error", err)
			os.Exit(1) // or handle more gracefully
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Adjust timeout as needed
	defer cancel()

	if err := apiServer.Stop(ctx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
	}

	// Close storage and cache
	// Assuming MemoryStorage might not have a Close() method or it's not needed for this simple type.
	// If it did, and it returned an error: if err := s.Close(); err != nil { logger.Error(...) }

	if err := ca.Close(); err != nil { // *cache.MemoryCache has a Close() error method
		logger.Error("Failed to close cache", "error", err)
	}

	logger.Info("Server exited gracefully")
}
