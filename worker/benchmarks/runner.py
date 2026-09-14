"""
Benchmark runner.

Takes a list of instances and runs them through the CP-SAT solver,
recording solve time, objective, and status.
"""
import sys
import time
from pathlib import Path
from typing import Callable, Dict, List

# Allow running as a script from the repo root
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from solvers import cpsat


def run_single(config: Dict, time_limit: float = 60.0) -> Dict:
    """Run one instance and return metrics."""
    start = time.time()
    try:
        result = cpsat.solve_nurse_rostering(
            config=config,
            time_limit_seconds=time_limit,
        )
        elapsed = time.time() - start
        return {
            "status": "ok",
            "solver_status": result["status"],
            "objective": result["objective_value"],
            "solve_time": round(elapsed, 4),
            "num_nurses": result["num_nurses"],
            "num_days": result["num_days"],
            "error": "",
        }
    except Exception as e:
        elapsed = time.time() - start
        return {
            "status": "error",
            "solver_status": "ERROR",
            "objective": None,
            "solve_time": round(elapsed, 4),
            "num_nurses": config.get("num_nurses", 0),
            "num_days": config.get("num_days", 0),
            "error": f"{type(e).__name__}: {e}",
        }


def run_benchmark(
    instances: List[Dict],
    time_limit: float = 60.0,
    on_result: Callable[[Dict], None] = None,
) -> List[Dict]:
    """
    Run all instances and return a list of result dicts.
    
    If on_result is provided, it's called after each instance finishes
    — useful for streaming output to the console.
    """
    results = []
    for i, instance in enumerate(instances):
        print(f"[{i+1}/{len(instances)}] {instance['id']} "
              f"({instance['config']['num_nurses']} nurses × "
              f"{instance['config']['num_days']} days)...", end=" ", flush=True)
        
        result = run_single(instance["config"], time_limit=time_limit)
        result["id"] = instance["id"]
        result["difficulty"] = instance.get("difficulty", "unknown")
        result["config"] = instance["config"]
        
        # Console feedback
        if result["status"] == "ok":
            print(f"{result['solver_status']} in {result['solve_time']}s "
                  f"(obj={result['objective']})")
        else:
            print(f"ERROR: {result['error']}")
        
        results.append(result)
        
        if on_result:
            on_result(result)
    
    return results