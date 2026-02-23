# mentor-api

[![CI](https://github.com/MathTrail/mentor-api/actions/workflows/ci.yml/badge.svg)](https://github.com/MathTrail/mentor-api/actions)
[![Latest Release](https://img.shields.io/github/v/release/MathTrail/mentor-api?style=flat-square)](https://github.com/MathTrail/mentor-api/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MathTrail/mentor-api)](https://goreportcard.com/report/github.com/MathTrail/mentor-api)
[![codecov](https://codecov.io/gh/MathTrail/mentor-api/branch/main/graph/badge.svg)](https://codecov.io/gh/MathTrail/mentor-api)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MathTrail/mentor-api)](https://github.com/MathTrail/mentor-api/blob/main/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/MathTrail/mentor-api.svg)](https://pkg.go.dev/github.com/MathTrail/mentor-api)

[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Debezium](https://img.shields.io/badge/Debezium-FF6A00?style=for-the-badge&logo=redhat&logoColor=white)](https://debezium.io/)
[![Apache Kafka](https://img.shields.io/badge/Kafka-000000?style=for-the-badge&logo=apachekafka&logoColor=white)](https://kafka.apache.org/)
[![Apache Flink](https://img.shields.io/badge/Flink-E6526F?style=for-the-badge&logo=apacheflink&logoColor=white)](https://flink.apache.org/)

[![Architecture: EDA](https://img.shields.io/badge/Architecture-Event--Driven-8A2BE2?style=for-the-badge&logo=eventstore)](https://aws.amazon.com/event-driven-architecture/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white)](./infra/helm/mentor-api)
[![Vault](https://img.shields.io/badge/Vault-FF3E00?style=for-the-badge&logo=hashicorpvault&logoColor=white)](https://www.vaultproject.io/)
[![Dapr](https://img.shields.io/badge/Dapr-007ACC?style=for-the-badge&logo=dapr&logoColor=white)](https://dapr.io/)

[![API Docs](https://img.shields.io/badge/API_Docs-Swagger-85EA2D?style=for-the-badge&logo=swagger&logoColor=black)](https://MathTrail.github.io/mentor-api/)
[![Tracing](https://img.shields.io/badge/Tracing-OTel-000000?style=for-the-badge&logo=opentelemetry&logoColor=white)](https://opentelemetry.io/)
[![Profiling](https://img.shields.io/badge/Profiling-Pyroscope-FF7800?style=for-the-badge&logo=pyroscope&logoColor=white)](https://pyroscope.io/)

---

Mentor API is the intelligence hub of the MathTrail platform, responsible for adapting the learning experience to each individual student. The service analyses feedback, tracks progress, and generates personalised learning recommendations.

## Business Capabilities

- **Feedback Analysis** — Interprets student feedback on task difficulty using an LLM.
- **Learning Roadmaps** — Generates adaptive learning paths based on each student's current progress.
- **Strategy Orchestration** — Determines the optimal teaching strategy to adjust content difficulty.

## System Architecture

```mermaid
graph TD
    User([Student UI]) -- Feedback --> Mentor[Mentor API]
    Mentor -- Analysis --> LLM[LLM Provider]
    Mentor -- "Sync Data" --> DB[(Database)]
    DB -- "Change Data Capture" --> Debezium[Debezium]
    Debezium -- "Events" --> Kafka{Kafka Bus}

    Validator[Solution Validator] -- "Progress Events" --> Kafka
    Kafka -- "Consume Progress" --> Mentor

    Mentor -- "Get Profile" --> Profile[Profile API]
```

## Development

All commands are run via `just`.

```bash
just deploy
just k6-load
```

## Debug

[Telepresence](https://www.telepresence.io/) intercepts live cluster traffic and routes it to your local process, so you can debug against real dependencies without deploying.

```bash
just tp-intercept   # deploy → connect to cluster → start intercept on port 8080
go run ./cmd/server/main.go

just tp-stop        # leave intercept and disconnect
```

## Releases

```bash
git tag -a v0.2.0 -m "Release description"
git push origin v0.2.0
```

GitHub Actions will build binaries, generate a Changelog, and publish a GitHub Release.
