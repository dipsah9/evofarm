# EvoFarm

EvoFarm is a distributed neuroevolution platform for evolving a small neural network against the XOR problem. It combines a React dashboard, Go API, Redis-backed job queue, and Python workers.

![EvoFarm Dashboard](docs/screenshots/dashboard.png)

## Architecture

```text
Browser dashboard (:3000)
          |
          v
Go API (:8080) <---- CORS ----> Redis (:6379) <---- Python workers
                                  |
                                  +-- job queue
                                  +-- job state, progress, history, results
```

## Services

- **Frontend**: React dashboard for submitting jobs and viewing status, progress, fitness history, and evolved weights.
- **API**: Go HTTP service for job creation, status queries, health checks, and browser CORS support.
- **Redis**: Stores queued job IDs and job state.
- **Worker**: Python neuroevolution service that evaluates the 2-4-1 neural network and updates Redis.

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
3. Submit the XOR evolution job.
4. Select the job to view its live status and fitness chart.
5. Review the final fitness and evolved network weights when the job completes.

The dashboard polls the API every two seconds. The API allows browser requests from the frontend during development through CORS headers.

## Submit a Job Through the API

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "population_size": 100,
    "generations": 50,
    "fitness_function": "xor"
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

## API Reference

### `GET /health`

Returns HTTP `200` when the API is running.

### `POST /jobs`

Creates a job, stores its configuration in Redis, and adds its ID to the job queue. Returns HTTP `400` for invalid JSON and HTTP `500` when Redis storage or queueing fails.

### `GET /jobs/{id}`

Returns job status and results. Returns HTTP `404` when the job does not exist.

Possible job states are `pending`, `running`, `completed`, and `failed`.

The status response includes:

- `best_fitness`: best fitness found so far
- `progress`: value from `0.0` to `1.0`
- `history`: fitness value recorded for each generation
- `total_generations`: number of completed generations
- `best_individual`: final 17-parameter network when completed
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
- [ ] Hybrid machine-learning-guided solver

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
├── worker/
│   ├── Dockerfile
│   ├── main.py
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

## License

This project is licensed under the [MIT License](LICENSE).
