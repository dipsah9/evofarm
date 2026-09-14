"""
Generates nurse rostering benchmark instances.

Each instance is a dict with:
  - id: unique identifier
  - config: the nurse_rostering config
  - expected_difficulty: rough label (small/medium/large)
"""
from typing import Dict, List


def small_instances() -> List[Dict]:
    """6-8 nurses, 7 days."""
    return [
        {"id": "small-01", "config": {"num_nurses": 6, "num_days": 7},  "difficulty": "small"},
        {"id": "small-02", "config": {"num_nurses": 7, "num_days": 7},  "difficulty": "small"},
        {"id": "small-03", "config": {"num_nurses": 8, "num_days": 7},  "difficulty": "small"},
        {"id": "small-04", "config": {"num_nurses": 6, "num_days": 14}, "difficulty": "small"},
        {"id": "small-05", "config": {"num_nurses": 8, "num_days": 10}, "difficulty": "small"},
    ]


def medium_instances() -> List[Dict]:
    """10-15 nurses, 14-21 days."""
    return [
        {"id": "medium-01", "config": {"num_nurses": 10, "num_days": 14}, "difficulty": "medium"},
        {"id": "medium-02", "config": {"num_nurses": 12, "num_days": 14}, "difficulty": "medium"},
        {"id": "medium-03", "config": {"num_nurses": 15, "num_days": 14}, "difficulty": "medium"},
        {"id": "medium-04", "config": {"num_nurses": 12, "num_days": 21}, "difficulty": "medium"},
        {"id": "medium-05", "config": {"num_nurses": 15, "num_days": 21}, "difficulty": "medium"},
    ]


def large_instances() -> List[Dict]:
    """20+ nurses, 28 days."""
    return [
        {"id": "large-01", "config": {"num_nurses": 20, "num_days": 28}, "difficulty": "large"},
        {"id": "large-02", "config": {"num_nurses": 25, "num_days": 28}, "difficulty": "large"},
        {"id": "large-03", "config": {"num_nurses": 30, "num_days": 28}, "difficulty": "large"},
    ]


def all_instances() -> List[Dict]:
    """All benchmark instances, sorted small → large."""
    return small_instances() + medium_instances() + large_instances()