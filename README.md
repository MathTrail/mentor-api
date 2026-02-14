# mathtrail-mentor
AI strategist that analyzes student profiles and determines optimal learning paths — a stateless analyst that instructs other services.

## Mission & Responsibilities
- Analyze student profile data to identify skill gaps
- Generate Learning Strategy (topic weights, difficulty curve)
- Respond to student requests ("more logic", "harder problems")
- Pass generation parameters to Task Service (type, difficulty, topic)
- Stateless: reads from Profile, writes instructions to Task

## Tech Stack
- **Language**: Go 1.25.7
- **Framework**: stdlib net/http
- **AI**: Rule-based engine (v1), LLM integration (v2)
- **Events**: Dapr pub/sub over Kafka

## Architecture
Mentor is a **stateless analyst** — it does NOT store data. It reads from Profile Service and publishes instructions for Task Service:
```
Profile → [data] → Mentor → [strategy] → Task
```

## API & Communication (Dapr)
- **Inbound**: REST API (POST /strategy/generate), Dapr service invocation from UI
- **Outbound**: Dapr invoke → mathtrail-profile (GET profile data)
- **Publishes**: `mentor.strategy.updated` (parameters for Task Service)
- **Subscribes**: `task.attempt.completed` (trigger re-evaluation)

## Data Persistence
- **None** — stateless service. All state lives in Profile Service.

## Secrets
- None required (reads only public profile data via Dapr)

## Infrastructure
Standard `infra/` layout:
- `infra/helm/` — Helm chart + environment overlays (dev, on-prem, cloud)
- `infra/terraform/` — Environment scaffolds (stateless — no DB)
- `infra/ansible/` — On-prem node preparation

## Development
- `just dev` — Skaffold dev loop
- `just test` — Go unit tests

## Roadmap
1. Implement rule-based Learning Strategy engine (topic weights + difficulty mapping)
2. Build Dapr service invocation client for Profile reads
3. Add pub/sub publisher for `mentor.strategy.updated` events
