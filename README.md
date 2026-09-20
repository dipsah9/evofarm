# 🧬 EvoFarm

[![CI](https://github.com/dipsah9/evofarm/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/dipsah9/evofarm/actions/workflows/ci.yml)

# EvoFarm --- Distributed Hybrid Optimization

### Submit a problem. Route it to the right solver. Watch it run.

**[Live demo](https://evofarm.vercel.app/)** · **[API](https://evofarm-api.fly.dev/)**

EvoFarm is a hybrid optimization platform for running evolutionary and
constraint-based workloads through one distributed job interface. Users
submit a problem, the platform routes it to the appropriate solver, and a
React dashboard reports progress and results as the job executes.

It demonstrates a modern distributed architecture built from polyglot
services, a Redis-backed job queue, a pluggable solver layer, and
containerized deployment across Vercel and Fly.io.

EvoFarm is currently an open-source engineering prototype and research
platform. It has a working deployed path, but it is not yet a hardened
multi-tenant production service. Authentication is implemented in the Go
API layer and is being integrated into the public job flow; observability
and autoscaling remain future work.

## What it demonstrates

-   **Hybrid solver routing:** evolutionary search for heuristic workloads
  and OR-Tools CP-SAT for discrete constraint problems.
-   **Distributed execution:** jobs are queued in Redis and processed by
  independent Python workers.
-   **Live observability:** the dashboard polls job state, fitness history,
  solver metadata, and generated nurse schedules.
-   **Authentication foundation:** bcrypt password hashing, JWT access
  tokens, Bearer-token parsing, and authenticated-user context support.
-   **Two reference workloads:** XOR neuroevolution and nurse rostering
  with coverage, consecutive-work, and night-rest constraints.
-   **Benchmark tooling:** CP-SAT instances can be run locally with CSV and
  Markdown reports written to `docs/benchmarks/`.

## Screenshots

### XOR (Evolution)

![XOR fitness chart climbing to 0.99 over 100 generations](docs/screenshots/dashboard_xor.png)

*Evolution solving XOR — fitness climbs from ~0.5 to 0.99 in 100 generations.*

### Nurse Rostering (CP-SAT)

![Color-coded 6×7 nurse schedule solved to optimality in 0.185 seconds](docs/screenshots/dashboard_cpsat.png)

*A CP-SAT schedule for 6 nurses over 7 days — solved to optimality in 0.221s.*

### Portfolio Optimization (Evolution)

![Portfolio allocation bars showing 33% real estate, 24% corporate bonds](docs/screenshots/dashboard_portfolio.png)

*Evolution optimizing an 8-asset portfolio — converges to Sharpe 0.812 in 50 generations.*

*Evolution optimizing a portfolio allocation to maximize Sharpe ratio.
Converges to 0.812 in 50 generations — beating brute-force local search (0.789).*

### Job History

Every job is persisted in Postgres. The History tab shows the full
audit trail — click any row to reload the job's fitness chart, evolved
weights, or schedule.

![History tab](docs/screenshots/dashboard_history.png)

*Postgres-backed history. Jobs survive Redis flushes, worker restarts,
and redeployments.*

### Authentication

The API includes an authentication layer implemented in `api/auth.go`:

-   Passwords are hashed with bcrypt using cost 12 and are never stored in
  plaintext.
-   Successful registration and login issue an HMAC-SHA256 JWT.
-   Tokens contain the user ID (`sub`), email, issue time, and a 24-hour
  expiration.
-   Protected handlers read tokens from
  `Authorization: Bearer <token>` and can access the authenticated user
  ID through request context.
-   Login failures return a generic `invalid credentials` response rather
  than revealing whether an email exists.

The current source includes registration, login, profile, and middleware
handlers. Before enabling them in a deployment, configure `JWT_SECRET`,
apply the user schema, and register the routes in `api/main.go`. The
checked-in job routes remain usable independently while this integration
is completed.

## Try the deployed app

Open the [EvoFarm dashboard](https://evofarm.vercel.app/) to submit a job
and monitor its progress. The dashboard is deployed on Vercel and uses the
public Go API deployed on Fly.io.

Check the API directly:

``` bash
curl https://evofarm-api.fly.dev/health
```

Expected response:

``` json
{"status":"healthy"}
```

The current public deployment is intended for demonstration and may be
reset, rate-limited, or unavailable while the project is being developed.

## Why EvoFarm?

Many optimization problems are not well served by a single algorithm.
Some require broad heuristic exploration; others are dominated by
explicit constraints and feasibility rules.

EvoFarm explores a **solver-portfolio architecture** where different
optimization strategies can run behind the same observable job
interface:

-   **Evolutionary search** for population-based exploration and
    heuristic optimization.
-   **CP-SAT** for discrete constraint problems with explicit
    feasibility requirements.
-   **Future hybrid methods** for learned solver selection, warm starts,
    neighborhood selection, and adaptive search.

The project is intentionally lightweight: the goal is not to replace
mature optimization libraries, but to provide a reusable infrastructure
for running, comparing, and observing multiple optimization strategies as
coordinated workloads.

## Architecture

``` text
                         ┌──────────────────────┐
                         │   React Dashboard    │
                         │       :3000          │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │       Go API         │
                         │       :8080          │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │        Redis         │
                         │       :6379          │
                         │  queue + job state   │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │   Python Workers     │
                         │                      │
                         │  ┌────────────────┐  │
                         │  │ Evolution      │  │
                         │  │ CP-SAT         │  │
                         │  │ Environments   │  │
                         │  └────────────────┘  │
                         └──────────────────────┘
```

The system separates the optimization workload from the execution
infrastructure:

-   The **Go API** accepts jobs and exposes their state.
-   **Redis** provides the queue and persisted job metadata.
-   **Python workers** dispatch jobs to pluggable solvers and problem
    environments.
-   The **React dashboard** makes progress, fitness history, solver
    metadata, and schedules visible.

## Services

  Service         Technology   Responsibility
  --------------- ------------ -----------------------------------------------
  API             Go           Job creation, status, health checks, CORS
  Queue / State   Redis        Job queue and persisted job metadata
  Worker          Python       Solver execution and result persistence
  Dashboard       React        Job submission, monitoring, and visualization

## Current Capabilities

-   Distributed job execution through Redis
-   Evolutionary solver for a 2-4-1 neural network solving XOR
-   CP-SAT solver for nurse rostering with coverage and rest constraints
-   Persisted progress, fitness history, schedules, and solver metadata
-   React dashboard with live polling, fitness charts, and schedule
    visualization
-   CORS support for browser-to-API development workflows
-   Pluggable separation between solvers and problem environments
-   Benchmark runner and markdown/CSV reporting for CP-SAT instances

The implementation is intentionally modular and easy to extend. It is
well suited to research and experimentation, but it is still an early
prototype rather than a fully productized optimization platform.

## Current benchmark status

The repository includes a baseline CP-SAT benchmark suite under
`docs/benchmarks/` generated by the worker benchmark runner. The current
report is representative and useful for validation, but it should be read
as a prototype benchmark rather than a final production benchmark dataset.

To run the benchmark locally:

``` bash
cd worker
python3 -m pip install -r requirements.txt
python3 run_benchmarks.py --difficulty small --time-limit 30
```

The current benchmark configuration focuses on small-to-medium nurse
rostering problems and is suitable for verifying solver behavior and
infrastructure wiring. It is not yet a large-scale stress benchmark with
deep industrial coverage.

## Run locally

### Requirements

-   Docker Desktop with Docker Compose
-   `curl` for API requests
-   Python 3.11+ only if running the local XOR regression test

### Start the platform

From the project root:

``` bash
docker compose up --build
```

Open the local dashboard:

``` text
http://localhost:3000
```

The local API is available at:

``` text
http://localhost:8080
```

Redis is exposed at:

``` text
localhost:6379
```

Check local API health:

``` bash
curl http://localhost:8080/health
```

Expected response:

``` json
{"status":"healthy"}
```

## Dashboard Workflow

1.  Open `http://localhost:3000`.
2.  Choose a population size and number of generations.
3.  Select the Evolution or CP-SAT solver.
4.  Configure and submit the job.
5.  Select the job to view its live status and result.
6.  Review the fitness history for Evolution jobs or the nurse schedule
    for CP-SAT jobs.

The dashboard polls the API every two seconds during development.

## Submit a Job Through the API

``` bash
curl -X POST https://evofarm-api.fly.dev/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "population_size": 100,
    "generations": 50,
    "fitness_function": "xor",
    "solver_type": "evolution"
  }'
```

Example response:

``` json
{"job_id":"0643fdcd278a62a5cb103463a8843e03","status":"pending"}
```

Query the job using the returned ID:

``` bash
curl https://evofarm-api.fly.dev/jobs/<job_id>
```

### Job Configuration

  -------------------------------------------------------------------------
  Field                                       Default Description
  ---------------------- ---------------------------- ---------------------
  `population_size`                             `100` Candidate solutions
                                                      evaluated per
                                                      generation

  `generations`                                  `50` Number of evolution
                                                      cycles

  `fitness_function`                            `xor` Fitness function used
                                                      by the worker

  `solver_type`                           `evolution` Solver used to
                                                      process the job

  `problem`                         `nurse_rostering` CP-SAT problem
                                                      definition

  `time_limit_seconds`                           `30` Maximum CP-SAT solve
                                                      time
  -------------------------------------------------------------------------

### Submit a CP-SAT job

Use `solver_type: "cpsat"`:

``` bash
curl -X POST https://evofarm-api.fly.dev/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "solver_type": "cpsat",
    "problem": "nurse_rostering",
    "time_limit_seconds": 30
  }'
```

The default nurse-rostering instance assigns six nurses across seven days
while enforcing shift coverage, one shift per nurse per day, maximum
consecutive work days, and post-night-shift rest. The same commands work
against a local API by replacing the public base URL with
`http://localhost:8080`.

## API Reference

### `GET /health`

Returns HTTP `200` when the API is running.

### `POST /jobs`

Creates a job, stores its configuration in Redis, and adds its ID to the
queue.

Returns HTTP `400` for invalid JSON and HTTP `500` when Redis storage or
queueing fails.

### `GET /jobs/{id}`

Returns job status and results. Returns HTTP `404` when the job does not
exist.

Possible job states:

``` text
pending
running
completed
failed
```

Jobs are dispatched using `solver_type`. Current solvers are:

``` text
evolution
cpsat
```

The status response includes:

-   `best_fitness`
-   `progress`
-   `history`
-   `total_generations`
-   `best_individual`
-   `result_meta`
-   `error`

`result_meta` contains solver-specific metadata such as problem name,
solve status, solve time, and schedule dimensions.

## Authentication Configuration

Set a strong, private signing secret in the API environment:

``` bash
JWT_SECRET=replace-with-a-long-random-secret
```

Never commit `JWT_SECRET` to the repository or expose it to the frontend.
The API exits at startup if token generation or validation is attempted
without this setting.

The authentication implementation expects the `users` table to contain
`email`, `password_hash`, and `name` columns. Extend `docs/schema.sql`
before enabling registration and login in a fresh database.

## Evolutionary Solver

The current Evolution worker evolves a **2-4-1 feedforward neural
network** for XOR.

The network contains 17 parameters:

-   8 input-to-hidden weights
-   4 hidden-layer biases
-   4 hidden-to-output weights
-   1 output bias

The activation function is `tanh`.

Fitness is calculated from the four XOR truth-table inputs:

``` text
fitness = 1 / (1 + total_squared_error)
```

Fitness is maximized.

The implementation is separated into:

``` text
worker/solvers/evolution.py
worker/environments/xor.py
```

This separation allows new evolutionary environments to be added without
changing the worker queue loop.

## CP-SAT Solver

The CP-SAT implementation is in:

``` text
worker/solvers/cpsat.py
```

The nurse-rostering problem is in:

``` text
worker/problems/nurse_rostering.py
```

CP-SAT results are rendered in the dashboard as a nurse-by-day schedule
with solver status, objective value, and solve time.

## Monitor a Job from the Terminal

Use the included watcher to refresh status every two seconds:

``` bash
chmod +x watch_job.sh
./watch_job.sh <job_id>
```

Press `Ctrl+C` to stop monitoring.

## Run the XOR Regression Test

Run the standalone deterministic test:

``` bash
python3 test_xor.py
```

The test checks all four XOR cases and exits with an assertion failure
if any prediction is incorrect.

## Research Direction

EvoFarm is intended to explore the infrastructure around **distributed
optimization**, rather than compete with mature algorithm libraries.

Potential research directions include:

-   Distributed population evaluation
-   Solver portfolios and automatic solver selection
-   Learned initialization and warm starts
-   Adaptive neighborhood selection
-   Hybrid evolutionary + constraint optimization
-   Reinforcement-learning environments such as CartPole
-   Large-scale scheduling and resource allocation
-   Kubernetes-based worker auto-scaling
-   Experiment tracking and reproducibility

The long-term idea is to make optimization workloads behave more like
production distributed jobs: **submit → schedule → execute → observe →
compare → iterate**.

See [docs/research/](docs/research/) for technical rationale and design
notes.

## Roadmap

-   [x] Distributed evolution with Redis queue
-   [x] Real-time fitness chart
-   [x] Persisted fitness history
-   [x] CP-SAT nurse-rostering prototype
-   [ ] CartPole and Flappy Bird environments
-   [ ] Distributed population evaluation
-   [ ] Solver portfolio orchestration
-   [x] Authentication foundation: bcrypt passwords and JWTs
-   [ ] Wire authentication routes and protect job endpoints
-   [ ] Associate jobs with authenticated users and enforce ownership
-   [ ] Multi-tenancy, roles, and account management
-   [ ] Kubernetes deployment with auto-scaling
-   [ ] Experiment tracking and reproducibility
-   [ ] Hybrid evolutionary + constraint optimization

## Configuration

### Redis

The API and worker read the Redis address from `REDIS_ADDR`:

``` bash
REDIS_ADDR=redis:6379
```

When running services outside Docker, the default is:

``` text
localhost:6379
```

### Frontend

The frontend reads the API URL from `REACT_APP_API_URL`:

``` bash
REACT_APP_API_URL=http://localhost:8080
```

### Postgres persistence

EvoFarm uses Postgres as the durable job-history store and Redis as the
live queue and execution-state store.

When a job is submitted, the API first writes its pending state to Redis
and then attempts to insert a matching row into the Postgres `jobs` table.
The Redis write remains authoritative for queueing, so a Postgres write
failure is logged without rejecting an otherwise valid job submission.
When a worker starts, completes, or fails a job, it updates the matching
Postgres row with the solver, result, history, metadata, or error.

Postgres persistence is optional for local development. If `DATABASE_URL`
is not set, the API and worker continue using Redis only:

``` bash
DATABASE_URL=postgresql://user:password@host:5432/database
```

For a configured Postgres database, apply the schema before starting the
API and workers:

``` bash
psql "$DATABASE_URL" -f docs/schema.sql
docker compose up --build
```

The schema creates:

-   `jobs`, which stores the problem, solver type, configuration, status,
    fitness, solution payload, fitness history, result metadata, errors,
    and timestamps.
-   `users`, which supports the authentication foundation and is intended
  to become the basis for multi-user ownership.
-   Indexes for user, status, and newest-job queries.
-   An `updated_at` trigger for tracking job changes.

The worker uses short-lived `psycopg` connections for persistence and
continues processing jobs if a database update fails. Redis-backed status
is still available through `GET /jobs/{id}` during the live run.

## Project Structure

``` text
.
├── api/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go
├── frontend/
│   ├── Dockerfile
│   ├── package.json
│   ├── public/
│   └── src/
│       ├── App.js
│       └── ScheduleGrid.js
├── worker/
│   ├── Dockerfile
│   ├── db.py
│   ├── main.py
│   ├── environments/
│   │   └── xor.py
│   ├── problems/
│   │   └── nurse_rostering.py
│   ├── solvers/
│   │   ├── cpsat.py
│   │   └── evolution.py
│   └── requirements.txt
├── shared/
│   └── types.py
├── docs/
│   ├── schema.sql
│   └── research/
├── docker-compose.yml
├── test_xor.py
└── watch_job.sh
```

## Stop the Application

Stop containers:

``` bash
docker compose down
```

Stop containers and remove Redis data:

``` bash
docker compose down -v
```

## Troubleshooting

View service logs:

``` bash
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f frontend
docker compose logs -f redis
```

If the dashboard reports a network or CORS error, confirm that:

-   the API is running on port `8080`
-   the frontend uses `REACT_APP_API_URL=http://localhost:8080`
-   the API was rebuilt after source changes

``` bash
docker compose up --build
```

If the API cannot connect to Redis, confirm that the Redis service is
running and that container services use:

``` text
redis:6379
```

rather than:

``` text
localhost:6379
```

Worker output uses unbuffered Python logging, so startup, solver, and
per-generation messages should appear immediately with:

``` bash
docker compose logs -f worker
```

## License

This project is licensed under the [MIT License](LICENSE).
