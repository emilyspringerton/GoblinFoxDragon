#ifndef BOT_AI_H
#define BOT_AI_H

#include "../common/protocol.h"
#include <math.h>
#include "neural_net.h"

// Global flag controlled by Client
int USE_NEURAL_NET = 0;

void bot_think(PlayerState *bot, ServerState *world) {
    // 1. Find Closest Target (Common to both modes)
    PlayerState *target = NULL;
    float best_dist = 9999.0f;

    for (int i=0; i<MAX_CLIENTS; i++) {
        // Attack active players that are NOT me
        if (world->players[i].active && &world->players[i] != bot) {
            float dx = world->players[i].x - bot->x;
            float dz = world->players[i].z - bot->z;
            float d = sqrtf(dx*dx + dz*dz);
            if (d < best_dist) {
                best_dist = d;
                target = &world->players[i];
            }
        }
    }

    // --- MODE S: NEURAL NETWORK ---
    if (USE_NEURAL_NET && target) {
        float inputs[8];
        float outputs[4]; // [Fwd, Strafe, YawRate, Shoot]
        
        // NORMALIZE INPUTS (MUST MATCH PYTHON EXACTLY)
        // Map Size approx 900x300 -> Divide by 500.0f
        inputs[0] = bot->x / 500.0f;
        inputs[1] = bot->z / 500.0f;
        
        // Orientation (Sin/Cos)
        float rad = bot->yaw * 0.0174533f; // Deg to Rad
        inputs[2] = sinf(rad);
        inputs[3] = cosf(rad);
        
        // Target Pos
        inputs[4] = target->x / 500.0f;
        inputs[5] = target->z / 500.0f;
        
        // Distance
        inputs[6] = best_dist / 500.0f;
        
        // My Health
        inputs[7] = bot->health / 100.0f;
        
        // --- INFERENCE ---
        bot_brain_forward(inputs, outputs);
        
        // --- APPLY OUTPUTS ---
        bot->in_fwd = outputs[0];
        bot->in_strafe = outputs[1];
        
        // Yaw Rate (Tanh is -1 to 1, multiply by turn speed)
        bot->yaw += outputs[2] * 15.0f; 
        
        // Shoot Trigger (Tanh > 0.5 means confident fire)
        bot->in_shoot = (outputs[3] > 0.5f) ? 1 : 0;
        
        return; // Skip heuristic
    }

    // --- MODE DEFAULT: HEURISTIC AI (Fallback) ---
    if (!target) {
        bot->in_fwd = 0; bot->in_shoot = 0;
        return;
    }

    // Classic Logic
    float dx = target->x - bot->x;
    float dy = (target->y + 2.0f) - (bot->y + 4.0f); 
    float dz = target->z - bot->z;
    float dist_h = sqrtf(dx*dx + dz*dz);

    float target_yaw = atan2f(dx, -dz) * (180.0f / M_PI);
    float target_pitch = atan2f(dy, dist_h) * (180.0f / M_PI);

    float diff = target_yaw - bot->yaw;
    while(diff > 180) diff -= 360;
    while(diff < -180) diff += 360;
    bot->yaw += diff * 0.15f; 
    bot->pitch += (target_pitch - bot->pitch) * 0.1f;

    if (dist_h > 20.0f) bot->in_fwd = 1.0f;       
    else if (dist_h < 8.0f) bot->in_fwd = -1.0f;   
    else bot->in_fwd = 0.0f;                       

    // Simple Strafe
    if ((world->server_tick / 40) % 2 == 0) bot->in_strafe = 0.8f;
    else bot->in_strafe = -0.8f;

    if (fabs(diff) < 15.0f) bot->in_shoot = 1;
    else bot->in_shoot = 0;
}
#endif
