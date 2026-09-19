"""
Tests for the genetic algorithm.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from solvers import evolution
from environments import xor


def test_evolution_improves_fitness():
    """Evolution should improve fitness over 20 generations."""
    history = []
    for _, fitness, _ in evolution.evolve(
        fitness_fn=xor.fitness,
        genome_size=xor.genome_size(),
        population_size=50,
        generations=20,
    ):
        history.append(fitness)

    assert len(history) == 20
    # XOR starts around 0.5 and should improve noticeably by gen 20
    assert history[-1] >= history[0], "fitness should not decrease"
    assert history[-1] > 0.6, f"evolution plateaued at {history[-1]}"


def test_mutation_bounds():
    """Mutated genes should stay within the [-5, 5] clamp."""
    genome = [0.0] * 17
    mutated = evolution.mutate(genome, mutation_rate=1.0, mutation_strength=10.0)
    assert all(-5.0 <= g <= 5.0 for g in mutated), "mutation escaped bounds"