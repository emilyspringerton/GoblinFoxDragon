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

    // Bots (Stable AR Loadout)
    for(int i=1; i<=bot_count; i++) {
        if (i >= MAX_CLIENTS) break;
        local_state.players[i].id = i;
        local_state.players[i].active = 1;
        local_state.players[i].is_bot = 1;
        float angle = (i * 6.28f) / bot_count;
        local_state.players[i].x = sinf(angle) * 15.0f;
        local_state.players[i].z = cosf(angle) * 15.0f;
        local_state.players[i].y = 10.0f;
        local_state.players[i].health = 100;
        local_state.players[i].current_weapon = WPN_AR;
    }
}

// THE UNIFIED PHYSICS SOLVER
void update_entity(PlayerState *p) {
    if (!p->active) return;

    if (p->jump_timer > 0) p->jump_timer--;
    
    // --- DECAY HIT MARKER ---
    if (p->hit_feedback > 0) p->hit_feedback--;

    apply_friction(p);

    float rad = -p->yaw * 0.0174533f;
    float fwd_x = sinf(rad); float fwd_z = -cosf(rad);
    float right_x = cosf(rad); float right_z = sinf(rad);

    float wish_x = (fwd_x * p->in_fwd) + (right_x * p->in_strafe);
    float wish_z = (fwd_z * p->in_fwd) + (right_z * p->in_strafe);
    
    float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_len > 1.0f) { wish_x /= wish_len; wish_z /= wish_len; wish_len = 1.0f; }

    float target_speed = p->crouching ? MAX_SPEED * 0.5f : MAX_SPEED;

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
    update_weapons(p, local_state.players, p->in_shoot, p->in_reload);
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
