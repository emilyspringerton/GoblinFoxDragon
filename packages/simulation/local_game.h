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
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        *out_yaw = target_yaw;
        *out_buttons |= BTN_ATTACK;
        
        // Dynamic Movement (Don't freeze!)
        if (min_dist > 8.0f) *out_fwd = 1.0f; // Chase
        else if (min_dist < 5.0f) *out_fwd = -1.0f; // Back up
        else *out_fwd = 0.2f; // Creep forward to avoid friction stop
        
    } else {
        // Patrol
        *out_yaw += 2.0f;
        *out_fwd = 0.5f;
    }
}

// --- UPDATE ENTITY (Rewired to use physics.h) ---
void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time) {
    if (!p->active) return;
    if (p->state == STATE_DEAD) return;

    // Apply Friction
    apply_friction(p);
    
    // Apply Gravity
    p->vy -= GRAVITY; p->y += p->vy;
    if (p->y < 0) p->y = 0; // Simple floor clamp if map doesn't catch it
    
    // Resolve Collisions
    resolve_collision(p);
    
    // Update Position
    p->x += p->vx;
    p->z += p->vz;

    // Weapons
    
    // Animation Decay
    if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
    if (p->recoil_anim < 0) p->recoil_anim = 0;
    
    update_weapons(p, local_state.players, p->in_shoot > 0, p->in_reload > 0);
}

// --- LOCAL UPDATE LOOP ---
void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, void *server_context, unsigned int cmd_time) {
    
    // 1. Update Local Player (Index 0)
    PlayerState *p0 = &local_state.players[0];
    p0->yaw = yaw;
    p0->pitch = pitch;
    
    // Weapon Switch
    if (weapon_req >= 0 && weapon_req < MAX_WEAPONS) p0->current_weapon = weapon_req;
    
    // Movement Physics (Quake Style)
    float rad = yaw * 3.14159f / 180.0f;
    float wish_x = sinf(rad) * fwd + cosf(rad) * str;
    float wish_z = cosf(rad) * fwd - sinf(rad) * str;
    
    // Normalize wish dir
    float wish_speed = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_speed > 1.0f) {
        wish_speed = 1.0f;
        wish_x /= wish_speed;
        wish_z /= wish_speed;
    }
    wish_speed *= MAX_SPEED;
    
    accelerate(p0, wish_x, wish_z, wish_speed, ACCEL);
    
    if (jump && p0->on_ground) {
        p0->y += 0.1f; // Lift off ground
        p0->vy += JUMP_FORCE;
    }
    p0->y += p0->vy;
    
    p0->in_shoot = shoot;
    p0->in_reload = reload;
    p0->crouching = crouch;
    
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
            // Bot Physics
            float brad = b_yaw * 3.14159f / 180.0f;
            float bx = sinf(brad) * b_fwd;
            float bz = cosf(brad) * b_fwd;
            accelerate(p, bx, bz, MAX_SPEED, ACCEL);
            
            p->in_shoot = (b_btns & BTN_ATTACK);
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
