#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include <math.h>
#include <string.h>
#include <stdlib.h>
#include "../common/protocol.h"
#include "../common/physics.h"

// THE UNIFIED STATE (Global to this compilation unit)
ServerState state;

void local_init() {
    memset(&state, 0, sizeof(ServerState));
    
    // Player 0 (You)
    state.players[0].active = 1;
    state.players[0].x = 0; state.players[0].y = 10; state.players[0].z = 0;
    state.players[0].health = 100;
    state.players[0].current_weapon = WPN_MAGNUM;
    for(int i=0; i<MAX_WEAPONS; i++) state.players[0].ammo[i] = WPN_STATS[i].ammo_max;

    // Player 1 (Dummy)
    state.players[1].active = 1;
    state.players[1].x = 5.0f; state.players[1].y = 0.0f; state.players[1].z = 5.0f;
    state.players[1].health = 100;
}

// Raycast for shooting
int check_hit(float ox, float oy, float oz, float dx, float dy, float dz, int shooter_id) {
    for(int i=0; i<MAX_CLIENTS; i++) {
        if (i == shooter_id || !state.players[i].active) continue;
        
        PlayerState *t = &state.players[i];
        float tx = t->x - ox; float ty = (t->y + 2.0f) - oy; float tz = t->z - oz;
        float dist = sqrtf(tx*tx + ty*ty + tz*tz);
        float dot = (tx*dx + ty*dy + tz*dz) / dist;
        
        // Simple sphere hit detection
        if (dot > 0.95f && dist < 100.0f) return i;
    }
    return -1;
}

void update_combat(PlayerState *p, int id, int shoot, int reload) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;
    if (p->hit_feedback > 0) p->hit_feedback--;

    int w = p->current_weapon;
    
    if (reload && p->reload_timer == 0 && p->ammo[w] < WPN_STATS[w].ammo_max) p->reload_timer = RELOAD_TIME;
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (p->ammo[w] > 0 || w == WPN_KNIFE) {
            p->is_shooting = 5; 
            p->recoil_anim = 1.0f;
            p->attack_cooldown = WPN_STATS[w].rof;
            if (w != WPN_KNIFE) p->ammo[w]--;

            float r = p->yaw * 0.0174533f;
            float rp = p->pitch * 0.0174533f;
            float dx = sinf(r) * cosf(rp);
            float dy = sinf(rp);
            float dz = -cosf(r) * cosf(rp);

            int hit = check_hit(p->x, p->y + 3.0f, p->z, dx, dy, dz, id);
            if (hit != -1) {
                p->hit_feedback = 10;
                state.players[hit].health -= WPN_STATS[w].dmg;
                if (state.players[hit].health <= 0) {
                    state.players[hit].x = (rand()%20) - 10;
                    state.players[hit].z = (rand()%20) - 10;
                    state.players[hit].y = 10;
                    state.players[hit].health = 100;
                }
            }
        }
    }
}

void local_update(float fwd, float strafe, float yaw, float pitch, int shoot, int weapon, int jump, int crouch, int reload) {
    state.tick++;

    for (int i=0; i<MAX_CLIENTS; i++) {
        if (!state.players[i].active) continue;
        PlayerState *p = &state.players[i];
        
        float i_fwd = 0, i_str = 0;
        int i_jump = 0, i_crouch = 0, i_shoot = 0, i_reload = 0;

        if (i == 0) {
            p->yaw = yaw; p->pitch = pitch;
            if (weapon != -1) p->current_weapon = weapon;
            i_fwd = fwd; i_str = strafe;
            i_jump = jump; i_crouch = crouch; i_shoot = shoot; i_reload = reload;
        } else {
            i_str = sinf(state.tick * 0.05f + i) * 0.8f; 
            p->yaw += 1.0f; 
        }

        apply_friction(p);

        // Standard Math (Yaw = Right)
        float rad = p->yaw * 0.0174533f;
        float fwd_x = sinf(rad); float fwd_z = -cosf(rad);
        float right_x = cosf(rad); float right_z = sinf(rad);

        float wish_x = (fwd_x * i_fwd) + (right_x * i_str);
        float wish_z = (fwd_z * i_fwd) + (right_z * i_str);
        
        float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
        if (wish_len > 0) { wish_x /= wish_len; wish_z /= wish_len; }

        float target_speed = i_crouch ? MAX_SPEED * 0.5f : MAX_SPEED;

        if (p->on_ground) {
            accelerate(p, wish_x, wish_z, target_speed, ACCEL);
            if (i_jump) { p->vy = JUMP_FORCE; p->on_ground = 0; }
        } else {
            accelerate(p, wish_x, wish_z, target_speed * MAX_AIR_SPEED, ACCEL);
        }

        p->vy -= GRAVITY;
        p->x += p->vx; p->z += p->vz; p->y += p->vy;

        if (p->y < -50.0f) { p->x=0; p->y=10; p->z=0; p->vx=0; p->vy=0; p->vz=0; }

        resolve_collision(p);
        if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
        update_combat(p, i, i_shoot, i_reload);
    }
}
#endif
