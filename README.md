# mentor-api

[![CI](https://github.com/MathTrail/mentor-api/actions/workflows/ci.yml/badge.svg)](https://github.com/MathTrail/mentor-api/actions)
[![Latest Release](https://img.shields.io/github/v/release/MathTrail/mentor-api?style=flat-square)](https://github.com/MathTrail/mentor-api/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MathTrail/mentor-api)](https://goreportcard.com/report/github.com/MathTrail/mentor-api)
[![codecov](https://codecov.io/gh/MathTrail/mentor-api/branch/main/graph/badge.svg)](https://codecov.io/gh/MathTrail/mentor-api)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MathTrail/mentor-api)](https://github.com/MathTrail/mentor-api/blob/main/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/MathTrail/mentor-api.svg)](https://pkg.go.dev/github.com/MathTrail/mentor-api)

[![SonarQube Cloud](https://sonarcloud.io/images/project_badges/sonarcloud-light.svg)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_mentor-api&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_mentor-api&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_mentor-api&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_mentor-api&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_mentor-api&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_mentor-api&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=MathTrail_mentor-api)

Mentor API is the intelligence hub of the MathTrail platform, responsible for adapting the learning experience to each individual student. The service analyses feedback, tracks progress, and generates personalised learning recommendations.

## Business Capabilities

- **Feedback Analysis** — Interprets student feedback on task difficulty using an LLM.
- **Learning Roadmaps** — Generates adaptive learning paths based on each student's current progress.
- **Strategy Orchestration** — Determines the optimal teaching strategy to adjust content difficulty.

## System Architecture

[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Debezium](https://img.shields.io/badge/Debezium-FF6A00?style=for-the-badge&logo=redhat&logoColor=white)](https://debezium.io/)
[![Apache Kafka](https://img.shields.io/badge/Kafka-000000?style=for-the-badge&logo=apachekafka&logoColor=white)](https://kafka.apache.org/)
[![Apache Flink](https://img.shields.io/badge/Flink-E6526F?style=for-the-badge&logo=apacheflink&logoColor=white)](https://flink.apache.org/)

[![Architecture: EDA](https://img.shields.io/badge/Architecture-Event--Driven-8A2BE2?style=for-the-badge&logo=eventstore)](https://aws.amazon.com/event-driven-architecture/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white)](./infra/helm/mentor-api)
[![Vault](https://img.shields.io/badge/Vault-FF3E00?style=for-the-badge&logo=hashicorpvault&logoColor=white)](https://www.vaultproject.io/)

[![API Docs](https://img.shields.io/badge/API_Docs-Swagger-85EA2D?style=for-the-badge&logo=swagger&logoColor=black)](https://MathTrail.github.io/mentor-api/)
[![Tracing](https://img.shields.io/badge/Tracing-OTel-000000?style=for-the-badge&logo=opentelemetry&logoColor=white)](https://opentelemetry.io/)
[![Profiling](https://img.shields.io/badge/Profiling-Pyroscope-FF7800?style=for-the-badge&logo=pyroscope&logoColor=white)](https://pyroscope.io/)

```mermaid
graph LR
    User([Student UI]) -- "Auth" --> OK[Oathkeeper]
    
    subgraph MentorService [Mentor API Platform]
        direction LR
        App["Mentor API"]
    end

    OK -- "X-User-ID" --> App

    subgraph Storage [Data Layer]
        direction TB
        PGB["PgBouncer"] --> PG[("Postgres")]
        Mig["Migration Job"] --> PG
    end

    App -- "SQL" --> PGB
    PG -- "CDC" --> Deb[Debezium]
    
    subgraph Bus [Event Bus]
        Kfk{Kafka}
    end

    Deb -- "feedback.created" --> Kfk
    Kfk -- "progress / profile" --> App
    App -- "strategy / roadmap" --> Kfk

    subgraph Support [Infra Support]
        direction TB
        Vault["Vault"] --> ESO["ESO"]
        Obs["Observability"]
    end

    ESO -- "Secrets" --> App
    App -- "Telemetry" --> Obs

    %% Styling
    classDef svc fill:#5b21b6,stroke:#7c3aed,color:#fff
    classDef authCls fill:#b45309,stroke:#f59e0b,color:#fff
    classDef dataCls fill:#1e3a5f,stroke:#3b82f6,color:#fff
    classDef cdcCls fill:#166534,stroke:#22c55e,color:#fff
    classDef eventCls fill:#1c1917,stroke:#78716c,color:#fff
    classDef secretCls fill:#7f1d1d,stroke:#ef4444,color:#fff
    classDef obsCls fill:#134e4a,stroke:#2dd4bf,color:#fff
    classDef actorCls fill:#1e1b4b,stroke:#818cf8,color:#fff

    class App svc; class OK authCls;
    class PGB,PG,Mig dataCls; class Deb cdcCls; class Kfk eventCls;
    class Vault,ESO secretCls; class Obs obsCls; class User actorCls;
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
just tp-intercept
go run ./cmd/server/main.go

just tp-stop
```

## Releases

```bash
git tag -a v0.2.0 -m "Release description"
git push origin v0.2.0
```

GitHub Actions will build binaries, generate a Changelog, and publish a GitHub Release.
