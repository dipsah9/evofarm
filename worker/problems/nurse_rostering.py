"""
Nurse rostering problem.

Given N nurses and D days, assign each nurse to one shift per day
(morning, evening, night, or off) such that:

HARD constraints (must be satisfied):
  1. Every day, exactly 2 nurses work morning, 2 work evening, 1 works night
  2. No nurse works two shifts on the same day
  3. No nurse works more than 5 days in a row
  4. Night shift must be followed by a rest day (no shift the next day)

SOFT constraints (maximize satisfaction):
  - Nurse preferences for certain shifts
  - Balanced workload across nurses

This is a classic problem in operations research. The worker
returns a schedule grid showing each nurse's assignment per day.
"""
from typing import Any, Dict, List


# Shift codes
OFF = 0
MORNING = 1
EVENING = 2
NIGHT = 3

SHIFT_NAMES = {
    OFF: "off",
    MORNING: "morning",
    EVENING: "evening",
    NIGHT: "night",
}

# Staffing requirements per day
REQUIRED = {
    MORNING: 2,
    EVENING: 2,
    NIGHT: 1,
}


def default_config() -> Dict[str, Any]:
    """Default problem instance for quick testing."""
    return {
        "num_nurses": 6,
        "num_days": 7,
        # (nurse_index, day_index) -> preferred shift (optional)
        # Empty by default; can be set via the API request
        "preferences": {},
        # Max days a nurse can work in a row before a rest
        "max_consecutive_days": 5,
    }


def validate_config(config: Dict[str, Any]) -> None:
    """Sanity check the problem configuration."""
    num_nurses = config.get("num_nurses", 0)
    num_days = config.get("num_days", 0)
    
    if num_nurses <= 0:
        raise ValueError("num_nurses must be positive")
    if num_days <= 0:
        raise ValueError("num_days must be positive")
    
    # Per day, we need 2+2+1 = 5 nurses working
    nurses_needed_per_day = sum(REQUIRED.values())
    if num_nurses < nurses_needed_per_day:
        raise ValueError(
            f"Need at least {nurses_needed_per_day} nurses to staff each day, "
            f"got {num_nurses}"
        )
    
    # Total shifts needed across the horizon
    total_shifts = nurses_needed_per_day * num_days
    # Each nurse can work at most num_days shifts
    max_possible_shifts = num_nurses * num_days
    if total_shifts > max_possible_shifts:
        raise ValueError("Not enough nurses to cover all shifts")


def describe() -> Dict[str, Any]:
    """Metadata about this problem for the API/dashboard."""
    return {
        "name": "nurse_rostering",
        "description": "Assign nurses to shifts satisfying coverage and rest rules",
        "default_config": default_config(),
        "shift_names": SHIFT_NAMES,
    }