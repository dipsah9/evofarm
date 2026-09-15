"""
Neuroevolution solver.

A genetic algorithm that evolves a population of genomes using
tournament selection, single-point crossover, Gaussian mutation,
and elitism.

The solver is agnostic to what the genomes represent — it just
asks the environment to score each one.
"""
import random
from typing import Callable, Generator, List, Tuple


def create_random_genome(size: int) -> List[float]:
    """A fresh genome of random weights in [-1, 1]."""
    return [random.uniform(-1, 1) for _ in range(size)]


def mutate(genome: List[float], mutation_rate: float = 0.15,
           mutation_strength: float = 0.5) -> List[float]:
    """Return a copy of the genome with random tweaks."""
    new_genome = genome.copy()
    for i in range(len(new_genome)):
        if random.random() < mutation_rate:
            new_genome[i] += random.gauss(0, mutation_strength)
            # Keep values bounded so they can't explode
            new_genome[i] = max(-5.0, min(5.0, new_genome[i]))
    return new_genome


def crossover(parent1: List[float], parent2: List[float]) -> List[float]:
    """Single-point crossover."""
    point = random.randint(1, len(parent1) - 1)
    return parent1[:point] + parent2[point:]


def evolve(
    fitness_fn: Callable[[List[float]], float],
    genome_size: int,
    population_size: int,
    generations: int,
    elitism_fraction: float = 0.2,
    mutation_rate: float = 0.15,
    mutation_strength: float = 0.5,
) -> Generator[Tuple[float, float, List[float]], None, None]:
    """
    Main evolution loop.
    
    Yields (progress, best_fitness, best_genome) after every generation.
    Progress is a float from 0.0 to 1.0.
    """
    # Initialize
    population = [create_random_genome(genome_size) for _ in range(population_size)]
    best_fitness = 0.0
    best_genome = population[0]
    
    for gen in range(generations):
        # 1. Score everyone
        scores = [fitness_fn(g) for g in population]
        
        # 2. Track the champion
        gen_best = max(scores)
        if gen_best > best_fitness:
            best_fitness = gen_best
            best_genome = population[scores.index(gen_best)].copy()
        
        # 3. Select top X% as parents (elitism)
        sorted_pairs = sorted(zip(population, scores), key=lambda p: p[1], reverse=True)
        num_parents = max(2, int(elitism_fraction * population_size))
        parents = [g for g, _ in sorted_pairs[:num_parents]]
        
        # 4. Breed next generation
        next_population = parents.copy()  # keep the elite
        while len(next_population) < population_size:
            p1 = random.choice(parents)
            p2 = random.choice(parents)
            child = crossover(p1, p2)
            child = mutate(child, mutation_rate=mutation_rate,
                           mutation_strength=mutation_strength)
            next_population.append(child)
        
        population = next_population
        
        # 5. Report progress (1-indexed for humans)
        progress = (gen + 1) / generations
        yield progress, best_fitness, best_genome