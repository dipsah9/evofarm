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
import time
import routing

import redis

# Local imports
from solvers import evolution
from environments import xor as xor_env
from solvers import cpsat


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
    environment_name = job.get("fitness_function", "").strip()
    if not environment_name:
        environment_name = "xor"  # safe default

    # Lazy import to keep startup fast and allow future environments
    if environment_name == "portfolio":
        from environments import portfolio as env_module
    elif environment_name == "xor":
        env_module = xor_env
    else:
        raise ValueError(f"Unknown environment: {environment_name}")

    fitness_fn = env_module.fitness
    genome_size = env_module.genome_size()

    print(f"  -> Evolution solver | env={environment_name} | "
          f"pop={population_size} | gens={generations} | "
          f"genome_size={genome_size}")

    history = []
    best_genome = None
    best_fitness = 0.0

     # Get environment-specific evolution params
    env_params = env_module.evolution_params() if hasattr(env_module, "evolution_params") else {}

    for progress, fitness, genome in evolution.evolve(
        fitness_fn=fitness_fn,
        genome_size=genome_size,
        population_size=population_size,
        generations=generations,
        **env_params,
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

        # Log every generation for XOR (fast), every 5 for portfolio (slower)
        if environment_name == "xor" or len(history) % 5 == 0 or len(history) == generations:
            print(f"     gen {len(history):3d} | fitness = {fitness:.4f}")

    # Final write
    r.hset(f"job:{job_id}", mapping={
        "status": "completed",
        "progress": 1.0,
        "best_fitness": best_fitness,
        "best_individual": json.dumps(best_genome),
        "history": json.dumps(history),
        "total_generations": len(history),
        "environment": environment_name,
    })

def run_cpsat(r: redis.Redis, job_id: str, job: dict) -> None:
    """Run a CP-SAT constraint job and stream results to Redis."""
    problem_name = job.get("problem", "nurse_rostering")
    time_limit = float(job.get("time_limit_seconds", 30))
    
    if problem_name not in cpsat.PROBLEMS:
        raise ValueError(f"Unknown problem: {problem_name}")
    
    # Parse config (comes as JSON string from Redis)
    config_raw = job.get("config", "{}")
    if isinstance(config_raw, str):
        config = json.loads(config_raw) if config_raw else {}
    else:
        config = config_raw
    
    # Fill in defaults
    problem_module = cpsat.PROBLEMS[problem_name]
    defaults = problem_module.default_config()
    defaults.update(config)
    config = defaults
    
    print(f"  -> CP-SAT solver | problem={problem_name} | "
          f"time_limit={time_limit}s")
    print(f"     config: {config}")
    
    # Progress callback — CP-SAT is coarse-grained
    def on_progress(fraction: float) -> None:
        r.hset(f"job:{job_id}", "progress", fraction)
    
    # Record start
    start = time.time()
    r.hset(f"job:{job_id}", mapping={
        "progress": 0.1,
        "history": json.dumps([]),
    })
    
    # Solve
    result = cpsat.solve_nurse_rostering(
        config=config,
        time_limit_seconds=time_limit,
        on_progress=on_progress,
    )
    
    elapsed = time.time() - start
    print(f"     status={result['status']} | "
          f"objective={result['objective_value']:.2f} | "
          f"time={elapsed:.2f}s")
    
    # Write final result
    r.hset(f"job:{job_id}", mapping={
        "status": "completed",
        "progress": 1.0,
        "best_fitness": result["objective_value"],
        "best_individual": json.dumps(result["schedule"]),
        "history": json.dumps([{
            "generation": 1,
            "fitness": result["objective_value"],
        }]),
        "total_generations": 1,
        "result_meta": json.dumps({
            "solver": "cpsat",
            "problem": problem_name,
            "status": result["status"],
            "solve_time_seconds": result["solve_time_seconds"],
            "num_nurses": result["num_nurses"],
            "num_days": result["num_days"],
        }),
    })


# ---------- Dispatch table ----------

SOLVERS = {
    "evolution": run_evolution,
    "cpsat": run_cpsat,
    # "cpsat": run_cpsat,   # <-- Day 2
}


# ---------- Job processing ----------

def process_job(r: redis.Redis, job_id: str) -> None:
    job = r.hgetall(f"job:{job_id}")
    if not job:
        print(f"Job {job_id} not found, skipping")
        return

    problem = job.get("problem", "")
    explicit_solver = job.get("solver_type", "")

    # Explicit solver_type wins (backward compat).
    # Otherwise, route via the hardcoded table.
    if explicit_solver:
        solver_type = explicit_solver
    else:
        solver_type = routing.resolve(problem)

    print(f"Job {job_id} | problem={problem or '(none)'} | solver={solver_type}")

    if solver_type == "auto":
        print(f"  -> Unknown problem; defaulting to evolution")
        solver_type = "evolution"

    r.hset(f"job:{job_id}", mapping={
        "status": "running",
        "solver_type": solver_type,
    })

    try:
        if solver_type == "evolution":
            # If the problem is known and fitness_function is missing/empty,
            # use the problem name as the environment name.
            if problem and not job.get("fitness_function"):
                job["fitness_function"] = problem
            run_evolution(r, job_id, job)
        elif solver_type == "cpsat":
            if problem and not job.get("problem"):
                job["problem"] = problem
            run_cpsat(r, job_id, job)
        else:
            raise ValueError(f"Unknown solver_type: {solver_type}")

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
    print(f"  Routing: {routing.ROUTING}")
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