import math
import json

# The evolved weights from your job (13 weights)
weights = [-1.023156425561667, -1.1486914170647506, -1.4456714573138707, 
           -1.2917790169580925, 0.9111159037341193, 0.051287020650064186, 
           -0.222303717816153, -0.7232353954771374, 0.24883758248520338, 
           -0.9374106512379963, 0.09326726178899802, 0.31548098720797446, 
           0.19767198449801304]

class NeuralNetwork:
    def __init__(self, weights):
        self.weights = weights
    
    def forward(self, inputs):
        # Layer 1: 2 inputs -> 4 hidden neurons, using the worker's 8 weights
        hidden = [
            math.tanh(self.weights[0]*inputs[0] + self.weights[1]*inputs[1] + self.weights[4]),
            math.tanh(self.weights[2]*inputs[0] + self.weights[3]*inputs[1] + self.weights[5]),
            math.tanh(self.weights[4]*inputs[0] + self.weights[5]*inputs[1] + self.weights[6]),
            math.tanh(self.weights[6]*inputs[0] + self.weights[7]*inputs[1] + self.weights[7])
        ]
        
        # Layer 2: 4 hidden -> 1 output, using 4 output weights and 1 bias
        output = math.tanh(
            self.weights[8]*hidden[0] +
            self.weights[9]*hidden[1] +
            self.weights[10]*hidden[2] +
            self.weights[11]*hidden[3] +
            self.weights[12]
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
    print(f"XOR({inputs[0]}, {inputs[1]}) = {output:.4f} -> {rounded} (expected: {expected})")