# AGP - Asynchronous Query Processor

AGP is a system for managing the asynchronous execution of SQL queries against a ClickHouse cluster. It efficiently handles query orchestration, execution, and result management, making it ideal for long-running analytical workloads.

## 🚀 Getting Started with Docker Compose

AGP relies on PostgreSQL for query orchestration and ClickHouse for execution. The provided Docker Compose file builds AGP and starts all required dependencies.

```sh
docker compose up -d
```

## 🔍 Key Concepts

### Execution
AGP orchestrates SQL query execution asynchronously, ensuring efficient resource utilization.

- **Execution**: A container for an SQL query scheduled for execution.
- **Queueing**: Executions are created via the API and stored in PostgreSQL.
- **Workers**: A fleet of workers processes executions against ClickHouse.
- **Tracking**: The API provides status updates (**PENDING, RUNNING, CANCELED, FAILED, SUCCEEDED**), query progress, and result availability.
- **Result Storage**: Successfully completed executions are stored in an object store for retrieval until expiration.

### Execution Collapsing
To prevent redundant execution of identical queries, AGP supports query deduplication:

- A **query_id** can be assigned at execution creation.
- At any given time, only **one in-flight Execution** (**PENDING** or **RUNNING**) exists per `query_id`.
- If no `query_id` is specified, it defaults to a hash of the SQL query.

### Result Storage
AGP supports long-running analytical queries where some degree of data staleness is acceptable:

- Execution results are stored in an **object store** or **local filesystem**.
- The API allows listing past executions for a given `query_id` and retrieving their results.
- Users can flexibly choose to use recent results instead of re-executing queries.

### Tier-based Resource Allocation
AGP dynamically manages execution priority based on user typology:

- **Customizable Tiers**: Executions are assigned a tier at creation (a string value).
- **Resource Allocation**: Higher-tier users (e.g., experienced analysts) may access more CPU and longer query times, while lower-tier users have more restricted execution limits.
- **Flexible Worker Configurations**: Different workers handle different tiers and can be configured with distinct ClickHouse settings or separate clusters.

## ⚙️ System Architecture

### API Server
The API server allows users to:
- Create executions
- List executions
- Poll execution statuses
- Retrieve results

### Worker
Workers are responsible for:
- Polling PostgreSQL for new executions
- Running queries against the ClickHouse cluster
- Updating execution status and storing results

### Bookkeeper
The Bookkeeper maintains system stability by:
- Detecting and recovering from dead workers to prevent stuck executions
- Expiring old executions and their results

## 📘 API Documentation

AGP's API is built with **OpenAPI**, and the documentation can be viewed using:

```sh
go run github.com/swaggest/swgui/cmd/swgui internal/api/v1/async/api.yaml
```

Alternatively, the API server exposes an embedded Swagger UI at:
[http://localhost:8888/v1/async/docs/](http://localhost:8888/v1/async/docs/)

---
