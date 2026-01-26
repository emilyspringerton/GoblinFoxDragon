#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "protocol.h"
#include "physics.h"
#include <string.h>

ServerState local_state;

// --- BOT AI (Internal Definition to avoid Conflicts) ---

// --- BOT AI (Restored Phase 418) ---
void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
    PlayerState *me = &players[bot_idx];
    if (me->state == STATE_DEAD) {
        *out_buttons = 0;
        return;
    }

    int target_idx = -1;
    float min_dist = 9999.0f;

    // Find Closest Enemy
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
        // ENGAGE
        PlayerState *t = &players[target_idx];
        float dx = t->x - me->x;
        float dz = t->z - me->z;
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        *out_yaw = target_yaw; // Aimbot
        
        // Move to engage range (keep distance 5-10 units)
        if (min_dist > 8.0f) *out_fwd = 1.0f;
        else if (min_dist < 4.0f) *out_fwd = -1.0f; // Back up
        else *out_fwd = 0.0f;
        
        *out_buttons |= BTN_ATTACK;
    } else {
        // PATROL
        *out_yaw += 2.0f;
        *out_fwd = 0.5f;
    }
}

void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time) {
    if (!p->active) return;
    if (p->state == STATE_DEAD) return;

    // Physics
    p->y -= GRAVITY * dt;
    if (p->y < 0) p->y = 0; // Floor collision

    // Weapons
    update_weapons(p, local_state.players, p->in_shoot > 0, 0, server_context, cmd_time);
}

// --- LOCAL UPDATE LOOP ---
void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, void *server_context, unsigned int cmd_time) {
    
    // 1. Update Local Player (Index 0)
    PlayerState *p0 = &local_state.players[0];
    p0->yaw = yaw;
    p0->pitch = pitch;
    
    // Simple Movement
    float rad = yaw * 3.14159f / 180.0f;
    p0->vx = sinf(rad) * fwd * MOVE_SPEED; p0->x += p0->vx;
    p0->vz = cosf(rad) * fwd * MOVE_SPEED; p0->z += p0->vz;
    
    if (shoot) p0->in_shoot = 1; // Trigger
    
    // 2. Update Bots / All Entities
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active) continue;
        
        // Bot AI Hook
        if (i > 0 && p->active && p->state != STATE_DEAD) {
            float b_fwd=0, b_yaw=p->yaw;
            int b_btns=0;
            
            bot_think(i, local_state.players, &b_fwd, &b_yaw, &b_btns);
            
            p->yaw = b_yaw;
            float brad = b_yaw * 3.14159f / 180.0f;
            p->x += sinf(brad) * b_fwd * MOVE_SPEED * 0.5f;
            p->z += cosf(brad) * b_fwd * MOVE_SPEED * 0.5f;
            
            if (b_btns & BTN_ATTACK) p->in_shoot = 1;
        }
        
        // Physics & Resolve
        update_entity(p, 0.016f, server_context, cmd_time);
    }
}

void local_init_match(int num_players, int mode) {
    memset(&local_state, 0, sizeof(ServerState));
    local_state.game_mode = mode;
    
    // Spawn Player
    local_state.players[0].active = 1;
    phys_respawn(&local_state.players[0], 0);
    
    // Spawn Bots
    for(int i=1; i<num_players; i++) {
        local_state.players[i].active = 1;
        phys_respawn(&local_state.players[i], i*100);
    }
}

#endif
