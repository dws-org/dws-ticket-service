# Ticket Service API Documentation

## Overview

The Ticket Service handles ticket purchases and management in the DWS platform. It provides:
- Ticket purchasing with async processing via RabbitMQ
- User ticket management (view own tickets)
- Ticket cancellation
- Integration with Event Service for event validation

## Base URLs

- **Production**: `https://ticket.ltu-m7011e-6.se`
- **Local**: `http://localhost:8081`

## Authentication

All endpoints require Bearer token authentication from Keycloak.

### Getting a Token

1. Login via frontend: `https://frontend.ltu-m7011e-6.se`
2. Or use Keycloak directly: `https://keycloak.ltu-m7011e-6.se`
3. Extract JWT token from browser DevTools or authentication response

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://ticket.ltu-m7011e-6.se/api/v1/tickets/my-tickets
```

## Ticket Purchase Workflow

1. **User submits purchase request** → Ticket created with `pending` status
2. **Message published to RabbitMQ** → Async processing queue
3. **Consumer service processes** → Updates ticket to `confirmed`
4. **User can view ticket** → Status shows as confirmed

## Endpoints

### POST /api/v1/tickets/purchase

Purchase tickets for an event.

**Authentication**: Required  
**Authorization**: All authenticated users

**Request Body**:
```json
{
  "event_id": "evt-001",
  "quantity": 2,
  "total_price": 1198.00
}
```

**Response**: `201 Created`
```json
{
  "id": "ticket-abc123",
  "user_id": "user-123",
  "event_id": "evt-001",
  "quantity": 2,
  "total_price": 1198.00,
  "status": "pending",
  "created_at": "2026-01-07T20:00:00Z",
  "updated_at": "2026-01-07T20:00:00Z"
}
```

**Validation**:
- `event_id`: Required, must be valid event
- `quantity`: Required, min: 1, max: 10
- `total_price`: Required, min: 0

**Error Responses**:
- `400 Bad Request` - Invalid input (e.g., quantity < 1)
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Database or RabbitMQ error

### GET /api/v1/tickets/my-tickets

Get all tickets for the authenticated user.

**Authentication**: Required  
**Authorization**: Users can only see their own tickets

**Response**: `200 OK`
```json
[
  {
    "id": "ticket-abc123",
    "user_id": "user-123",
    "event_id": "evt-001",
    "quantity": 2,
    "total_price": 1198.00,
    "status": "confirmed",
    "created_at": "2026-01-07T20:00:00Z",
    "updated_at": "2026-01-07T20:05:00Z"
  },
  {
    "id": "ticket-def456",
    "user_id": "user-123",
    "event_id": "evt-002",
    "quantity": 1,
    "total_price": 299.00,
    "status": "pending",
    "created_at": "2026-01-06T18:00:00Z",
    "updated_at": "2026-01-06T18:00:00Z"
  }
]
```

**Notes**:
- Results ordered by `created_at DESC` (newest first)
- Empty array if user has no tickets

### GET /api/v1/tickets/{id}

Get a specific ticket by ID.

**Authentication**: Required  
**Authorization**: User must own the ticket

**Parameters**:
- `id` (path) - Ticket ID

**Response**: `200 OK`
```json
{
  "id": "ticket-abc123",
  "user_id": "user-123",
  "event_id": "evt-001",
  "quantity": 2,
  "total_price": 1198.00,
  "status": "confirmed",
  "created_at": "2026-01-07T20:00:00Z",
  "updated_at": "2026-01-07T20:05:00Z"
}
```

**Error Responses**:
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Ticket belongs to different user
- `404 Not Found` - Ticket does not exist

### DELETE /api/v1/tickets/{id}

Cancel a ticket.

**Authentication**: Required  
**Authorization**: User must own the ticket

**Parameters**:
- `id` (path) - Ticket ID

**Response**: `200 OK`
```json
{
  "id": "ticket-abc123",
  "user_id": "user-123",
  "event_id": "evt-001",
  "quantity": 2,
  "total_price": 1198.00,
  "status": "cancelled",
  "created_at": "2026-01-07T20:00:00Z",
  "updated_at": "2026-01-07T20:10:00Z"
}
```

**Error Responses**:
- `400 Bad Request` - Ticket already cancelled
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Ticket belongs to different user
- `404 Not Found` - Ticket does not exist

## Ticket Status

| Status | Description |
|--------|-------------|
| `pending` | Ticket created, awaiting confirmation |
| `confirmed` | Ticket confirmed by consumer service |
| `cancelled` | Ticket cancelled by user |

## Health & Monitoring

### GET /livez

Kubernetes liveness probe.

**Response**: `200 OK`

### GET /readyz

Kubernetes readiness probe. Checks database and RabbitMQ connections.

**Response**: `200 OK` / `503 Service Unavailable`

### GET /metrics

Prometheus metrics endpoint.

**Response**: `200 OK` (text/plain)
```
# HELP tickets_purchased_total Total number of tickets purchased
# TYPE tickets_purchased_total counter
tickets_purchased_total 42

# HELP tickets_confirmed_total Total number of tickets confirmed  
# TYPE tickets_confirmed_total counter
tickets_confirmed_total 38
```

## Error Handling

All errors follow this format:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

Common error codes:
- `unauthorized` - Authentication required
- `forbidden` - Access denied
- `not_found` - Resource not found
- `invalid_request` - Bad request payload
- `already_cancelled` - Ticket already cancelled
- `database_error` - Database operation failed
- `messaging_error` - RabbitMQ operation failed

## Database Schema

```sql
CREATE TABLE tickets (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL,
  event_id    TEXT NOT NULL,
  quantity    INTEGER NOT NULL,
  total_price DECIMAL NOT NULL,
  status      TEXT NOT NULL,  -- pending | confirmed | cancelled
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tickets_user_id ON tickets(user_id);
CREATE INDEX idx_tickets_status ON tickets(status);
```

## RabbitMQ Integration

**Exchange**: `ticket_events`  
**Queue**: `ticket_purchases`  
**Routing Key**: `ticket.purchased`

**Message Format**:
```json
{
  "ticket_id": "ticket-abc123",
  "user_id": "user-123",
  "event_id": "evt-001",
  "quantity": 2,
  "total_price": 1198.00,
  "timestamp": "2026-01-07T20:00:00Z"
}
```

## Testing

### Manual Testing

```bash
# 1. Get auth token from browser
TOKEN="your_jwt_token_here"

# 2. Purchase tickets
curl -X POST https://ticket.ltu-m7011e-6.se/api/v1/tickets/purchase \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-001",
    "quantity": 2,
    "total_price": 1198.00
  }'

# 3. View your tickets
curl https://ticket.ltu-m7011e-6.se/api/v1/tickets/my-tickets \
  -H "Authorization: Bearer $TOKEN"

# 4. Cancel a ticket
curl -X DELETE https://ticket.ltu-m7011e-6.se/api/v1/tickets/{ticket_id} \
  -H "Authorization: Bearer $TOKEN"
```

## Support

- GitHub: https://github.com/dws-org/dws-ticket-service
- OpenAPI Spec: `docs/openapi.yaml`
- README: `README.md`
