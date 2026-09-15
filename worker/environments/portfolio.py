"""
Portfolio optimization environment.

Evolve a set of asset weights to maximize the Sharpe ratio of a
portfolio. Weights represent the fraction of capital allocated to
each asset and must sum to 1.

This is a continuous, non-linear optimization problem — the exact
kind where neuroevolution outperforms constraint solvers.

Assets and their historical statistics are synthetic but realistic.
"""
import math
from typing import List


# Asset universe: (name, expected annual return, annual volatility)
# All numbers are illustrative — no real market data.
ASSETS = [
    ("US Equity",        0.11, 0.16),
    ("EU Equity",        0.10, 0.18),
    ("Emerging Markets", 0.15, 0.24),
    ("Government Bonds", 0.04, 0.05),
    ("Corporate Bonds",  0.06, 0.08),
    ("Gold",             0.06, 0.14),
    ("Real Estate",      0.08, 0.10),
    ("Commodities",      0.07, 0.20),
]

# Simplified: assume every pair of assets has the same correlation.
# A production implementation would use a full correlation matrix.
BASE_CORRELATION = 0.3

RISK_FREE_RATE = 0.02  # 2% annual — the "no-risk" baseline


def evolution_params() -> dict:
    """
    Environment-specific evolution hyperparameters.

    Portfolio has a rugged fitness landscape, so it needs:
      - Larger population (more diversity)
      - More aggressive selection (top 10% instead of top 20%)
      - Higher mutation (bigger jumps to escape local optima)
    """
    return {
        "elitism_fraction": 0.1,      # keep top 10%
        "mutation_rate": 0.35,         # 35% of weights mutate per child
        "mutation_strength": 1.0,      # larger jumps
    }


def genome_size() -> int:
    """One weight per asset."""
    return len(ASSETS)


def describe() -> dict:
    """Metadata for API/dashboard."""
    return {
        "name": "portfolio",
        "description": "Maximize Sharpe ratio of a multi-asset portfolio",
        "genome_size": genome_size(),
        "max_fitness": None,  # no theoretical max
        "assets": [name for name, _, _ in ASSETS],
    }


def _softmax_weights(raw: List[float]) -> List[float]:
    """
    Convert raw genome values into portfolio weights that:
      - are all >= 0
      - sum to exactly 1

    We use softmax to enforce these automatically. This lets evolution
    optimize freely in R^n while we map the result into the valid
    simplex. No explicit constraint handling needed.
    """
    # Subtract max for numerical stability (prevents exp overflow)
    m = max(raw)
    exps = [math.exp(x - m) for x in raw]
    total = sum(exps)
    return [e / total for e in exps]


def _portfolio_stats(weights: List[float]):
    """Return (expected_return, volatility) for a weight vector."""
    # Expected return: weighted sum of asset returns
    expected_return = sum(
        w * ret for w, (_, ret, _) in zip(weights, ASSETS)
    )

    # Variance: Σ wᵢ²σᵢ² + Σᵢ≠ⱼ wᵢwⱼρσᵢσⱼ
    variance = 0.0
    for i, (w_i, (_, _, sigma_i)) in enumerate(zip(weights, ASSETS)):
        for j, (w_j, (_, _, sigma_j)) in enumerate(zip(weights, ASSETS)):
            if i == j:
                variance += (w_i ** 2) * (sigma_i ** 2)
            else:
                variance += w_i * w_j * BASE_CORRELATION * sigma_i * sigma_j

    volatility = math.sqrt(max(variance, 1e-12))
    return expected_return, volatility


def fitness(genome: List[float]) -> float:
    """
    Sharpe ratio of the portfolio.

    Higher is better. In practice:
      - Sharpe < 0.5   : poor
      - Sharpe ~ 1.0   : good
      - Sharpe > 1.5   : excellent
      - Sharpe > 2.0   : suspicious (usually overfit)
    """
    if len(genome) != genome_size():
        return 0.0

    weights = _softmax_weights(genome)
    expected_return, volatility = _portfolio_stats(weights)

    sharpe = (expected_return - RISK_FREE_RATE) / volatility

    # Clamp negative Sharpes to 0 (bad portfolios shouldn't get partial credit)
    return max(sharpe, 0.0)