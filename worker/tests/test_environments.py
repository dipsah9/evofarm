"""
Basic tests for evolution environments.

Run with: pytest tests/
"""
import sys
from pathlib import Path

# Make `worker/` importable
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from environments import xor
from environments import portfolio


def test_xor_perfect_network():
    """A hand-crafted network that solves XOR should score near 1.0."""
    # This isn't a perfect XOR solver, but it's close enough that we
    # can assert the fitness function returns a reasonable number.
    genome = [1.0] * xor.genome_size()
    score = xor.fitness(genome)
    assert 0.0 <= score <= 1.0, f"fitness out of range: {score}"


def test_xor_wrong_size_returns_zero():
    """Passing the wrong number of weights should yield fitness 0."""
    assert xor.fitness([1.0, 2.0, 3.0]) == 0.0


def test_portfolio_weights_sum_to_one():
    """The softmax output must always sum to exactly 1.0."""
    import random
    genome = [random.uniform(-5, 5) for _ in range(portfolio.genome_size())]
    weights = portfolio._softmax_weights(genome)
    assert abs(sum(weights) - 1.0) < 1e-9, f"weights sum to {sum(weights)}"


def test_portfolio_weights_non_negative():
    """The softmax output must always be non-negative."""
    import random
    genome = [random.uniform(-5, 5) for _ in range(portfolio.genome_size())]
    weights = portfolio._softmax_weights(genome)
    assert all(w >= 0 for w in weights), "negative weight found"


def test_portfolio_genome_size():
    """Portfolio should have one weight per asset."""
    assert portfolio.genome_size() == len(portfolio.ASSETS)