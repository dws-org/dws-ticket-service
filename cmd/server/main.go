package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oskargbc/dws-ticket-service/configs"
	"github.com/oskargbc/dws-ticket-service/internal/pkg/rabbitmq"
	"github.com/oskargbc/dws-ticket-service/internal/router"
	"github.com/oskargbc/dws-ticket-service/internal/services"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Configure logging
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// Load configuration
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level from config
	if level, err := log.ParseLevel(cfg.Logging.Level); err == nil {
		log.SetLevel(level)
	}

	log.WithFields(log.Fields{
		"port":        cfg.Server.Port,
		"environment": cfg.Server.Environment,
	}).Info("Starting dws-ticket-service")

	// Initialize database
	dbService, err := services.GetDatabaseServiceInstance(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize database service")
	}
	defer func() {
		if err := dbService.Disconnect(); err != nil {
			log.WithError(err).Error("Failed to disconnect from database")
		}
	}()

	// Initialize RabbitMQ
	rmqService, err := rabbitmq.GetRabbitMQServiceInstance(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize RabbitMQ service")
	}
	defer func() {
		if err := rmqService.Close(); err != nil {
			log.WithError(err).Error("Failed to close RabbitMQ connection")
		}
	}()

	// Setup router
	r := router.SetupRouter(cfg, dbService, rmqService)

	// Create HTTP server
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in goroutine
	go func() {
		log.WithField("port", cfg.Server.Port).Info("Server is running")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}
