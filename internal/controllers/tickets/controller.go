package tickets

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oskargbc/dws-ticket-service/internal/pkg/rabbitmq"
	"github.com/oskargbc/dws-ticket-service/internal/services"
	"github.com/oskargbc/dws-ticket-service/internal/types"
	"github.com/oskargbc/dws-ticket-service/prisma/db"
	log "github.com/sirupsen/logrus"
)

type TicketsController struct {
	dbService       *services.DatabaseService
	rabbitmqService *rabbitmq.RabbitMQService
}

func NewTicketsController(dbSvc *services.DatabaseService, rmqSvc *rabbitmq.RabbitMQService) *TicketsController {
	return &TicketsController{
		dbService:       dbSvc,
		rabbitmqService: rmqSvc,
	}
}

// PurchaseTicket handles POST /api/v1/tickets/purchase
func (tc *TicketsController) PurchaseTicket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	var req types.PurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create ticket with pending status
	ticket, err := tc.dbService.Client.Ticket.CreateOne(
		db.Ticket.UserID.Set(userID.(string)),
		db.Ticket.EventID.Set(req.EventID),
		db.Ticket.Quantity.Set(req.Quantity),
		db.Ticket.TotalPrice.Set(req.TotalPrice),
		db.Ticket.Status.Set("pending"),
	).Exec(ctx)

	if err != nil {
		log.WithError(err).Error("Failed to create ticket")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create ticket",
		})
		return
	}

	// Publish message to RabbitMQ
	msg := types.TicketMessage{
		TicketID:   ticket.ID,
		UserID:     ticket.UserID,
		EventID:    ticket.EventID,
		Quantity:   ticket.Quantity,
		TotalPrice: ticket.TotalPrice,
		Timestamp:  time.Now(),
	}

	if err := tc.rabbitmqService.PublishTicketPurchased(msg); err != nil {
		log.WithError(err).Error("Failed to publish message to RabbitMQ")
		// Don't fail the request, ticket is already created
	}

	c.JSON(http.StatusCreated, mapTicketToResponse(ticket))
}

// GetMyTickets handles GET /api/v1/tickets/my-tickets
func (tc *TicketsController) GetMyTickets(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tickets, err := tc.dbService.Client.Ticket.FindMany(
		db.Ticket.UserID.Equals(userID.(string)),
	).OrderBy(
		db.Ticket.CreatedAt.Order(db.DESC),
	).Exec(ctx)

	if err != nil {
		log.WithError(err).Error("Failed to fetch tickets")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to fetch tickets",
		})
		return
	}

	response := make([]types.TicketResponse, len(tickets))
	for i, ticket := range tickets {
		response[i] = mapTicketToResponse(&ticket)
	}

	c.JSON(http.StatusOK, response)
}

// GetAllTickets handles GET /api/v1/tickets (admin/organiser only)
func (tc *TicketsController) GetAllTickets(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tickets, err := tc.dbService.Client.Ticket.FindMany().OrderBy(
		db.Ticket.CreatedAt.Order(db.DESC),
	).Exec(ctx)

	if err != nil {
		log.WithError(err).Error("Failed to fetch all tickets")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to fetch tickets",
		})
		return
	}

	response := make([]types.TicketResponse, len(tickets))
	for i, ticket := range tickets {
		response[i] = mapTicketToResponse(&ticket)
	}

	c.JSON(http.StatusOK, response)
}

// GetTicketByID handles GET /api/v1/tickets/:id
func (tc *TicketsController) GetTicketByID(c *gin.Context) {
	ticketID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	ticket, err := tc.dbService.Client.Ticket.FindUnique(
		db.Ticket.ID.Equals(ticketID),
	).Exec(ctx)

	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Error:   "not_found",
			Message: "Ticket not found",
		})
		return
	}

	// Check if ticket belongs to user
	if ticket.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, types.ErrorResponse{
			Error:   "forbidden",
			Message: "You don't have permission to view this ticket",
		})
		return
	}

	c.JSON(http.StatusOK, mapTicketToResponse(ticket))
}

// CancelTicket handles DELETE /api/v1/tickets/:id
func (tc *TicketsController) CancelTicket(c *gin.Context) {
	ticketID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Check if ticket exists and belongs to user
	ticket, err := tc.dbService.Client.Ticket.FindUnique(
		db.Ticket.ID.Equals(ticketID),
	).Exec(ctx)

	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Error:   "not_found",
			Message: "Ticket not found",
		})
		return
	}

	if ticket.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, types.ErrorResponse{
			Error:   "forbidden",
			Message: "You don't have permission to cancel this ticket",
		})
		return
	}

	if ticket.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "already_cancelled",
			Message: "Ticket is already cancelled",
		})
		return
	}

	// Update ticket status to cancelled
	updatedTicket, err := tc.dbService.Client.Ticket.FindUnique(
		db.Ticket.ID.Equals(ticketID),
	).Update(
		db.Ticket.Status.Set("cancelled"),
	).Exec(ctx)

	if err != nil {
		log.WithError(err).Error("Failed to cancel ticket")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to cancel ticket",
		})
		return
	}

	c.JSON(http.StatusOK, mapTicketToResponse(updatedTicket))
}

func mapTicketToResponse(ticket *db.TicketModel) types.TicketResponse {
	return types.TicketResponse{
		ID:         ticket.ID,
		UserID:     ticket.UserID,
		EventID:    ticket.EventID,
		Quantity:   ticket.Quantity,
		TotalPrice: ticket.TotalPrice,
		Status:     ticket.Status,
		CreatedAt:  ticket.CreatedAt,
		UpdatedAt:  ticket.UpdatedAt,
	}
}
