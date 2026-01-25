#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include <math.h>
#include <string.h>
#include <stdlib.h>
#include "../common/protocol.h"
#include "../common/physics.h"

// THE SINGLE TRUTH
ServerState state;

void spawn_player(int id, float x, float y, float z) {
    memset(&state.players[id], 0, sizeof(PlayerState));
    state.players[id].active = 1;
    state.players[id].x = x;
    state.players[id].y = y;
    state.players[id].z = z;
    state.players[id].health = 100;
    state.players[id].current_weapon = WPN_AR; // Bots love ARs
    for(int i=0; i<MAX_WEAPONS; i++) state.players[id].ammo[i] = WPN_STATS[i].ammo_max;
}

void local_init() {
    memset(&state, 0, sizeof(ServerState));
    
    // Player 0 (You)
    spawn_player(0, 0, 10, 0);
    state.players[0].current_weapon = WPN_MAGNUM;

    // Player 1 (Dummy)
    spawn_player(1, 5, 10, 5);
}

// Raycast for shooting (Simple version)
int check_hit(float ox, float oy, float oz, float dx, float dy, float dz, int shooter_id) {
    for(int i=0; i<MAX_CLIENTS; i++) {
        if (i == shooter_id || !state.players[i].active) continue;
        
        PlayerState *t = &state.players[i];
        // Sphere intersection approx
        float tx = t->x - ox; float ty = (t->y + 2.0f) - oy; float tz = t->z - oz;
        float dist = sqrtf(tx*tx + ty*ty + tz*tz);
        
        // Dot product to check alignment
        float dot = (tx*dx + ty*dy + tz*dz) / dist;
        
        if (dot > 0.98f && dist < 100.0f) { // Hit!
            return i;
        }
    }
    return -1;
}

void update_combat(PlayerState *p, int id, int shoot, int reload) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;
    if (p->hit_feedback > 0) p->hit_feedback--;

    int w = p->current_weapon;
    
    // Reload
    if (reload && p->reload_timer == 0 && p->ammo[w] < WPN_STATS[w].ammo_max) {
        p->reload_timer = RELOAD_TIME;
    }
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;

    // Shoot
    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (p->ammo[w] > 0 || w == WPN_KNIFE) {
            p->is_shooting = 5; 
            p->recoil_anim = 1.0f;
            p->attack_cooldown = WPN_STATS[w].rof;
            if (w != WPN_KNIFE) p->ammo[w]--;

            // Hitscan
            float r = -p->yaw * 0.0174533f;
            float rp = p->pitch * 0.0174533f;
            float dx = sinf(r) * cosf(rp);
            float dy = sinf(rp);
            float dz = -cosf(r) * cosf(rp);

            int hit = check_hit(p->x, p->y + 3.0f, p->z, dx, dy, dz, id);
            if (hit != -1) {
                p->hit_feedback = 10; // Show hitmarker frame
                state.players[hit].health -= WPN_STATS[w].dmg;
                
                // Death Logic
                if (state.players[hit].health <= 0) {
                    // Respawn target
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

    // Loop through ALL entities (0 = Player, 1+ = Bots)
    for (int i=0; i<MAX_CLIENTS; i++) {
        if (!state.players[i].active) continue;
        
        PlayerState *p = &state.players[i];
        
        // --- 1. INPUT GATHERING ---
        float i_fwd = 0, i_str = 0;
        int i_jump = 0, i_crouch = 0, i_shoot = 0, i_reload = 0;

        if (i == 0) {
            // Local Player Input
            p->yaw = yaw; p->pitch = pitch;
            if (weapon != -1) p->current_weapon = weapon;
            i_fwd = fwd; i_str = strafe;
            i_jump = jump; i_crouch = crouch;
            i_shoot = shoot; i_reload = reload;
        } else {
            // Dummy AI Input (Simple Strafe)
            i_str = sinf(state.tick * 0.05f + i) * 0.8f; // Move left/right
            p->yaw += 1.0f; // Spin slowly
        }

        // --- 2. PHYSICS ENGINE (Shared) ---
        apply_friction(p);

        float rad = -p->yaw * 0.0174533f;
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

        if (p->y < -50.0f) { p->x=0; p->y=10; p->z=0; p->vx=0; p->vy=0; p->vz=0; } // Void safety

        resolve_collision(p);

        if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
        
        // --- 3. COMBAT ---
        update_combat(p, i, i_shoot, i_reload);
    }
}
#endif
