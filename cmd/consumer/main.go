package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oskargbc/dws-ticket-service/configs"
	"github.com/oskargbc/dws-ticket-service/internal/pkg/rabbitmq"
	"github.com/oskargbc/dws-ticket-service/internal/services"
	"github.com/oskargbc/dws-ticket-service/internal/types"
	"github.com/oskargbc/dws-ticket-service/prisma/db"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
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

	log.Info("Starting dws-ticket-service consumer")

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

	// Start consuming messages
	if err := consumeTicketMessages(cfg, dbService, rmqService); err != nil {
		log.WithError(err).Fatal("Failed to start consumer")
	}
}

func consumeTicketMessages(cfg *configs.Config, dbService *services.DatabaseService, rmqService *rabbitmq.RabbitMQService) error {
	// Get channel from RabbitMQ service
	msgs, err := rmqService.ConsumeTicketPurchased()
	if err != nil {
		return err
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Consumer started, waiting for messages...")

	// Process messages
	go func() {
		for msg := range msgs {
			if err := processTicketMessage(msg, dbService); err != nil {
				log.WithError(err).Error("Failed to process message")
				msg.Nack(false, true) // Requeue on error
			} else {
				msg.Ack(false) // Acknowledge successful processing
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("Shutting down consumer...")
	return nil
}

func processTicketMessage(msg amqp.Delivery, dbService *services.DatabaseService) error {
	// Parse message
	var ticketMsg types.TicketMessage
	if err := json.Unmarshal(msg.Body, &ticketMsg); err != nil {
		log.WithError(err).Error("Failed to unmarshal message")
		return err
	}

	log.WithFields(log.Fields{
		"ticket_id": ticketMsg.TicketID,
		"user_id":   ticketMsg.UserID,
		"event_id":  ticketMsg.EventID,
		"quantity":  ticketMsg.Quantity,
	}).Info("Processing ticket purchase message")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current ticket
	ticket, err := dbService.Client.Ticket.FindUnique(
		db.Ticket.ID.Equals(ticketMsg.TicketID),
	).Exec(ctx)

	if err != nil {
		log.WithError(err).Error("Failed to find ticket")
		return err
	}

	// If already confirmed, skip
	if ticket.Status == "confirmed" {
		log.WithField("ticket_id", ticketMsg.TicketID).Info("Ticket already confirmed, skipping")
		return nil
	}

	// Simulate payment processing (in real app: call payment gateway)
	time.Sleep(500 * time.Millisecond)

	// Update ticket status to confirmed
	updatedTicket, err := dbService.Client.Ticket.FindUnique(
		db.Ticket.ID.Equals(ticketMsg.TicketID),
	).Update(
		db.Ticket.Status.Set("confirmed"),
	).Exec(ctx)

	if err != nil {
		log.WithError(err).Error("Failed to update ticket status")
		return err
	}

	log.WithFields(log.Fields{
		"ticket_id": updatedTicket.ID,
		"status":    updatedTicket.Status,
	}).Info("Ticket confirmed successfully")

	// Send confirmation email (mock)
	sendConfirmationEmail(ticketMsg)

	return nil
}

func sendConfirmationEmail(ticketMsg types.TicketMessage) {
	// Mock email sending
	log.WithFields(log.Fields{
		"ticket_id": ticketMsg.TicketID,
		"user_id":   ticketMsg.UserID,
		"event_id":  ticketMsg.EventID,
	}).Info("📧 Sending confirmation email (mock)")

	// In production: integrate with SendGrid, AWS SES, or similar
}
