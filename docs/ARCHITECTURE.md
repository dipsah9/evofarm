# EvoFarm Architecture

This document explains how EvoFarm is designed, why each choice was made, and how the pieces fit together.

---

## Overview

EvoFarm is a distributed optimization platform. Users submit problems; the platform routes each to the right solver, runs it across parallel workers, and streams results to a live dashboard.

The system has **four layers**:

1. **Presentation** — React dashboard (Vercel)
2. **API** — Go HTTP gateway (Fly.io)
3. **Coordination** — Redis queue and state (Upstash)
4. **Execution** — Python workers with pluggable solvers (Fly.io)

Each layer has a single responsibility. Each can be replaced without touching the others.

---

## 1. Presentation Layer

**Tech:** React 18, Chart.js, Axios  
**Hosted:** Vercel  
**URL:** https://evofarm.vercel.app

### Responsibilities

- Submit jobs to the API
- Poll job status every 2 seconds
- Render job progress, results, and history

### Polymorphic Rendering

The dashboard renders three different shapes based on job type:

| Job Type | Detection | Rendering |
| :--- | :--- | :--- |
| XOR | genome size = 17 | Fitness chart + weight tags |
| Portfolio | genome size = 8 | Fitness chart + allocation bars |
| Nurse Rostering | `result_meta.solver === "cpsat"` | Solver stats + schedule grid |

**Why polymorphic?** Different solvers produce fundamentally different outputs. A single UI that adapts is better than three separate UIs.

---

## 2. API Layer

**Tech:** Go 1.21, gorilla/mux, go-redis  
**Hosted:** Fly.io (Frankfurt)  
**URL:** https://evofarm-api.fly.dev

### Endpoints

| Method | Path | Purpose |
| :--- | :--- | :--- |
| `GET` | `/health` | Liveness check |
| `GET` | `/problems` | List available problems |
| `POST` | `/jobs` | Submit a new job |
| `GET` | `/jobs/{id}` | Fetch job status and results |

### Responsibilities

- Accept job submissions
- Assign a unique ID to each job
- Store job state in Redis (`pending` initially)
- Push the job ID onto the Redis queue
- Return the job ID immediately (async processing)
- Answer status queries with the latest state from Redis
- Handle CORS for the Vercel frontend

### What It Doesn't Do

The API **never solves anything**. It doesn't run evolution, doesn't call OR-Tools, doesn't touch the actual problem. It's a router and a state broker.

**Why?** Separation of concerns. The API can be scaled, restarted, or rewritten without affecting how problems are solved.

### Async by Design

The API returns in milliseconds. The job runs for seconds or minutes. Users poll for status.

```
POST /jobs          →  { "job_id": "abc...", "status": "pending" }   (instant)
GET  /jobs/abc...   →  { "status": "running", "progress": 0.4, ... }  (2s later)
GET  /jobs/abc...   →  { "status": "completed", "result": ... }       (30s later)
```

**Why async?** Long-running requests time out. Clients disconnect. Browsers close. Async is the only honest way to handle work that takes longer than an HTTP request.

---

## 3. Coordination Layer

**Tech:** Redis 7 via Upstash  
**Hosted:** Upstash (Frankfurt, same region as API and workers)

### What Redis Stores

| Key | Type | Purpose |
| :--- | :--- | :--- |
| `job_queue` | List | FIFO queue of pending job IDs |
| `job:{id}` | Hash | Full state of a single job |
| `config` | Hash | Runtime configuration (e.g., problem catalog) |

### Job State Shape

```
job:{id} = {
  status:              "pending" | "running" | "completed" | "failed",
  problem:             "xor" | "portfolio" | "nurse_rostering",
  solver_type:         "evolution" | "cpsat",
  progress:            0.0 – 1.0,
  best_fitness:        number,
  best_individual:     JSON string (weights or schedule),
  history:             JSON array of {generation, fitness},
  total_generations:   number,
  result_meta:         JSON (solver-specific metadata),
  error:               string
}
```

### Why Redis

- **Sub-millisecond latency** — workers poll continuously, low latency matters
- **Atomic queue operations** — `BRPOP` gives us safe, blocking consumption
- **No schema migrations** — hash fields evolve freely
- **Managed by Upstash** — no ops, pay-per-request

### Queue Semantics

Workers use `BRPOP` (Blocking Right Pop). This means:

- A worker **blocks** until a job appears
- When a job arrives, one worker gets it — no race conditions
- If a worker dies mid-job, the job stays in Redis for a retry

---

## 4. Execution Layer

**Tech:** Python 3.11, OR-Tools, redis-py  
**Hosted:** Fly.io (Frankfurt), 2 replicas  
**No public URL** — workers only talk to Redis

### Responsibilities

- Poll the Redis queue for pending job IDs
- Read the job configuration from Redis
- Route the job to the correct solver
- Execute the solver, streaming progress back to Redis
- Write the final result and mark the job complete

### The Dispatcher

```python
def process_job(r, job_id):
    job = r.hgetall(f"job:{job_id}")
    problem = job.get("problem", "")
    solver_type = routing.resolve(problem)

    if solver_type == "evolution":
        run_evolution(r, job_id, job)
    elif solver_type == "cpsat":
        run_cpsat(r, job_id, job)
```

**The dispatcher is the only place that knows about both solvers.** Everything else is solver-agnostic.

### The Solvers

#### `solvers/evolution.py`

A genetic algorithm that operates on genomes. It doesn't know what a genome *means* — it just asks the environment to score each one.

```python
for progress, fitness, genome in evolve(
    fitness_fn=fitness_fn,
    genome_size=genome_size,
    population_size=population_size,
    generations=generations,
):
    # stream progress to Redis
```

#### `solvers/cpsat.py`

A constraint solver that builds a CP model from a problem definition and calls OR-Tools.

```python
model = cp_model.CpModel()
# build variables and constraints from the problem module
solver = cp_model.CpSolver()
status = solver.Solve(model)
```

### The Environments and Problems

Environments (for evolution) and problems (for CP-SAT) are **plug-ins**. Each module exports:

- `describe()` — metadata
- `fitness()` (environments) — the scoring function
- `genome_size()` (environments) — genome length
- `evolution_params()` (environments) — hyperparameters
- `default_config()` (problems) — starting configuration

**Adding a problem requires exactly one file.** The rest of the system discovers it via the routing table.

---

## Data Flow: End-to-End

Here's what happens when a user submits a job:

```
1. User picks "portfolio" in the dashboard
   ↓
2. Dashboard POSTs to https://evofarm-api.fly.dev/jobs
   ↓
3. Go API:
   a. Generates job_id (16 random bytes, hex)
   b. HSET job:{id} with status=pending, problem=portfolio
   c. LPUSH job_queue job_id
   d. Returns {job_id, status: pending} immediately
   ↓
4. Worker:
   a. BRPOP job_queue → gets job_id
   b. HSET job:{id} status=running
   c. Looks up routing: portfolio → evolution
   d. Imports environments/portfolio.py
   e. Runs 50 generations, updating Redis after each
   ↓
5. Dashboard:
   a. Polls GET /jobs/{id} every 2 seconds
   b. Renders fitness chart climbing in real-time
   c. When status=completed, renders allocation bars
   ↓
6. User sees final portfolio: 33% real estate, 24% bonds, ...
```

Total time from submission to completion: ~30 seconds.

---

## Key Design Decisions

### 1. Async Job Queue

**Decision:** API returns immediately; workers process asynchronously.

**Why:** Long-running jobs can't fit inside a request/response cycle. Sync would cause timeouts and a fragile system.

### 2. Redis for Both Queue and State

**Decision:** One Redis instance handles the queue and all state.

**Why:** Simpler than running Kafka + Postgres + Redis. Redis is fast enough for both roles at this scale.

**Trade-off:** Not durable long-term. Redis restarts lose data.

### 3. Solver-Agnostic Workers

**Decision:** Workers don't know what a job means until they read the routing table.

**Why:** Adding a new solver is one file. No changes to the queue, API, or dashboard.

### 4. Polymorphic Dashboard

**Decision:** One React app renders three shapes.

**Why:** Users get a consistent experience. Developers maintain one codebase.

### 5. Auto-Routing

**Decision:** Users pick problems, not solvers.

**Why:** It shifts cognitive load. Users don't need to know what "CP-SAT" is. The platform decides.

---

## What's Next

See [docs/research/scheduling-at-scale.md](research/scheduling-scalability.md) for the roadmap toward hybrid and learned optimization.

---

## Deployment

| Component | Platform | Region | Notes |
| :--- | :--- | :--- | :--- |
| Frontend | Vercel | Global | Auto-deploys on push |
| API | Fly.io | fra | 2 machines, always on |
| Workers | Fly.io | fra | 1 primary + 1 standby |
| Redis | Upstash | fra | Pay-as-you-go |

### Costs

| Item | Monthly Cost |
| :--- | ---: |
| 2 × API machines (256MB) | ~$4.86 |
| 2 × Worker machines (256MB) | ~$4.86 |
| Upstash Redis | ~$0–1 |
| Vercel | $0 |
| **Total** | **~$10–12** |