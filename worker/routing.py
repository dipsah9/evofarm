"""
Solver routing.

Maps problem names to the solver that handles them.

Evolution wins on:
  - Continuous spaces (weights, parameters)
  - Non-linear objectives (Sharpe ratio, game scores)
  - Problems where you can only score an answer

CP-SAT wins on:
  - Discrete choices (assignments, schedules)
  - Hard constraints (must/must-not)
  - Problems where you want provable optimality

TO ADD A NEW PROBLEM:
  1. Create worker/environments/foo.py OR worker/problems/foo.py
  2. Add an entry to ROUTING below
  3. Add an entry to PROBLEM_CATALOG below
  (Yes, this is 3 edits. When it becomes painful, we'll refactor
   to auto-discovery.)
"""
from typing import Dict


# Routing table: problem name → solver type
ROUTING = {
    # Evolution environments
    "xor": "evolution",
    "portfolio": "evolution",

    # CP-SAT problems
    "nurse_rostering": "cpsat",
}


# Human-readable catalog for the frontend
PROBLEM_CATALOG = {
    "xor": {
        "solver": "evolution",
        "description": "Evolve a neural network to solve the XOR truth table",
        "category": "puzzle",
    },
    "portfolio": {
        "solver": "evolution",
        "description": "Maximize Sharpe ratio of a multi-asset portfolio",
        "category": "optimization",
    },
    "nurse_rostering": {
        "solver": "cpsat",
        "description": "Assign nurses to shifts satisfying coverage and rest rules",
        "category": "scheduling",
    },
}


def resolve(problem: str) -> str:
    """Given a problem name, return the solver that should handle it."""
    if not problem:
        return "auto"
    return ROUTING.get(problem, "auto")


def list_problems() -> Dict[str, dict]:
    """Return the full catalog for the frontend."""
    return PROBLEM_CATALOG