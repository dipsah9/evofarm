import math
# Layout: 8 input weights, 4 hidden biases, 4 output weights, 1 output bias.
# These deterministic weights provide a known-good XOR reference model.
weights = [
    5.0, 5.0, 5.0, 5.0, 0.0, 0.0, 0.0, 0.0,
    -2.5, -7.5, 0.0, 0.0,
    5.0, -5.0, 0.0, 0.0, -2.5,
]

class NeuralNetwork:
    def __init__(self, weights):
        if len(weights) != 17:
            raise ValueError(f"Expected 17 weights, got {len(weights)}")
        self.weights = weights
    
    def forward(self, inputs):
        # Layer 1: 2 inputs -> 4 hidden neurons.
        hidden = [
            math.tanh(self.weights[0] * inputs[0] + self.weights[1] * inputs[1] + self.weights[8]),
            math.tanh(self.weights[2] * inputs[0] + self.weights[3] * inputs[1] + self.weights[9]),
            math.tanh(self.weights[4] * inputs[0] + self.weights[5] * inputs[1] + self.weights[10]),
            math.tanh(self.weights[6] * inputs[0] + self.weights[7] * inputs[1] + self.weights[11]),
        ]
        
        # Layer 2: 4 hidden -> 1 output.
        output = math.tanh(
            self.weights[12] * hidden[0] +
            self.weights[13] * hidden[1] +
            self.weights[14] * hidden[2] +
            self.weights[15] * hidden[3] +
            self.weights[16]
        )
        return output

# Test XOR
nn = NeuralNetwork(weights)
test_cases = [
    ([0, 0], 0),
    ([0, 1], 1),
    ([1, 0], 1),
    ([1, 1], 0)
]

print("XOR Test Results:")
print("-" * 40)
for inputs, expected in test_cases:
    output = nn.forward(inputs)
    rounded = 1 if output > 0.5 else 0
    assert rounded == expected, f"XOR{inputs}: expected {expected}, got {rounded}"
    print(f"XOR({inputs[0]}, {inputs[1]}) = {output:.4f} -> {rounded} (expected: {expected})")

print("XOR test passed")