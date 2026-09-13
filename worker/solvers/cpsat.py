"""
CP-SAT solver.

A generic solver that calls Google OR-Tools' CP-SAT engine on a
constraint problem loaded from problems/. Currently supports:

  - nurse_rostering

The solver:
  1. Builds a CP-SAT model from the problem definition
  2. Solves with a time limit
  3. Returns a schedule grid + objective value

Progress is reported based on solver status events. CP-SAT doesn't
have per-generation progress like evolution, so we report coarse
progress: 0% (start), 50% (solving), 100% (done).
"""
import json
import time
from typing import Any, Dict

from ortools.sat.python import cp_model

from problems import nurse_rostering


# Registry of supported problems
PROBLEMS = {
    "nurse_rostering": nurse_rostering,
}


def solve_nurse_rostering(
    config: Dict[str, Any],
    time_limit_seconds: float = 30.0,
    on_progress=None,
):
    """
    Build and solve the nurse rostering CP-SAT model.
    
    Returns (schedule, objective_value, status_name).
    schedule is a list of (nurse, day) -> shift assignments.
    """
    nr = nurse_rostering
    nr.validate_config(config)
    
    num_nurses = config["num_nurses"]
    num_days = config["num_days"]
    max_consecutive = config.get("max_consecutive_days", 5)
    preferences = config.get("preferences", {})
    
    model = cp_model.CpModel()
    
    # ---- Decision variables ----
    # shifts[n][d][s] = 1 if nurse n works shift s on day d
    # We model 4 shifts per day: off, morning, evening, night
    shifts = {}
    for n in range(num_nurses):
        for d in range(num_days):
            for s in [nr.OFF, nr.MORNING, nr.EVENING, nr.NIGHT]:
                shifts[(n, d, s)] = model.NewBoolVar(f"shift_n{n}_d{d}_s{s}")
    
    # ---- HARD constraint 1: One shift per nurse per day ----
    for n in range(num_nurses):
        for d in range(num_days):
            model.AddExactlyOne(
                shifts[(n, d, s)] for s in [nr.OFF, nr.MORNING, nr.EVENING, nr.NIGHT]
            )
    
    # ---- HARD constraint 2: Staffing requirements per day ----
    for d in range(num_days):
        for shift, required in nr.REQUIRED.items():
            model.Add(
                sum(shifts[(n, d, shift)] for n in range(num_nurses)) == required
            )
    
    # ---- HARD constraint 3: No more than max_consecutive work days in a row ----
    if num_days > max_consecutive:
        for n in range(num_nurses):
            for start in range(num_days - max_consecutive):
                window = range(start, start + max_consecutive + 1)
                # Sum of "working" vars in the window must be <= max_consecutive
                working_vars = []
                for d in window:
                    is_working = model.NewBoolVar(f"working_n{n}_d{d}")
                    model.Add(is_working == 1 - shifts[(n, d, nr.OFF)])
                    working_vars.append(is_working)
                model.Add(sum(working_vars) <= max_consecutive)
    
    # ---- HARD constraint 4: Night shift followed by a rest day ----
    for n in range(num_nurses):
        for d in range(num_days - 1):
            # If night on day d, then off on day d+1
            model.AddImplication(
                shifts[(n, d, nr.NIGHT)],
                shifts[(n, d + 1, nr.OFF)],
            )
    
    # ---- SOFT constraint: Nurse preferences ----
    # Each satisfied preference adds 1 to the objective
    preference_terms = []
    for (n_str, d_str), preferred_shift_name in preferences.items():
        n = int(n_str)
        d = int(d_str)
        # Map shift name -> code
        shift_code = None
        for code, name in nr.SHIFT_NAMES.items():
            if name == preferred_shift_name:
                shift_code = code
                break
        if shift_code is None:
            continue
        if n >= num_nurses or d >= num_days:
            continue
        preference_terms.append(shifts[(n, d, shift_code)])
    
    # ---- SOFT constraint: Balanced workload ----
    # Minimize the variance-ish: penalize max total shifts per nurse
    # Simple approach: keep the spread small by adding a soft bound.
    total_shifts_per_nurse = []
    for n in range(num_nurses):
        worked = []
        for d in range(num_days):
            w = model.NewBoolVar(f"worked_n{n}_d{d}")
            model.Add(w == 1 - shifts[(n, d, nr.OFF)])
            worked.append(w)
        total_shifts_per_nurse.append(sum(worked))
    
    # Add both objectives with a combined score
    # (satisfy preferences, then balance)
    model.Maximize(
        sum(preference_terms) * 10  # preferences weigh more
        - 0  # workload balance could be added later
    )
    
    # ---- Solve ----
    solver = cp_model.CpSolver()
    solver.parameters.max_time_in_seconds = time_limit_seconds
    solver.parameters.num_search_workers = 4  # parallel search
    
    if on_progress:
        on_progress(0.5)  # mid-progress: solving started
    
    start = time.time()
    status = solver.Solve(model)
    elapsed = time.time() - start
    
    if on_progress:
        on_progress(1.0)
    
    status_name = solver.StatusName(status)
    
    if status not in (cp_model.OPTIMAL, cp_model.FEASIBLE):
        raise RuntimeError(
            f"CP-SAT could not find a solution. Status: {status_name}. "
            f"Try more nurses, fewer days, or check your constraints."
        )
    
    # ---- Extract schedule ----
    schedule = []
    for n in range(num_nurses):
        nurse_row = []
        for d in range(num_days):
            for s in [nr.OFF, nr.MORNING, nr.EVENING, nr.NIGHT]:
                if solver.Value(shifts[(n, d, s)]) == 1:
                    nurse_row.append(nr.SHIFT_NAMES[s])
                    break
        schedule.append(nurse_row)
    
    objective_value = solver.ObjectiveValue()
    
    return {
        "schedule": schedule,
        "objective_value": objective_value,
        "status": status_name,
        "solve_time_seconds": round(elapsed, 3),
        "num_nurses": num_nurses,
        "num_days": num_days,
    }