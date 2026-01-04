# dws-ticket-service

Microservice for event ticket purchases with RabbitMQ integration.

## Architecture

This service is part of the DWS (Distributed Web Services) event platform:
- **dws-event-service**: Manages events and organizers
- **dws-ticket-service**: Handles ticket purchases and confirmations (this service)
- **dws-frontend**: Next.js frontend application
- **RabbitMQ**: Message broker for async processing
- **Keycloak**: OAuth2 authentication
- **PostgreSQL**: Database per service

## Features

- ✅ Ticket purchase API with Keycloak authentication
- ✅ RabbitMQ integration for async ticket confirmation
- ✅ Personal ticket history per user
- ✅ Ticket cancellation/refund
- ✅ CI/CD with GitHub Actions
- ✅ Kubernetes deployment with Helm
- ✅ Comprehensive test coverage

## API Endpoints

### Tickets
- `POST /api/v1/tickets/purchase` - Purchase tickets for an event
- `GET /api/v1/tickets/my-tickets` - Get current user's tickets
- `GET /api/v1/tickets/:id` - Get ticket details
- `DELETE /api/v1/tickets/:id` - Cancel ticket (refund)

### Health
- `GET /api/v1/health` - Health check
- `GET /api/v1/health/db` - Database connection status
- `GET /api/v1/health/rabbitmq` - RabbitMQ connection status

## Technology Stack

- **Go 1.23.1** with Gin web framework
- **Prisma Client Go** for type-safe database access
- **PostgreSQL 16** for data persistence
- **RabbitMQ** for message queuing
- **Keycloak** for JWT authentication
- **Docker** for containerization
- **Kubernetes** for orchestration
- **ArgoCD** for GitOps deployment

## Development

### Prerequisites
- Go 1.23.1+
- Docker & Docker Compose
- PostgreSQL 16
- RabbitMQ

### Local Setup

```bash
# Install dependencies
go mod download

# Generate Prisma client
go run github.com/steebchen/prisma-client-go generate

# Run database migrations
go run github.com/steebchen/prisma-client-go db push

# Run tests
go test ./... -v -cover

# Run service
go run cmd/server/main.go
```

### Environment Variables

Create a `.env` file:
```env
DATABASE_URL="postgresql://user:password@localhost:5432/tickets?schema=public"
RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
KEYCLOAK_URL="http://localhost:8080"
KEYCLOAK_REALM="dws"
```

## Database Schema

```prisma
model Ticket {
  id          String   @id @default(uuid())
  userId      String   // From Keycloak JWT
  eventId     String   // Reference to event in event-service
  quantity    Int
  totalPrice  Float
  status      String   // pending, confirmed, cancelled
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt
}
```

## RabbitMQ Message Flow

1. **Purchase Request** → Create ticket (status: pending)
2. **Publish** to `ticket.purchased` queue
3. **Consumer** processes message:
   - Validate payment
   - Update status to `confirmed`
   - Send confirmation email
4. **Error handling** with retry logic

## Testing

Tests include:
- Unit tests for controllers
- Integration tests with PostgreSQL
- RabbitMQ message flow tests
- Auth middleware tests
- Minimum 50% code coverage (CI enforced)

## Deployment

Deployed via ArgoCD GitOps:
- Helm chart in `dws-org/gitops` repository
- Namespace: `dws-ticket-service`
- Auto-sync enabled
- Health checks configured

## License

MIT
