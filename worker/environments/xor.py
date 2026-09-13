"""
XOR fitness environment.

A 2-4-1 feedforward neural network is evolved to solve the XOR truth table.
Fitness is 1 / (1 + total_squared_error), so perfect = 1.0.
"""
import math
from typing import List


class NeuralNetwork:
    """A simple 2-4-1 feedforward network."""
    
    def __init__(self, weights: List[float]):
        if len(weights) != 17:
            raise ValueError(f"Expected 17 weights, got {len(weights)}")
        self.weights = weights
    
    def forward(self, inputs: List[float]) -> List[float]:
        # Layer 1: 2 inputs -> 4 hidden neurons
        hidden = [
            math.tanh(self.weights[0]*inputs[0] + self.weights[1]*inputs[1] + self.weights[8]),
            math.tanh(self.weights[2]*inputs[0] + self.weights[3]*inputs[1] + self.weights[9]),
            math.tanh(self.weights[4]*inputs[0] + self.weights[5]*inputs[1] + self.weights[10]),
            math.tanh(self.weights[6]*inputs[0] + self.weights[7]*inputs[1] + self.weights[11]),
        ]
        # Layer 2: 4 hidden -> 1 output
        output = math.tanh(
            self.weights[12]*hidden[0] +
            self.weights[13]*hidden[1] +
            self.weights[14]*hidden[2] +
            self.weights[15]*hidden[3] +
            self.weights[16]
        )
        return [output]


# The XOR truth table
XOR_CASES = [
    ([0.0, 0.0], 0.0),
    ([0.0, 1.0], 1.0),
    ([1.0, 0.0], 1.0),
    ([1.0, 1.0], 0.0),
]


def fitness(weights: List[float]) -> float:
    """Higher is better. Perfect = 1.0."""
    try:
        nn = NeuralNetwork(weights)
    except ValueError:
        return 0.0
    
    total_error = 0.0
    for inputs, expected in XOR_CASES:
        output = nn.forward(inputs)[0]
        total_error += (output - expected) ** 2
    
    return 1.0 / (1.0 + total_error)


def genome_size() -> int:
    """Number of parameters in the network."""
    return 17


def describe() -> dict:
    """Metadata about this environment for the API/dashboard."""
    return {
        "name": "xor",
        "description": "Evolve a 2-4-1 neural network to solve the XOR truth table",
        "genome_size": 17,
        "max_fitness": 1.0,
    }