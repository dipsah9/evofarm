import json
import random
import math
from typing import List, Tuple
import redis
import os

# ============ NEUROEVOLUTION ENGINE ============

WEIGHT_COUNT = 17

class NeuralNetwork:
    """A simple feedforward neural network"""
    def __init__(self, weights: List[float]):
        if len(weights) != WEIGHT_COUNT:
            raise ValueError(f"Expected {WEIGHT_COUNT} weights, got {len(weights)}")
        self.weights = weights
        
    def forward(self, inputs: List[float]) -> List[float]:
        """Evaluate a 2-input, 4-hidden-neuron, 1-output network."""
        if len(inputs) != 2:
            raise ValueError(f"Expected 2 inputs, got {len(inputs)}")

        # Layout: 8 input weights, 4 hidden biases, 4 output weights, 1 output bias.
        w1 = self.weights[0:8]
        hidden_biases = self.weights[8:12]
        hidden = [
            math.tanh(w1[0] * inputs[0] + w1[1] * inputs[1] + hidden_biases[0]),
            math.tanh(w1[2] * inputs[0] + w1[3] * inputs[1] + hidden_biases[1]),
            math.tanh(w1[4] * inputs[0] + w1[5] * inputs[1] + hidden_biases[2]),
            math.tanh(w1[6] * inputs[0] + w1[7] * inputs[1] + hidden_biases[3]),
        ]
        
        w2 = self.weights[12:16]
        output_bias = self.weights[16]
        output = math.tanh(
            w2[0] * hidden[0] + w2[1] * hidden[1] +
            w2[2] * hidden[2] + w2[3] * hidden[3] + output_bias
        )
        return [output]

def create_random_weights() -> List[float]:
    """Create random weights for a 2-4-1 network."""
    return [random.uniform(-1, 1) for _ in range(WEIGHT_COUNT)]

def fitness_xor(weights: List[float]) -> float:
    """Fitness function: how well does the network solve XOR?"""
    nn = NeuralNetwork(weights)
    total_error = 0.0
    
    # XOR truth table
    xor_inputs = [
        ([0, 0], 0),
        ([0, 1], 1),
        ([1, 0], 1),
        ([1, 1], 0)
    ]
    
    for inputs, expected in xor_inputs:
        output = nn.forward(inputs)[0]
        error = (output - expected) ** 2
        total_error += error
    
    # Fitness = inverse of error (maximize)
    return 1.0 / (1.0 + total_error)  # Max fitness = 1.0

def mutate(weights: List[float], mutation_rate: float = 0.1) -> List[float]:
    """Randomly mutate weights"""
    new_weights = weights.copy()
    for i in range(len(new_weights)):
        if random.random() < mutation_rate:
            new_weights[i] += random.gauss(0, 0.5)
            # Clip to reasonable range
            new_weights[i] = max(-5, min(5, new_weights[i]))
    return new_weights

def crossover(parent1: List[float], parent2: List[float]) -> List[float]:
    """Single-point crossover between two parents"""
    point = random.randint(0, len(parent1) - 1)
    child = parent1[:point] + parent2[point:]
    return child

def evolve(population_size: int, generations: int) -> Tuple[List[float], float]:
    """Main evolution loop"""
    # Initialize population
    population = [create_random_weights() for _ in range(population_size)]
    best_fitness = 0.0
    best_individual = population[0]
    
    for gen in range(generations):
        # Evaluate fitness
        fitness_scores = [fitness_xor(ind) for ind in population]
        
        # Track best
        gen_best_fitness = max(fitness_scores)
        if gen_best_fitness > best_fitness:
            best_fitness = gen_best_fitness
            best_individual = population[fitness_scores.index(gen_best_fitness)]
        
        # Selection: tournament selection (top 20%)
        sorted_pairs = sorted(zip(population, fitness_scores), key=lambda x: x[1], reverse=True)
        top_20_percent = int(0.2 * population_size)
        parents = [ind for ind, _ in sorted_pairs[:top_20_percent]]
        
        # Create next generation
        next_population = parents.copy()  # Elitism: keep the best
        
        while len(next_population) < population_size:
            # Select two parents
            p1 = random.choice(parents)
            p2 = random.choice(parents)
            
            # Crossover
            child = crossover(p1, p2)
            
            # Mutation
            child = mutate(child, mutation_rate=0.15)
            
            next_population.append(child)
        
        population = next_population
        
        # Report progress
        print(f"Gen {gen+1}/{generations}: Best fitness = {best_fitness:.4f}")
        
        # Yield progress for the API
        yield gen / generations, best_fitness, best_individual
    
    return best_individual, best_fitness

# ============ WORKER LOGIC ============

def get_redis_connection():
    redis_addr = os.getenv("REDIS_ADDR", "localhost:6379")
    return redis.Redis(host=redis_addr.split(":")[0], 
                       port=int(redis_addr.split(":")[1]),
                       decode_responses=True)

def process_job(job_id: str):
    """Process a single evolution job"""
    r = get_redis_connection()
    
    # Get job details from Redis
    job_data = r.hgetall(f"job:{job_id}")
    if not job_data:
        print(f"Job {job_id} not found")
        return
    
    population_size = int(job_data.get("population_size", 100))
    generations = int(job_data.get("generations", 50))
    fitness_function = job_data.get("fitness_function", "xor")
    
    print(f"Starting job {job_id}: {population_size} pop, {generations} gens")
    
    # Update status to running
    r.hset(f"job:{job_id}", "status", "running")
    
    try:
        # Run evolution
        best_individual, best_fitness = None, 0.0
        
        for progress, fitness, individual in evolve(population_size, generations):
            best_individual = individual
            best_fitness = fitness
            
            # Update progress in Redis
            r.hset(f"job:{job_id}", "progress", progress)
            r.hset(f"job:{job_id}", "best_fitness", fitness)
        
        # Job complete!
        r.hset(f"job:{job_id}", "status", "completed")
        r.hset(f"job:{job_id}", "best_fitness", best_fitness)
        r.hset(f"job:{job_id}", "best_individual", json.dumps(best_individual))
        r.hset(f"job:{job_id}", "progress", 1.0)
        
        print(f"Job {job_id} complete! Best fitness: {best_fitness:.4f}")
        
    except Exception as e:
        # Handle errors
        r.hset(f"job:{job_id}", "status", "failed")
        r.hset(f"job:{job_id}", "error", str(e))
        print(f"Job {job_id} failed: {e}")

def main():
    r = get_redis_connection()
    print("Worker started, waiting for jobs...")
    
    while True:
        # Block and wait for a job (BRPOP gives the newest job)
        result = r.brpop("job_queue", timeout=5)
        
        if result:
            _, job_id = result
            print(f"Received job: {job_id}")
            process_job(job_id)
        else:
            # No job in queue, just keep alive
            pass

if __name__ == "__main__":
    main()