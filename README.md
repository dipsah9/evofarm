# EvoFarm

EvoFarm is a small distributed neuroevolution service for evolving a neural network against the XOR problem. Jobs are submitted through a Go HTTP API, queued in Redis, and processed asynchronously by Python workers.

## Architecture

```text
Client
	|
	v
Go API (:8080) ----> Redis (:6379) <---- Python workers
												 |
												 +-- job queue
												 +-- job status and results
```

### Services

- **API**: Go service exposing job submission, status, and health endpoints.
- **Redis**: Persistent queue and job state storage.
- **Worker**: Python service that runs the neuroevolution loop and updates job progress.

## Requirements

- Docker Desktop with Docker Compose
- `curl` for API requests
- Python 3.11+ only if running the local XOR test directly

## Quick Start

Start all services from the project root:

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080` and Redis at `localhost:6379`.

Check that the API is healthy:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"healthy"}
```

## Submit a Job

Submit an XOR evolution job with custom population and generation settings:

```bash
curl -X POST http://localhost:8080/jobs \
	-H "Content-Type: application/json" \
	-d '{
		"population_size": 100,
		"generations": 50,
		"fitness_function": "xor"
	}'
```

The API returns a job ID:

```json
{"job_id":"0643fdcd278a62a5cb103463a8843e03","status":"pending"}
```

Save the returned `job_id` and query its status:

```bash
curl http://localhost:8080/jobs/<job_id>
```

Example completed response:

```json
{
	"job_id": "0643fdcd278a62a5cb103463a8843e03",
	"status": "completed",
	"best_fitness": 0.91,
	"progress": 1,
	"best_individual": [0.12, -0.34, 0.56, -0.78, 0.11, 0.22, -0.33, 0.44, -0.55, 0.66, -0.77, 0.88, 0.19, -0.29, 0.39, -0.49, 0.59],
	"error": ""
}
```

The request fields are optional. Defaults are:

| Field | Default | Description |
| --- | ---: | --- |
| `population_size` | `100` | Number of candidate solutions per generation |
| `generations` | `50` | Number of evolution cycles |
| `fitness_function` | `xor` | Fitness function used by the worker |

## Monitor a Job

The repository includes a terminal watcher that refreshes a job status every two seconds:

```bash
chmod +x watch_job.sh
./watch_job.sh <job_id>
```

Press `Ctrl+C` to stop monitoring.

## Run the XOR Test Locally

The standalone test uses Python 3 and the sample weights stored in `test_xor.py`:

```bash
python3 test_xor.py
```

This prints the model output and binary prediction for each row in the XOR truth table.

The network uses 17 parameters:

- 8 input-to-hidden weights for the 2-4 layer
- 4 hidden-layer biases
- 4 hidden-to-output weights for the 4-1 layer
- 1 output bias

## API Reference

### `GET /health`

Returns HTTP `200` when the API is running.

### `POST /jobs`

Creates a job and adds its ID to the Redis queue. Returns HTTP `400` for invalid JSON and HTTP `500` if the job cannot be stored or queued.

### `GET /jobs/{id}`

Returns the current job status and results. Returns HTTP `404` when the job does not exist.

Possible job states include `pending`, `running`, `completed`, and `failed`.

## Configuration

Both the API and worker read the Redis connection from the `REDIS_ADDR` environment variable.

```bash
REDIS_ADDR=redis:6379
```

When running either service outside Docker, the default is `localhost:6379`.

## Project Structure

```text
.
├── api/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go
├── shared/
│   └── types.py
├── worker/
│   ├── Dockerfile
│   ├── main.py
│   └── requirements.txt
├── docker-compose.yml
├── test_xor.py
└── watch_job.sh
```

## Stopping the Services

```bash
docker compose down
```

To remove the Redis container and its stored data as well:

```bash
docker compose down -v
```

## Troubleshooting

View service logs:

```bash
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f redis
```

If the API cannot connect to Redis, confirm that the Compose services are running and that the container address `redis:6379` is being used. If running `watch_job.sh` on macOS, use Python 3 as shown in the script.
# evofarm
