"""
EvoFarm worker.

Pops jobs from the Redis queue and dispatches them to the right solver.
Currently supports:
  - solver_type="evolution"  -> genetic algorithm (see solvers/evolution.py)
"""
import json
import os
import sys
import traceback

import redis

# Local imports
from solvers import evolution
from environments import xor as xor_env


# ---------- Redis helpers ----------

def get_redis_connection() -> redis.Redis:
    addr = os.getenv("REDIS_ADDR", "localhost:6379")
    host, port = addr.split(":")
    return redis.Redis(host=host, port=int(port), decode_responses=True)


# ---------- Solvers ----------

def run_evolution(r: redis.Redis, job_id: str, job: dict) -> None:
    """Run a neuroevolution job and stream progress to Redis."""
    population_size = int(job.get("population_size", 100))
    generations = int(job.get("generations", 50))
    environment = job.get("fitness_function", "xor")

    if environment != "xor":
        raise ValueError(f"Unknown environment: {environment}")

    fitness_fn = xor_env.fitness
    genome_size = xor_env.genome_size()
    
    print(f"  -> Evolution solver | env={environment} | "
          f"pop={population_size} | gens={generations}")
    
    history = []
    best_genome = None
    best_fitness = 0.0
    
    for progress, fitness, genome in evolution.evolve(
        fitness_fn=fitness_fn,
        genome_size=genome_size,
        population_size=population_size,
        generations=generations,
    ):
        best_fitness = fitness
        best_genome = genome
        history.append({"generation": len(history) + 1, "fitness": fitness})
        
        # Stream to Redis
        r.hset(f"job:{job_id}", mapping={
            "progress": progress,
            "best_fitness": fitness,
            "history": json.dumps(history),
        })
        
        # Print to worker log every generation (or every few for long runs)
        print(f"     gen {len(history):3d} | fitness = {fitness:.4f}")
    
    # Final write
    r.hset(f"job:{job_id}", mapping={
        "status": "completed",
        "progress": 1.0,
        "best_fitness": best_fitness,
        "best_individual": json.dumps(best_genome),
        "history": json.dumps(history),
        "total_generations": len(history),
    })


# ---------- Dispatch table ----------

SOLVERS = {
    "evolution": run_evolution,
    # "cpsat": run_cpsat,   # <-- Day 2
}


# ---------- Job processing ----------

def process_job(r: redis.Redis, job_id: str) -> None:
    job = r.hgetall(f"job:{job_id}")
    if not job:
        print(f"Job {job_id} not found, skipping")
        return
    
    solver_type = job.get("solver_type", "evolution")
    print(f"Job {job_id} | solver={solver_type}")
    
    r.hset(f"job:{job_id}", "status", "running")
    
    try:
        solver_fn = SOLVERS.get(solver_type)
        if solver_fn is None:
            raise ValueError(f"Unknown solver_type: {solver_type}")
        solver_fn(r, job_id, job)
        print(f"Job {job_id} complete")

    except Exception as e:
        error_msg = f"{type(e).__name__}: {e}"
        print(f"Job {job_id} FAILED: {error_msg}")
        traceback.print_exc()
        r.hset(f"job:{job_id}", mapping={
            "status": "failed",
            "error": error_msg,
        })


# ---------- Main loop ----------

def main() -> None:
    r = get_redis_connection()
    print("=" * 60)
    print("EvoFarm worker started")
    print(f"  Redis: {os.getenv('REDIS_ADDR', 'localhost:6379')}")
    print(f"  Solvers: {list(SOLVERS.keys())}")
    print("=" * 60)
    
    while True:
        try:
            result = r.brpop("job_queue", timeout=5)
            if result:
                _, job_id = result
                process_job(r, job_id)
        except KeyboardInterrupt:
            print("\nShutting down...")
            sys.exit(0)
        except Exception as e:
            print(f"Worker loop error: {e}")
            traceback.print_exc()


if __name__ == "__main__":
    main()