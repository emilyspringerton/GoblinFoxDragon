#ifndef NEURAL_NET_H
#define NEURAL_NET_H

#include <math.h>
#include <string.h>
#include "brain_weights.h" // The file you just exported

// Simple matrix multiplication + bias + activation
// out = activation( (weights * in) + bias )
// in_size = columns, out_size = rows
void dense_layer(const float* in, const float* w, const float* b, float* out, int in_size, int out_size, int act_type) {
    for (int i = 0; i < out_size; i++) {
        float sum = b[i];
        for (int j = 0; j < in_size; j++) {
            // w is flattened [rows * cols]
            // Access: w[row * cols + col]
            sum += in[j] * w[i * in_size + j];
        }
        
        // Activation Function
        if (act_type == 1) { // ReLU
            out[i] = (sum > 0.0f) ? sum : 0.0f;
        } else if (act_type == 2) { // Tanh
            out[i] = tanhf(sum);
        } else { // Linear / None
            out[i] = sum;
        }
    }
}

// Forward Pass: 8 Inputs -> 256 -> 128 -> 4 Outputs
// Must match the PyTorch architecture exactly!
void bot_brain_forward(float* input_vec, float* output_vec) {
    float h1[256];
    float h2[128];
    
    // Layer 1: Input(8) -> H1(256) [ReLU]
    dense_layer(input_vec, l1_w, l1_b, h1, 8, 256, 1);
    
    // Layer 2: H1(256) -> H2(128) [ReLU]
    dense_layer(h1, l2_w, l2_b, h2, 256, 128, 1);
    
    // Layer 3: H2(128) -> Out(4) [Tanh]
    // Output: [Fwd, Strafe, YawRate, ShootTrigger]
    dense_layer(h2, l3_w, l3_b, output_vec, 128, 4, 2);
}

#endif
