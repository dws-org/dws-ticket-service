package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPurchaseRequest(t *testing.T) {
	req := PurchaseRequest{
		EventID:    "123e4567-e89b-12d3-a456-426614174000",
		Quantity:   2,
		TotalPrice: 49.99,
	}

	assert.NotEmpty(t, req.EventID)
	assert.Equal(t, 2, req.Quantity)
	assert.Equal(t, 49.99, req.TotalPrice)
}

func TestTicketResponse(t *testing.T) {
	now := time.Now()
	resp := TicketResponse{
		ID:         "ticket-123",
		UserID:     "user-456",
		EventID:    "event-789",
		Quantity:   3,
		TotalPrice: 75.00,
		Status:     "confirmed",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	assert.Equal(t, "ticket-123", resp.ID)
	assert.Equal(t, "confirmed", resp.Status)
	assert.Equal(t, 3, resp.Quantity)
}

func TestTicketMessage(t *testing.T) {
	msg := TicketMessage{
		TicketID:   "ticket-123",
		UserID:     "user-456",
		EventID:    "event-789",
		Quantity:   1,
		TotalPrice: 25.00,
		Timestamp:  time.Now(),
	}

	assert.NotEmpty(t, msg.TicketID)
	assert.NotEmpty(t, msg.UserID)
	assert.NotEmpty(t, msg.EventID)
	assert.Greater(t, msg.Quantity, 0)
}

func TestErrorResponse(t *testing.T) {
	err := ErrorResponse{
		Error:   "validation_error",
		Message: "Invalid quantity",
	}

	assert.Equal(t, "validation_error", err.Error)
	assert.Equal(t, "Invalid quantity", err.Message)
}
