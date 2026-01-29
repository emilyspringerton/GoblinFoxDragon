#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "../common/protocol.h"
#include "../common/physics.h"
#include <string.h>

ServerState local_state;

// --- BOT AI ---
void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
    PlayerState *me = &players[bot_idx];
    *out_fwd = 0.5f;
    *out_yaw += 1.0f;
    if (rand()%100 == 0) *out_buttons |= BTN_JUMP;
}

// --- PROJECTILE SIMULATION (With Ricochet) ---
void update_projectiles() {
    for (int i=0; i<MAX_PROJECTILES; i++) {
        Projectile *p = &local_state.projectiles[i];
        if (!p->active) continue;

        float next_x = p->x + p->vx;
        float next_y = p->y + p->vy;
        float next_z = p->z + p->vz;

        float hit_x, hit_y, hit_z, nx, ny, nz;
        if (trace_map(p->x, p->y, p->z, next_x, next_y, next_z, &hit_x, &hit_y, &hit_z, &nx, &ny, &nz)) {
            if (p->bounces_left > 0) {
                // RICOCHET
                reflect_vector(&p->vx, &p->vy, &p->vz, nx, ny, nz);
                p->x = hit_x; p->y = hit_y; p->z = hit_z;
                p->bounces_left--;
                printf("BOUNCE! Left: %d\n", p->bounces_left);
            } else {
                p->active = 0; // Destroy
            }
        } else {
            p->x = next_x; p->y = next_y; p->z = next_z;
        }
        
        if (p->x > 4000 || p->x < -4000 || p->z > 4000 || p->z < -4000 || p->y > 2000) p->active = 0;
    }
}

void fire_projectile(PlayerState *p, int damage, int bounces, float speed_mult) {
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if (!local_state.projectiles[i].active) {
            Projectile *proj = &local_state.projectiles[i];
            proj->active = 1;
            proj->owner_id = p->id;
            proj->damage = damage;
            proj->bounces_left = bounces;
            
            float rad_y = p->yaw * 0.01745f;
            float rad_p = p->pitch * 0.01745f;
            float speed = 4.0f * speed_mult; // Base projectile speed
            
            proj->vx = sinf(rad_y) * cosf(rad_p) * speed;
            proj->vy = sinf(rad_p) * speed;
            proj->vz = -cosf(rad_y) * cosf(rad_p) * speed;
            
            proj->x = p->x; proj->y = p->y + EYE_HEIGHT; proj->z = p->z;
            break;
        }
    }
}

// --- WEAPONS SYSTEM ---
void update_weapons(PlayerState *p, int shoot, int reload, int ability_press) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->ability_cooldown > 0) p->ability_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;

    // STORM ARROW ACTIVATION
    if (ability_press && p->ability_cooldown == 0 && p->storm_charges == 0) {
        p->storm_charges = 5;
        p->ability_cooldown = 480; // 8 seconds cooldown (60Hz)
        printf("STORM ARROWS ACTIVE!\n");
    }

    int w = p->current_weapon;
    int rof = WPN_STATS[w].rof;
    int damage = WPN_STATS[w].dmg;
    
    // STORM MODE OVERRIDES
    if (p->storm_charges > 0 && w == WPN_SNIPER) { // Hanzo Bow
        rof = 8; // Rapid Fire
        damage = (int)(damage * 0.7f); 
    }

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (p->storm_charges > 0 && w == WPN_SNIPER) {
            fire_projectile(p, damage, 1, 1.5f); // 1 Bounce, Fast
            p->storm_charges--;
            p->attack_cooldown = rof;
            p->recoil_anim = 0.5f;
        } else {
            // Standard Fire
            p->is_shooting = 5; 
            p->recoil_anim = 1.0f;
            p->attack_cooldown = rof;
            if (w != WPN_KNIFE) p->ammo[w]--;
            // Hitscan logic normally here...
        }
    }
}

void local_update(float fwd, float str, float yaw, float pitch, int shoot, int wpn, int jump, int crouch, int reload, int ability, void* ctx, int time) {
    PlayerState *p = &local_state.players[0];
    p->yaw = yaw; p->pitch = pitch;
    if (wpn >= 0) p->current_weapon = wpn;
    
    float rad = yaw * 0.01745f;
    float wx = sinf(rad) * fwd + cosf(rad) * str;
    float wz = -cosf(rad) * fwd + sinf(rad) * str; // Fixed Z axis sign for proper movement
    
    accelerate(p, wx, wz, MAX_SPEED, ACCEL);
    
    if (jump && p->on_ground) { p->vy = JUMP_FORCE; p->on_ground = 0; }
    p->crouching = crouch;
    
    update_entity(p, 0.016f, NULL, 0);
    update_weapons(p, shoot, reload, ability);
    update_projectiles();
}

void local_init_match(int num, int mode) {
    memset(&local_state, 0, sizeof(ServerState));
    local_state.players[0].active = 1;
    phys_respawn(&local_state.players[0], 0);
    // Give player a Sniper (Bow) for testing
    local_state.players[0].current_weapon = WPN_SNIPER; 
    local_state.players[0].ammo[WPN_SNIPER] = 100;
}
#endif
