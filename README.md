# mentor-api

Student Feedback Loop service for the MathTrail platform. Receives student feedback about task difficulty, delegates analysis to an LLM, and stores the resulting strategy in PostgreSQL.

## Mission & Responsibilities

- **Receive student feedback** about task difficulty
- **Analyse feedback via LLM** (mock stub now, real integration planned)
- **Store feedback history** in PostgreSQL with JSONB strategy snapshots
- **Event publishing** via Debezium CDC (monitoring feedback table)

## Architecture

```
Student → POST /v1/feedback → FeedbackService → LLMClient (stub)
                               ↓
                           PostgreSQL (feedback table with strategy_snapshot JSONB)
                               ↓
                           Debezium CDC → Kafka → mentor.strategy.updated event
```

**Key Design Decisions:**
- **No Dapr publisher** - Events are published by Debezium CDC monitoring the PostgreSQL feedback table
- **JSONB for strategy_snapshot** - Flexible schema for storing strategy state at the time of feedback
- **PostgreSQL ENUM** - difficulty_level ('easy', 'ok', 'hard')
- **LLM-first** - All analysis delegated to LLM; currently a mock returning neutral strategy

## Tech Stack

- **Language**: Go 1.25.7
- **Framework**: Gin (HTTP), pgx (PostgreSQL driver)
- **Database**: PostgreSQL with JSONB for strategy snapshots
- **Events**: Debezium CDC (handled externally)
- **Testing**: Go testing + testify, Grafana k6
- **Infrastructure**: Docker, Helm (mathtrail-service-lib), Skaffold

## API Endpoints

### POST /api/v1/feedback
Submit student feedback about task difficulty.

**Request:**
```json
{
  "student_id": "550e8400-e29b-41d4-a716-446655440000",
  "task_id": "task-123",
  "message": "This is too hard"
}
```

**Response:**
```json
{
  "student_id": "550e8400-e29b-41d4-a716-446655440000",
  "task_id": "task-123",
  "difficulty_adjustment": 0.0,
  "topic_weights": {"general": 1.0},
  "sentiment": "neutral",
  "strategy_snapshot": {
    "difficulty_weight": 1.0,
    "feedback_based": true,
    "sentiment": "neutral"
  },
  "timestamp": "2026-02-16T10:12:45Z"
}
```

### GET /health/startup, /health/liveness, /health/ready
Kubernetes health probes.

## Database Schema

```sql
CREATE TYPE difficulty_level AS ENUM ('easy', 'ok', 'hard');

CREATE TABLE feedback (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id           UUID NOT NULL,
    message              TEXT,
    perceived_difficulty difficulty_level NOT NULL,
    strategy_snapshot    JSONB NOT NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_student_id ON feedback(student_id);
CREATE INDEX idx_feedback_created_at ON feedback(created_at DESC);
CREATE INDEX idx_feedback_strategy_snapshot ON feedback USING GIN (strategy_snapshot);
```

## Development

### Prerequisites
- k3d cluster running (see `infra-local-k3s` repo)
- DevContainer (VS Code) or Go 1.25.7 + dependencies installed locally

### Quick Start

```bash
# Start development mode (hot-reload + port-forward)
just dev

# Build binary
just build

# Run tests
just test

# Run k6 load test
just k6-load

# Generate Swagger docs
just swagger
```

### DevContainer
The repository includes a fully configured DevContainer with Go, Docker, kubectl, Helm, Skaffold, Telepresence, and all necessary tools.

```bash
# Open in VS Code
code .
# Command Palette → "Dev Containers: Reopen in Container"
```

## Deployment

### Local k3d Cluster

```bash
# Deploy to local k3d
just deploy

# View logs
just logs

# Check status
just status

# Access locally
just test-endpoints
```

### Production
Helm chart is available at `infra/helm/mentor-api` with environment-specific values overlays.

## Event Publishing (Debezium CDC)

This service does NOT publish events directly. Instead, Debezium monitors the `feedback` table and automatically publishes change events to Kafka:

- **Table:** `feedback`
- **Topic:** `mentor.feedback`
- **Event Type:** `mentor.strategy.updated`
- **Payload:** Full feedback row including JSONB strategy_snapshot

**Why Debezium?**
- Guaranteed delivery (no missed events)
- Transactional consistency (event published only if DB commit succeeds)
- No application code needed for event publishing
- Automatic schema evolution support

## Testing

### Unit Tests
```bash
just test
```

### Load Tests
```bash
# Local
BASE_URL=http://localhost:8080 just k6-load

# In cluster
just load-test
```

### CI/CD
GitHub Actions workflow runs on every PR:
- Lint (golangci-lint)
- Unit tests
- k6 load test in ephemeral namespace
- Automatic cleanup

## Future Enhancements

1. **Real LLM Integration**: Connect mock LLM client to OpenAI/Claude API
2. **Topic Extraction**: LLM identifies specific math topics (algebra, geometry) from feedback
3. **Real-time Dashboards**: Grafana dashboards for feedback analytics
4. **Feedback Aggregation**: Weekly/monthly reports on difficulty trends

## References

- **Architecture Docs**: `../core/docs/architecture/feedback-loop.md`
- **Library Chart**: `mathtrail-charts/charts/mathtrail-service-lib`
- **Profile Service**: `../profile-api` (similar patterns)
- **Debezium Docs**: https://debezium.io/documentation/
