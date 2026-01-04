package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/oskargbc/dws-ticket-service/configs"
	"github.com/oskargbc/dws-ticket-service/internal/controllers/health"
	"github.com/oskargbc/dws-ticket-service/internal/controllers/tickets"
	"github.com/oskargbc/dws-ticket-service/internal/middlewares"
	"github.com/oskargbc/dws-ticket-service/internal/pkg/rabbitmq"
	"github.com/oskargbc/dws-ticket-service/internal/services"
	log "github.com/sirupsen/logrus"
)

func SetupRouter(cfg *configs.Config, dbService *services.DatabaseService, rmqService *rabbitmq.RabbitMQService) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger())

	// CORS configuration
	corsConfig := cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		AllowCredentials: true,
	}
	router.Use(cors.New(corsConfig))

	// Initialize controllers
	healthController := health.NewHealthController(dbService, rmqService)
	ticketsController := tickets.NewTicketsController(dbService, rmqService)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health routes (no auth required)
		healthGroup := v1.Group("/health")
		{
			healthGroup.GET("", healthController.HealthCheck)
			healthGroup.GET("/db", healthController.DatabaseHealth)
			healthGroup.GET("/rabbitmq", healthController.RabbitMQHealth)
		}

		// Tickets routes (auth required)
		ticketsGroup := v1.Group("/tickets")
		ticketsGroup.Use(middlewares.KeycloakAuthMiddleware(cfg))
		{
			ticketsGroup.POST("/purchase", ticketsController.PurchaseTicket)
			ticketsGroup.GET("/my-tickets", ticketsController.GetMyTickets)
			ticketsGroup.GET("/:id", ticketsController.GetTicketByID)
			ticketsGroup.DELETE("/:id", ticketsController.CancelTicket)
		}
	}

	return router
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		log.WithFields(log.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("Request processed")
	}
}
