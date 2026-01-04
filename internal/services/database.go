package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/oskargbc/dws-ticket-service/configs"
	"github.com/oskargbc/dws-ticket-service/prisma/db"
	log "github.com/sirupsen/logrus"
)

var (
	dbInstance *DatabaseService
	dbOnce     sync.Once
)

type DatabaseService struct {
	Client *db.PrismaClient
	mu     sync.RWMutex
}

func GetDatabaseServiceInstance(cfg *configs.Config) (*DatabaseService, error) {
	var err error
	dbOnce.Do(func() {
		client := db.NewClient()
		if connectErr := client.Prisma.Connect(); connectErr != nil {
			err = fmt.Errorf("failed to connect to database: %w", connectErr)
			log.WithError(connectErr).Error("Database connection failed")
			return
		}

		log.Info("Successfully connected to database")
		dbInstance = &DatabaseService{
			Client: client,
		}
	})

	if err != nil {
		return nil, err
	}

	if dbInstance == nil {
		return nil, fmt.Errorf("database instance is nil")
	}

	return dbInstance, nil
}

func (ds *DatabaseService) Disconnect() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.Client != nil {
		if err := ds.Client.Prisma.Disconnect(); err != nil {
			log.WithError(err).Error("Failed to disconnect from database")
			return err
		}
		log.Info("Successfully disconnected from database")
	}
	return nil
}

func (ds *DatabaseService) HealthCheck(ctx context.Context) error {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.Client == nil {
		return fmt.Errorf("database client is nil")
	}

	// Simple query to check connection
	_, err := ds.Client.Ticket.FindMany().Exec(ctx)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
