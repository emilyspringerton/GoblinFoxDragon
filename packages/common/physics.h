
#ifndef PHYSICS_H
#define PHYSICS_H

#include <math.h>
#include <stdlib.h> // For rand()
#include "protocol.h"

// --- CONSTANTS ---
#define GRAVITY 0.5f
#define MOVE_SPEED 0.5f
#define JUMP_FORCE 2.0f
#define EYE_HEIGHT 1.5f

// Golden Ratio Math (Phase 319)
#define PHI         1.61803398875f
#define GOLDEN_GAIN 0.38196601125f

float golden_smooth(float current, float target) {
    float error = target - current;
    return current + (error * GOLDEN_GAIN);
}

float golden_noise(unsigned int seed, float offset) {
    double val = (seed * PHI) + offset;
    return (float)(val - (long)val);
}

// --- FORWARD DECLARATIONS ---
int phys_resolve_rewind(ServerState *server, int client_id, unsigned int target_time, float *out_pos);

// --- LOGIC ---
int check_hit_location(float x, float y, float z, float dx, float dy, float dz, PlayerState *target) {
    // Simple Cylinder/Box Hitbox
    // Target center is target->x, target->z
    // Height is 0 to 2.0
    
    // This is a placeholder for ray-cylinder intersection
    // For now, simple distance check along the ray? No, that's inaccurate.
    // Let's assume a simplified "Cone of Fire" or distance check for Alpha.
    
    float tx = target->x - x;
    float tz = target->z - z;
    float dist = sqrtf(tx*tx + tz*tz);
    
    if (dist < 1.0f) return 1; // Hit
    return 0;
}

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1;
    p->state = STATE_ALIVE;
    p->health = MAX_HEALTH;
    p->respawn_time = 0;
    
    // Golden Spawn
    float n_x = golden_noise(p->deaths + now, 0.1f);
    float n_z = golden_noise(p->deaths + now, 0.9f);
    
    p->x = (n_x * 40.0f) - 20.0f;
    p->z = (n_z * 40.0f) - 20.0f;
    p->y = 0;
}

void phys_damage(PlayerState *victim, PlayerState *attacker, int damage, unsigned int now) {
    if (victim->state == STATE_DEAD) return;
    
    victim->health -= damage;
    victim->last_hit_time = now;
    
    if (victim->health <= 0) {
        victim->health = 0;
        victim->state = STATE_DEAD;
        victim->deaths++;
        victim->respawn_time = now + RESPAWN_DELAY;
        if (attacker && attacker != victim) attacker->frags++;
    }
}

void update_weapons(PlayerState *p, PlayerState *targets, int shoot, int reload, void *server_ptr, unsigned int cmd_time) {
    if (p->in_shoot > 0) p->in_shoot--;
    
    if (shoot && p->in_shoot == 0) {
        p->in_shoot = 10; // Cooldown
        
        // Raycast Direction
        float rad_yaw = p->yaw * 3.14159f / 180.0f;
        float dx = sinf(rad_yaw);
        float dz = cosf(rad_yaw);
        float dy = 0; // Pitch ignored for simple 2D aim
        
        // Check Hits
        for(int i=0; i<MAX_CLIENTS; i++) {
            if (&targets[i] == p) continue;
            if (!targets[i].active) continue;
            if (targets[i].state == STATE_DEAD) continue;
            
            // TIME MACHINE REWIND
            float original_pos[3];
            int rewound = 0;
            
            if (server_ptr && cmd_time > 0) {
                ServerState *s = (ServerState*)server_ptr;
                original_pos[0] = targets[i].x;
                original_pos[1] = targets[i].y;
                original_pos[2] = targets[i].z;
                
                float past[3];
                if (phys_resolve_rewind(s, i, cmd_time, past)) {
                    targets[i].x = past[0];
                    targets[i].y = past[1];
                    targets[i].z = past[2];
                    rewound = 1;
                }
            }
            
            // Check Hit
            if (check_hit_location(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, &targets[i])) {
                if (server_ptr) {
                    phys_damage(&targets[i], p, 25, cmd_time);
                }
            }
            
            // Restore
            if (rewound) {
                targets[i].x = original_pos[0];
                targets[i].y = original_pos[1];
                targets[i].z = original_pos[2];
            }
        }
    }
}

// --- HISTORY STORE ---
void phys_store_history(ServerState *server, int client_id, unsigned int now) {
    if (client_id < 0 || client_id >= MAX_CLIENTS) return;
    int slot = (now / 16) % LAG_HISTORY; // Simple tick map
    
    server->history[client_id][slot].active = 1;
    server->history[client_id][slot].timestamp = now;
    server->history[client_id][slot].x = server->players[client_id].x;
    server->history[client_id][slot].y = server->players[client_id].y;
    server->history[client_id][slot].z = server->players[client_id].z;
}

int phys_resolve_rewind(ServerState *server, int client_id, unsigned int target_time, float *out_pos) {
    LagRecord *hist = server->history[client_id];
    // Simple linear scan for match (Optimized version in real engine)
    for(int i=0; i<LAG_HISTORY; i++) {
        if (!hist[i].active) continue;
        if (hist[i].timestamp == target_time) { // Exact match for simplicity
            out_pos[0] = hist[i].x; out_pos[1] = hist[i].y; out_pos[2] = hist[i].z;
            return 1;
        }
    }
    return 0; // Interpolation logic omitted for brevity in reset, assume 60Hz lock
}

#endif
