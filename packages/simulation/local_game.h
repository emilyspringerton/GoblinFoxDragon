#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include <math.h>
#include <string.h>
#include <stdlib.h>
#include "../common/protocol.h"
#include "../common/physics.h"
#include "bot_ai.h"

#define JUMP_COOLDOWN 15

ServerState local_state;

void local_init_match(int bot_count) {
    memset(&local_state, 0, sizeof(ServerState));
    
    // Player 0 (Human)
    local_state.players[0].id = 0;
    local_state.players[0].active = 1;
    local_state.players[0].is_bot = 0;
    local_state.players[0].y = 10.0f;
    local_state.players[0].health = 100;
    local_state.players[0].current_weapon = WPN_MAGNUM;
    for(int i=0; i<MAX_WEAPONS; i++) local_state.players[0].ammo[i] = WPN_STATS[i].ammo_max;

    // Bots (Safe Spawn)
    for(int i=1; i<=bot_count; i++) {
        if (i >= MAX_CLIENTS) break;
        local_state.players[i].id = i;
        local_state.players[i].active = 1;
        local_state.players[i].is_bot = 1;
        
        // Modulo math to ensure we wrap around the map instead of going off the edge
        // Map X range: -450 to 450. Safe range: -400 to 400.
        // We calculate an offset based on ID.
        int step = i * 25; 
        int x_raw = (step % 800) - 400; // -400 to 400
        
        local_state.players[i].x = (float)x_raw;
        // Alternate sides of the canyon (Z axis +/- 50)
        local_state.players[i].z = (i % 2 == 0) ? 50.0f : -50.0f;
        local_state.players[i].y = 10.0f;
        
        local_state.players[i].health = 100;
        local_state.players[i].current_weapon = WPN_AR;
    }
}

// THE UNIFIED PHYSICS SOLVER

// --- PHASE 408: BOT ARTIFICIAL INTELLIGENCE ---

#include <math.h>

void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
    PlayerState *me = &players[bot_idx];
    if (me->state == STATE_DEAD) {
        *out_buttons = 0;
        return;
    }

    int target_idx = -1;
    float min_dist = 9999.0f;

    // 1. Find Closest Living Enemy
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (i == bot_idx) continue;
        if (!players[i].active) continue;
        if (players[i].state == STATE_DEAD) continue;
        
        // Team Check (Simple: If mode is TDM, check team)
        // For now, assume FFA (Attack everyone)
        
        float dx = players[i].x - me->x;
        float dz = players[i].z - me->z;
        float dist = sqrtf(dx*dx + dz*dz);
        
        if (dist < min_dist) {
            min_dist = dist;
            target_idx = i;
        }
    }

    // 2. Engage
    if (target_idx != -1) {
        PlayerState *t = &players[target_idx];
        float dx = t->x - me->x;
        float dz = t->z - me->z;
        
        // Calculate Yaw (Atan2 returns radians, convert to degrees)
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        // Smoothly rotate towards target (Simple P-Controller)
        // For now, snap aim (Aimbot style) because they are bots.
        *out_yaw = target_yaw;
        
        // Move towards them if far away, strafe if close
        if (min_dist > 5.0f) {
            *out_fwd = 1.0f; // Run fwd
        } else {
            *out_fwd = 0.0f; // Stand ground
        }
        
        // Shoot!
        *out_buttons |= BTN_ATTACK;
    } else {
        // No targets? Patrol.
        *out_yaw += 2.0f; // Spin
        *out_fwd = 0.5f;
    }
}

void update_entity(PlayerState *p) {
    if (!p->active) return;

    if (p->jump_timer > 0) p->jump_timer--;
    
    if (p->hit_feedback > 0) p->hit_feedback--;

    apply_friction(p);

    // 1. Convert Input
    float rad = -p->yaw * 0.0174533f;
    float fwd_x = sinf(rad); float fwd_z = -cosf(rad);
    float right_x = cosf(rad); float right_z = sinf(rad);

    float wish_x = (fwd_x * p->in_fwd) + (right_x * p->in_strafe);
    float wish_z = (fwd_z * p->in_fwd) + (right_z * p->in_strafe);
    
    // 2. Normalize
    float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_len > 1.0f) { wish_x /= wish_len; wish_z /= wish_len; wish_len = 1.0f; }

    float target_speed = p->crouching ? MAX_SPEED * 0.5f : MAX_SPEED;

    // 3. Accelerate
    if (p->on_ground) {
        accelerate(p, wish_x, wish_z, target_speed, ACCEL);
        if (p->in_jump && p->jump_timer == 0) { 
            p->vy = JUMP_FORCE; 
            p->on_ground = 0;
            p->jump_timer = JUMP_COOLDOWN;
            if (wish_len > 0) { p->vx *= 1.15f; p->vz *= 1.15f; } 
        }
    } else {
        accelerate(p, wish_x, wish_z, target_speed * 0.1f, ACCEL);
    }

    p->vy -= GRAVITY;
    p->x += p->vx; p->z += p->vz; p->y += p->vy;

    if (p->y < -50.0f) { p->x=0; p->y=10; p->z=0; p->vx=0; p->vy=0; p->vz=0; }

    resolve_collision(p);

    if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
    update_weapons(p, local_state.players, p->in_shoot, p->in_reload, server_context, cmd_time);
}

void local_update(float fwd, float strafe, float yaw, float pitch, int shoot, int weapon, int jump, int crouch, int reload) {
    local_state.server_tick++;
    
    PlayerState *p0 = &local_state.players[0];
    p0->in_fwd = fwd; p0->in_strafe = strafe;
    p0->yaw = yaw; p0->pitch = pitch;
    p0->in_shoot = shoot; p0->in_jump = jump; 
    p0->crouching = crouch; p0->in_reload = reload;
    if (weapon != -1) p0->current_weapon = weapon;
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.players[i].active && local_state.players[i].is_bot) {
            bot_think(&local_state.players[i], &local_state);
        }
    }

    for(int i=0; i<MAX_CLIENTS; i++) {
        if (local_state.players[i].active) {
            update_entity(&local_state.players[i]);
        }
    }
}
#endif
