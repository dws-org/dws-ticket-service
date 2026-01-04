package rabbitmq

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/oskargbc/dws-ticket-service/configs"
	"github.com/oskargbc/dws-ticket-service/internal/types"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	rabbitInstance *RabbitMQService
	rabbitOnce     sync.Once
)

type RabbitMQService struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	config   *configs.RabbitMQConfig
	mu       sync.RWMutex
	isHealthy bool
}

func GetRabbitMQServiceInstance(cfg *configs.Config) (*RabbitMQService, error) {
	var err error
	rabbitOnce.Do(func() {
		conn, connErr := amqp.Dial(cfg.RabbitMQ.URL)
		if connErr != nil {
			err = fmt.Errorf("failed to connect to RabbitMQ: %w", connErr)
			log.WithError(connErr).Error("RabbitMQ connection failed")
			return
		}

		channel, chanErr := conn.Channel()
		if chanErr != nil {
			conn.Close()
			err = fmt.Errorf("failed to open RabbitMQ channel: %w", chanErr)
			log.WithError(chanErr).Error("RabbitMQ channel creation failed")
			return
		}

		// Declare exchange
		if exchErr := channel.ExchangeDeclare(
			cfg.RabbitMQ.Exchange,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		); exchErr != nil {
			channel.Close()
			conn.Close()
			err = fmt.Errorf("failed to declare exchange: %w", exchErr)
			log.WithError(exchErr).Error("RabbitMQ exchange declaration failed")
			return
		}

		// Declare queues
		if queueErr := declareQueues(channel, cfg); queueErr != nil {
			channel.Close()
			conn.Close()
			err = queueErr
			return
		}

		log.Info("Successfully connected to RabbitMQ")
		rabbitInstance = &RabbitMQService{
			conn:      conn,
			channel:   channel,
			config:    &cfg.RabbitMQ,
			isHealthy: true,
		}
	})

	if err != nil {
		return nil, err
	}

	return rabbitInstance, nil
}

func declareQueues(channel *amqp.Channel, cfg *configs.Config) error {
	queues := []string{
		cfg.RabbitMQ.Queue.Purchased,
		cfg.RabbitMQ.Queue.Confirmed,
	}

	for _, queueName := range queues {
		queue, err := channel.QueueDeclare(
			queueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}

		// Bind queue to exchange
		if err := channel.QueueBind(
			queue.Name,
			queueName,
			cfg.RabbitMQ.Exchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", queueName, err)
		}
	}

	return nil
}

func (r *RabbitMQService) PublishTicketPurchased(msg types.TicketMessage) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return fmt.Errorf("RabbitMQ channel is nil")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := r.channel.Publish(
		r.config.Exchange,
		r.config.Queue.Purchased,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	); err != nil {
		log.WithError(err).Error("Failed to publish message to RabbitMQ")
		r.isHealthy = false
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.WithFields(log.Fields{
		"ticket_id": msg.TicketID,
		"event_id":  msg.EventID,
		"user_id":   msg.UserID,
	}).Info("Published ticket purchased message")

	return nil
}

func (r *RabbitMQService) HealthCheck() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.isHealthy || r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is not healthy")
	}

	return nil
}

func (r *RabbitMQService) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			log.WithError(err).Warn("Failed to close RabbitMQ channel")
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close RabbitMQ connection")
		}
	}

	log.Info("Successfully closed RabbitMQ connection")
	return nil
}
