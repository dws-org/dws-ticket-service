package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oskargbc/dws-ticket-service/internal/pkg/rabbitmq"
	"github.com/oskargbc/dws-ticket-service/internal/services"
	"github.com/oskargbc/dws-ticket-service/internal/types"
)

type HealthController struct {
	dbService       *services.DatabaseService
	rabbitmqService *rabbitmq.RabbitMQService
}

func NewHealthController(db *services.DatabaseService, rmq *rabbitmq.RabbitMQService) *HealthController {
	return &HealthController{
		dbService:       db,
		rabbitmqService: rmq,
	}
}

func (h *HealthController) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, types.HealthResponse{
		Status: "healthy",
	})
}

func (h *HealthController) DatabaseHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.dbService.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, types.HealthResponse{
			Status: "unhealthy",
			Services: map[string]string{
				"database": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.HealthResponse{
		Status: "healthy",
		Services: map[string]string{
			"database": "connected",
		},
	})
}

func (h *HealthController) RabbitMQHealth(c *gin.Context) {
	if err := h.rabbitmqService.HealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, types.HealthResponse{
			Status: "unhealthy",
			Services: map[string]string{
				"rabbitmq": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.HealthResponse{
		Status: "healthy",
		Services: map[string]string{
			"rabbitmq": "connected",
		},
	})
}
