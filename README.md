# EvoFarm

## Optimization Infrastructure for Distributed Search

EvoFarm is an experimental platform for running evolutionary and constraint-based optimization as distributed workloads.

The project separates the **optimization workload** from the **execution infrastructure**:

- A Go API accepts jobs and exposes their state.
- Redis provides the queue and durable job metadata.
- Python workers dispatch jobs to pluggable solvers and problem environments.
- A React dashboard makes progress, fitness history, solver metadata, and schedules visible.

EvoFarm is not intended to be another general-purpose evolutionary algorithms library. Its focus is the operational layer around optimization: submitting work, selecting a solver, distributing execution, tracking progress, and returning structured results.

![EvoFarm Dashboard](docs/screenshots/dashboard.png)

## Why EvoFarm?

Real scheduling and optimization systems combine hard constraints with changing conditions: resource conflicts, precedence relationships, machine availability, new arrivals, and strict response-time requirements. Different parts of the problem may require different techniques.

EvoFarm explores a solver-portfolio approach:

- **Evolutionary search** for population-based exploration and heuristic optimization.
- **CP-SAT** for discrete constraint problems with explicit feasibility rules.
- **Future hybrid methods** for using learned guidance to improve solver choices, warm starts, or neighborhood selection.

The goal is not to replace exact optimization with machine learning. The goal is to make multiple optimization strategies available behind one observable, distributed job interface.

## Architecture

```text
Browser dashboard (:3000)
          |
          v
Go API (:8080) <----> Redis (:6379) <----> Python workers
  ^                    |                    |
  |                    +-- job queue        +-- solver modules
  |                    +-- job state        +-- problem environments
```

## Services

- **Frontend**: React dashboard for submitting jobs and viewing status, progress, fitness history, and evolved weights.
- **API**: Go HTTP service for job creation, status queries, health checks, and browser CORS support.
- **Redis**: Stores queued job IDs and job state.
- **Worker**: Python service that dispatches jobs to solver modules, evaluates problem environments, and updates Redis.

## Current Capabilities

- Distributed job execution through Redis
- Evolutionary solver for a 2-4-1 neural network solving XOR
- CP-SAT solver for nurse rostering with coverage and rest constraints
- Persisted progress, fitness history, schedules, and solver metadata
- React dashboard with live polling, fitness charts, and schedule visualization
- CORS support for browser-to-API development workflows

The current implementation is an early research and engineering prototype. The roadmap describes planned environments, deployment options, and hybrid optimization features.

## Requirements

- Docker Desktop with Docker Compose
- `curl` for API requests
- Python 3.11+ only for running the local XOR test directly

## Quick Start

From the project root, build and start the complete application:

```bash
docker compose up --build
```

Open the dashboard at:

```text
http://localhost:3000
```

The API is available at `http://localhost:8080`, and Redis is exposed at `localhost:6379`.

Check the API health endpoint:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"healthy"}
```

## Dashboard Workflow

1. Open `http://localhost:3000`.
2. Choose a population size and number of generations.
3. Select either the Evolution or CP-SAT solver.
4. Configure the selected job and submit it.
5. Select the job to view its live status and results.
6. Review the fitness chart for Evolution jobs or the nurse schedule for CP-SAT jobs.

The dashboard polls the API every two seconds. The API allows browser requests from the frontend during development through CORS headers.

## Submit a Job Through the API

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "population_size": 100,
    "generations": 50,
    "fitness_function": "xor",
    "solver_type": "evolution",
    "problem": "nurse_rostering",
    "time_limit_seconds": 30
  }'
```

Example response:

```json
{"job_id":"0643fdcd278a62a5cb103463a8843e03","status":"pending"}
```

Query the job using the returned ID:

```bash
curl http://localhost:8080/jobs/<job_id>
```

The request fields are optional and default to:

| Field | Default | Description |
| --- | ---: | --- |
| `population_size` | `100` | Candidate solutions evaluated per generation |
| `generations` | `50` | Number of evolution cycles |
| `fitness_function` | `xor` | Fitness function used by the worker |
| `solver_type` | `evolution` | Solver used to process the job |
| `problem` | `nurse_rostering` | CP-SAT problem definition |
| `time_limit_seconds` | `30` | Maximum CP-SAT solve time |

### Submit a CP-SAT Job

Use `solver_type: "cpsat"` to run the nurse-rostering constraint solver:

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "solver_type": "cpsat",
    "problem": "nurse_rostering",
    "time_limit_seconds": 30
  }'
```

The current problem assigns six nurses across seven days while enforcing shift coverage, one shift per nurse per day, maximum consecutive work days, and post-night-shift rest.

## API Reference

### `GET /health`

Returns HTTP `200` when the API is running.

### `POST /jobs`

Creates a job, stores its configuration in Redis, and adds its ID to the job queue. Returns HTTP `400` for invalid JSON and HTTP `500` when Redis storage or queueing fails.

### `GET /jobs/{id}`

Returns job status and results. Returns HTTP `404` when the job does not exist.

Possible job states are `pending`, `running`, `completed`, and `failed`.

Jobs are dispatched using `solver_type`. The available solvers are `evolution` and `cpsat`.

The status response includes:

- `best_fitness`: best fitness found so far
- `progress`: value from `0.0` to `1.0`
- `history`: fitness value recorded for each generation
- `total_generations`: number of completed generations
- `best_individual`: final 17-parameter network when completed
- `result_meta`: solver-specific metadata such as problem name, solve status, solve time, and schedule dimensions
- `error`: failure details when the job fails

## Neural Network

The worker evolves a 2-4-1 feedforward network using 17 parameters:

- 8 input-to-hidden weights
- 4 hidden-layer biases
- 4 hidden-to-output weights
- 1 output bias

The activation function is `tanh`, and XOR fitness is calculated from the four truth-table inputs. Fitness is maximized using:

```text
fitness = 1 / (1 + total_squared_error)
```

The worker keeps the evolution algorithm in `worker/solvers/evolution.py` and the XOR environment in `worker/environments/xor.py`. This separation allows additional solvers and environments to be added without changing the worker queue loop.

## CP-SAT Solver

The CP-SAT implementation is in `worker/solvers/cpsat.py`, and the nurse-rostering problem is in `worker/problems/nurse_rostering.py`. CP-SAT results are rendered in the dashboard as a color-coded nurse-by-day schedule with solver status, objective value, and solve time.

## Monitor a Job from the Terminal

Use the included watcher to refresh status every two seconds:

```bash
chmod +x watch_job.sh
./watch_job.sh <job_id>
```

The script uses `python3` on the host. Press `Ctrl+C` to stop monitoring.

## Run the XOR Regression Test

Run the standalone deterministic test with Python 3:

```bash
python3 test_xor.py
```

The test checks all four XOR cases and exits with an assertion failure if any prediction is incorrect.

## Roadmap

- [x] Distributed evolution with Redis queue
- [x] Real-time fitness chart
- [x] Persisted fitness history
- [ ] CartPole and Flappy Bird environments
- [ ] Authentication and multi-tenancy
- [ ] Kubernetes deployment with auto-scaling
- [ ] OR-Tools integration for constraint problems

See [docs/research/](docs/research/) for technical rationale and design notes.

## Configuration

The API and worker read the Redis address from `REDIS_ADDR`:

```bash
REDIS_ADDR=redis:6379
```

When running services outside Docker, the default is `localhost:6379`.

The frontend reads the API URL from `REACT_APP_API_URL`:

```bash
REACT_APP_API_URL=http://localhost:8080
```

## Project Structure

```text
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
│   └── research/
├── docker-compose.yml
├── test_xor.py
└── watch_job.sh
```

## Stop the Application

Stop containers:

```bash
docker compose down
```

Stop containers and remove Redis data:

```bash
docker compose down -v
```

## Troubleshooting

View service logs:

```bash
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f frontend
docker compose logs -f redis
```

If the dashboard reports a network or CORS error, confirm that the API is running on port `8080`, the frontend is using `REACT_APP_API_URL=http://localhost:8080`, and the API was rebuilt after source changes:

```bash
docker compose up --build
```

If the API cannot connect to Redis, confirm that the Redis service is running and that container services use `redis:6379` rather than `localhost:6379`.

Worker output is configured for unbuffered Python logging, so startup, solver, and per-generation messages should appear immediately with `docker compose logs -f worker`.

## License

This project is licensed under the [MIT License](LICENSE).
