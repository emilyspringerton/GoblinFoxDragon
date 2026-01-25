#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include <math.h>
#include <string.h>
#include <stdlib.h>
#include "../common/protocol.h"
#include "../common/physics.h"

#define JUMP_COOLDOWN 15

ServerState local_state;

void local_init() {
    memset(&local_state, 0, sizeof(ServerState));
    
    // Player
    local_state.players[0].active = 1;
    local_state.players[0].y = 10.0f;
    local_state.players[0].health = 100;
    local_state.players[0].current_weapon = WPN_MAGNUM;
    for(int i=0; i<MAX_WEAPONS; i++) local_state.players[0].ammo[i] = WPN_STATS[i].ammo_max;

    // Bot
    local_state.players[1].active = 1;
    local_state.players[1].x = 5.0f; local_state.players[1].y = 0.0f; local_state.players[1].z = 5.0f;
    local_state.players[1].health = 100;
}

void local_update(float fwd, float strafe, float yaw, float pitch, int shoot, int weapon, int jump, int crouch, int reload) {
    local_state.server_tick++;
    PlayerState *p = &local_state.players[0];

    p->yaw = yaw;
    p->pitch = pitch;
    if (weapon != -1) p->current_weapon = weapon;

    if (p->jump_timer > 0) p->jump_timer--;

    apply_friction(p);

    float rad = -p->yaw * 0.0174533f;
    float fwd_x = sinf(rad); float fwd_z = -cosf(rad);
    float right_x = cosf(rad); float right_z = sinf(rad);

    float wish_x = (fwd_x * fwd) + (right_x * strafe);
    float wish_z = (fwd_z * fwd) + (right_z * strafe);
    
    float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_len > 0) { wish_x /= wish_len; wish_z /= wish_len; }

    float target_speed = crouch ? MAX_SPEED * 0.5f : MAX_SPEED;

    if (p->on_ground) {
        accelerate(p, wish_x, wish_z, target_speed, ACCEL);
        if (jump && p->jump_timer == 0) { 
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
    
    // FIRE WEAPONS (Pass the whole player array so we can hit bots)
    update_weapons(p, local_state.players, shoot, reload);

    // Bot Logic
    for(int i=1; i<MAX_CLIENTS; i++) {
        PlayerState *bot = &local_state.players[i];
        if (!bot->active) continue;
        // Dumb AI: Strafe left/right
        bot->vx = sinf(local_state.server_tick * 0.05f) * 0.2f;
        bot->vy -= GRAVITY;
        bot->x += bot->vx; bot->y += bot->vy;
        resolve_collision(bot);
    }
}
#endif
