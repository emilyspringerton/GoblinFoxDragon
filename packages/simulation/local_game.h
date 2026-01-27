#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "protocol.h"
#include "physics.h"
#include <string.h>

ServerState local_state;

// --- BOT AI (Phase 418 Hunter + Phase 421 Anti-Stuck) ---
void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
    PlayerState *me = &players[bot_idx];
    if (me->state == STATE_DEAD) {
        *out_buttons = 0;
        return;
    }

    int target_idx = -1;
    float min_dist = 9999.0f;

    // Find Enemy
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (i == bot_idx) continue;
        if (!players[i].active) continue;
        if (players[i].state == STATE_DEAD) continue;
        
        float dx = players[i].x - me->x;
        float dz = players[i].z - me->z;
        float dist = sqrtf(dx*dx + dz*dz);
        
        if (dist < min_dist) {
            min_dist = dist;
            target_idx = i;
        }
    }

    if (target_idx != -1) {
        // Aim and Attack
        PlayerState *t = &players[target_idx];
        float dx = t->x - me->x;
        float dz = t->z - me->z;
        
        // Aim Smoothing
        PlayerState *t = &players[target_idx];
        float dx = t->x - me->x;
        float dz = t->z - me->z;
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        // Smoothly rotate towards target (10 degrees per tick max)
        float diff = angle_diff(target_yaw, *out_yaw);
        if (diff > 10.0f) diff = 10.0f;
        if (diff < -10.0f) diff = -10.0f;
        *out_yaw += diff;

    }
}

#endif
