package types

import "time"

// PurchaseRequest represents a ticket purchase request
type PurchaseRequest struct {
	EventID    string  `json:"event_id" binding:"required"`
	Quantity   int     `json:"quantity" binding:"required,min=1,max=10"`
	TotalPrice float64 `json:"total_price" binding:"required,min=0"`
}

// TicketResponse represents a ticket in API responses
type TicketResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	EventID    string    `json:"event_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TicketMessage represents a message published to RabbitMQ
type TicketMessage struct {
	TicketID   string    `json:"ticket_id"`
	UserID     string    `json:"user_id"`
	EventID    string    `json:"event_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Timestamp  time.Time `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Services map[string]string `json:"services,omitempty"`
	Status   string            `json:"status"`
}
