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
    User([Student UI])
    Validator([Solution Validator])

    subgraph auth [" Auth "]
        OK[Oathkeeper]
    end

    subgraph app [" Mentor API "]
        HTTP["Go Service · Gin\n:8080"]
        Sidecar["Dapr Sidecar\n:3500"]
    end

    subgraph external [" External Services "]
        LLM[LLM Provider]
        ProfileSvc[Profile API]
    end

    subgraph data [" Data "]
        PGB["PgBouncer\n:6432"]
        PG[("PostgreSQL\n:5432")]
        MigJob["Migration Job"]
    end

    subgraph cdc [" CDC · Events "]
        Deb[Debezium]
        Kfk{"Kafka"}
    end

    subgraph secrets [" Secrets "]
        Vault["HashiCorp Vault"]
        ESO["External Secrets Operator"]
    end

    Obs["Observability\ntraces · logs · metrics · profiling"]

    User --> OK
    OK -->|"X-User-ID header"| HTTP
    HTTP -->|"invoke binding"| Sidecar
    Sidecar -->|"SQL · :6432"| PGB
    PGB --> PG
    MigJob -->|"DDL · :5432 direct"| PG

    PG -->|"CDC (feedback table)"| Deb
    Deb -->|events| Kfk
    Validator -->|"progress events"| Kfk
    Kfk -->|"consume"| HTTP

    HTTP -->|"analyse feedback"| LLM
    HTTP -->|"get profile"| ProfileSvc

    Vault -->|"dynamic lease"| ESO
    ESO -->|"K8s Secret → conn string"| Sidecar

    HTTP -->|"OTel · Pyroscope"| Obs

    classDef svc fill:#5b21b6,stroke:#7c3aed,color:#fff
    classDef dapr fill:#0369a1,stroke:#38bdf8,color:#fff
    classDef authCls fill:#b45309,stroke:#f59e0b,color:#fff
    classDef dataCls fill:#1e3a5f,stroke:#3b82f6,color:#fff
    classDef cdcCls fill:#166534,stroke:#22c55e,color:#fff
    classDef eventCls fill:#1c1917,stroke:#78716c,color:#fff
    classDef secretCls fill:#7f1d1d,stroke:#ef4444,color:#fff
    classDef obsCls fill:#134e4a,stroke:#2dd4bf,color:#fff
    classDef extCls fill:#713f12,stroke:#f97316,color:#fff
    classDef actorCls fill:#1e1b4b,stroke:#818cf8,color:#fff

    class HTTP svc
    class Sidecar dapr
    class OK authCls
    class PGB,PG,MigJob dataCls
    class Deb cdcCls
    class Kfk eventCls
    class Vault,ESO secretCls
    class Obs obsCls
    class LLM,ProfileSvc extCls
    class User,Validator actorCls
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
