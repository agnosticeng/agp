AGP is a system that manages the asynchronous execution of SQL queries against a ClickHouse cluster.

## Running with Docker Compose

AGP relies on Postgres for query orchestration, and Clickhouse for query execution.
The Docker Compose file will build AGP and run all the dependencies.

```sh
docker compose up -d
```

## Concepts

### Execution

The primary role of AGP is to orchestrate the asynchronous execution of SQL queries and manage their results.

An Execution serves as a container for an SQL query that is scheduled for future execution. These executions are created via the API and placed in a queue within a PostgreSQL table.

A fleet of Workers continuously polls for new pending executions and processes them against a ClickHouse cluster.

The API enables tracking of execution state changes, including key statuses (PENDING, RUNNING, CANCELED, FAILED, SUCCEEDED), query progress metrics (such as the number of rows scanned and bytes read), and the availability of results.

When an Execution completed successfully, it's result is stored in an Object Store and can be retrieve until it is expired.

### Execution collapsing

A query_id can be assigned when an Execution is created to prevent redundant queuing of the same query, thereby optimizing resource usage.

At any given time, only one in-flight Execution (i.e., with a status of PENDING or RUNNING) can exist for a specific query_id.

If no query_id is provided at creation, it defaults to the hash of the SQL query.

### Result storage

Successful execution results are stored in an object store or local filesystem for future retrieval.

AGP is designed to support long-running analytical queries, where some degree of data staleness is acceptable.

The API enables listing existing executions for a given query_id and retrieving their results, allowing for a flexible approach where users can access relatively recent results if they meet their needs.

### Tier

The system is designed to allocate resources based on user typology, ensuring different user groups have varying levels of access to underlying resources.

For example, an experienced analyst may be granted higher CPU allocation and the ability to run longer queries on the ClickHouse cluster, while a junior analyst might be restricted to shorter queries with minimal CPU usage.

To support this flexibility, each Execution is assigned a tier (a customizable string value) at creation. Different worker processes handle different tiers, allowing for distinct ClickHouse configurations, including varied settings or even separate clusters.

## Architecture

### Server

The API server enables users to interact with the system by creating Executions, listing them, polling their status, and retrieving results.

### Worker

The worker is responsible for polling PostgreSQL for new available Executions and executing them against the ClickHouse cluster.

It also updates the execution status throughout the process and stores the results in the object store.

Multiple workers can operate simultaneously, each assigned to different tiers as needed.

### Bookkeeper

The Bookkeeper is a process responsible for running various control loops to maintain the system's stability and efficiency.

Its tasks include:

- Ensuring that executions do not remain stuck in a RUNNING status due to dead workers.
- Expiring old executions and their associated results.

## API documentation

The API is built with OpenAPI and it's documentation can be viewed by running the following command:

```sh
go run github.com/swaggest/swgui/cmd/swgui internal/api/v1/async/api.yaml
```

The server also expose an embedded Swagger UI at [http://localhost:8888/v1/async/docs/](http://localhost:8888/v1/async/docs/).