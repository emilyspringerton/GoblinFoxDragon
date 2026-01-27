#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "protocol.h"
#include "physics.h"
#include <string.h>

ServerState local_state;

// Track previous jump state to nerf auto-hops
int was_holding_jump = 0;

void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, void *server_context, unsigned int cmd_time);
void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time);
void local_init_match(int num_players, int mode);

void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
    PlayerState *me = &players[bot_idx];
    if (me->state == STATE_DEAD) { *out_buttons = 0; return; }

    int target_idx = -1;
    float min_dist = 9999.0f;

    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (i == bot_idx) continue;
        if (!players[i].active) continue;
        if (players[i].state == STATE_DEAD) continue;
        
        float dx = players[i].x - me->x;
        float dz = players[i].z - me->z;
        float dist = sqrtf(dx*dx + dz*dz);
        if (dist < min_dist) { min_dist = dist; target_idx = i; }
    }

    if (target_idx != -1) {
        PlayerState *t = &players[target_idx];
        float dx = t->x - me->x;
        float dz = t->z - me->z;
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        float diff = angle_diff(target_yaw, *out_yaw);
        if (diff > 10.0f) diff = 10.0f;
        if (diff < -10.0f) diff = -10.0f;
        *out_yaw += diff;
        
        *out_buttons |= BTN_ATTACK;
        
        if (min_dist > 8.0f) *out_fwd = 1.0f; 
        else if (min_dist < 5.0f) *out_fwd = -1.0f; 
        else *out_fwd = 0.2f; 
        
        if (me->on_ground && (rand()%100 < 5)) *out_buttons |= BTN_JUMP;
        
    } else {
        *out_yaw += 2.0f;
        *out_fwd = 0.5f;
        if (me->on_ground && (rand()%100 < 2)) *out_buttons |= BTN_JUMP;
    }
}

void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time) {
    if (!p->active) return;
    if (p->state == STATE_DEAD) return;

    apply_friction(p);
    
    // --- FEATHER GRAVITY LOGIC ---
    // If Holding Jump -> Low Gravity (Float)
    // If Released Jump -> High Gravity (Drop)
    float g = (p->in_jump) ? GRAVITY_FLOAT : GRAVITY_DROP;
    
    p->vy -= g; 
    p->y += p->vy;
    
    resolve_collision(p);
    p->x += p->vx;
    p->z += p->vz;

    if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
    if (p->recoil_anim < 0) p->recoil_anim = 0;

    update_weapons(p, local_state.players, p->in_shoot > 0, p->in_reload > 0);
}

void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, void *server_context, unsigned int cmd_time) {
    
    PlayerState *p0 = &local_state.players[0];
    p0->yaw = yaw;
    p0->pitch = pitch;
    
    if (weapon_req >= 0 && weapon_req < MAX_WEAPONS) p0->current_weapon = weapon_req;
    
    float rad = yaw * 3.14159f / 180.0f;
    float wish_x = sinf(rad) * fwd + cosf(rad) * str;
    float wish_z = cosf(rad) * fwd - sinf(rad) * str;
    
    float wish_speed = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_speed > 1.0f) { wish_speed = 1.0f; wish_x/=wish_speed; wish_z/=wish_speed; }
    wish_speed *= MAX_SPEED;
    
    accelerate(p0, wish_x, wish_z, wish_speed, ACCEL);
    
    // --- JUMP LOGIC (With Nerf) ---
    // Only boost if we are NOT holding jump from the previous frame (fresh press)
    // OR if we were already on the ground (to allow rapid hops if timed well)
    // BUT user said: "if holding jump when landing -> ineligible"
    // So we check: If we just landed this frame, were we holding jump?
    
    int fresh_jump_press = (jump && !was_holding_jump);
    
    if (jump && p0->on_ground) {
        float speed = sqrtf(p0->vx*p0->vx + p0->vz*p0->vz);
        
        // SUPERGLIDE CHECK: 
        // 1. Must be sliding (Crouch + Speed)
        // 2. Must be a FRESH jump press (Cannot hold space through landing)
        if (p0->crouching && speed > 0.5f && fresh_jump_press) {
            p0->vx *= 1.5f;
            p0->vz *= 1.5f;
            // printf("SUPERGLIDE!\n");
        }
        
        p0->y += 0.1f; 
        p0->vy += JUMP_FORCE;
    }
    
    p0->in_shoot = shoot;
    p0->in_reload = reload;
    p0->crouching = crouch;
    p0->in_jump = jump; // Store for gravity logic
    
    was_holding_jump = jump; // Update history
    
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active) continue;
        
        if (i > 0 && p->active && p->state != STATE_DEAD) {
            float b_fwd=0, b_yaw=p->yaw;
            int b_btns=0;
            bot_think(i, local_state.players, &b_fwd, &b_yaw, &b_btns);
            p->yaw = b_yaw;
            float brad = b_yaw * 3.14159f / 180.0f;
            float bx = sinf(brad) * b_fwd;
            float bz = cosf(brad) * b_fwd;
            accelerate(p, bx, bz, MAX_SPEED, ACCEL);
            p->in_shoot = (b_btns & BTN_ATTACK);
            p->in_jump = (b_btns & BTN_JUMP);
            if ((b_btns & BTN_JUMP) && p->on_ground) { p->y += 0.1f; p->vy += JUMP_FORCE; }
        }
        
        update_entity(p, 0.016f, server_context, cmd_time);
    }
}

void local_init_match(int num_players, int mode) {
    memset(&local_state, 0, sizeof(ServerState));
    local_state.game_mode = mode;
    local_state.players[0].active = 1;
    phys_respawn(&local_state.players[0], 0);
    for(int i=1; i<num_players; i++) {
        local_state.players[i].active = 1;
        phys_respawn(&local_state.players[i], i*100);
    }
}
#endif
