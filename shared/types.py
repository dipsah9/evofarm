from typing import List, Dict, Any
from dataclasses import dataclass, asdict
import json

@dataclass
class EvolutionJob:
    """A job to evolve a neural network"""
    job_id: str
    status: str  # 'pending', 'running', 'completed', 'failed'
    population_size: int
    generations: int
    fitness_function: str  # 'xor', 'cartpole', etc.
    best_fitness: float
    best_individual: List[float]  # Flattened weights
    progress: float  # 0.0 to 1.0
    error: str
    
    def to_json(self) -> str:
        return json.dumps(asdict(self))
    
    @classmethod
    def from_json(cls, data: str):
        return cls(**json.loads(data))