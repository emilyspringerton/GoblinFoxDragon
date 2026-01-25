#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include <math.h>
#include <string.h>
#include <stdlib.h>
#include "../common/protocol.h"
#include "../common/physics.h"

#define MAX_PROJECTILES 64
#define EYE_HEIGHT 6.0f

typedef struct {
    int active;
    float x, y, z;
    float vx, vy, vz;
    int owner_id;
    int dmg;
    int life;
} Projectile;

ServerState state;
Projectile projectiles[MAX_PROJECTILES];

void local_init() {
    memset(&state, 0, sizeof(ServerState));
    memset(projectiles, 0, sizeof(projectiles));
    
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

// Spawn a physical bullet
void fire_projectile(float x, float y, float z, float yaw, float pitch, float speed, float spread, int dmg, int owner) {
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if (!projectiles[i].active) {
            projectiles[i].active = 1;
            projectiles[i].life = 100; // 1.5 seconds approx
            projectiles[i].owner_id = owner;
            projectiles[i].dmg = dmg;
            projectiles[i].x = x; projectiles[i].y = y; projectiles[i].z = z;
            
            // Calc Velocity
            float rad = -yaw * 0.0174533f;
            float rp = pitch * 0.0174533f;
            
            // Add jitter
            float spr_y = ((rand()%100)/100.0f - 0.5f) * spread;
            float spr_p = ((rand()%100)/100.0f - 0.5f) * spread;
            
            rad += spr_y;
            rp += spr_p;

            float dx = sinf(rad) * cosf(rp);
            float dy = sinf(rp);
            float dz = -cosf(rad) * cosf(rp);
            
            projectiles[i].vx = dx * speed;
            projectiles[i].vy = dy * speed;
            projectiles[i].vz = dz * speed;
            return;
        }
    }
}

// Raycast (Hitscan)
int check_hit(float ox, float oy, float oz, float dx, float dy, float dz, int shooter_id) {
    float best_dist = 1000.0f;
    int best_hit = -1;

    for(int i=0; i<MAX_CLIENTS; i++) {
        if (i == shooter_id || !state.players[i].active) continue;
        
        PlayerState *t = &state.players[i];
        
        // Target Center (Torso)
        float tx = t->x; 
        float ty = t->y + 3.0f; 
        float tz = t->z;

        float ox_t = tx - ox; float oy_t = ty - oy; float oz_t = tz - oz;
        float t_proj = (ox_t*dx + oy_t*dy + oz_t*dz);
        
        if (t_proj < 0) continue; 

        float cx = dx * t_proj; float cy = dy * t_proj; float cz = dz * t_proj;
        float dist_sq = (ox_t-cx)*(ox_t-cx) + (oy_t-cy)*(oy_t-cy) + (oz_t-cz)*(oz_t-cz);
        
        // Hitbox Radius ~1.5 units
        if (dist_sq < 2.25f) {
            float d = sqrtf(ox_t*ox_t + oy_t*oy_t + oz_t*oz_t);
            if (d < best_dist) { best_dist = d; best_hit = i; }
        }
    }
    return best_hit;
}

void damage_player(int id, int dmg, int shooter_id) {
    if (state.players[id].active) {
        state.players[id].health -= dmg;
        if (shooter_id == 0) state.players[0].hit_feedback = 5; // Flash Red
        
        if (state.players[id].health <= 0) {
            state.players[id].health = 100;
            state.players[id].x = (rand()%40) - 20;
            state.players[id].z = (rand()%40) - 20;
            state.players[id].y = 5;
        }
    }
}

void update_projectiles() {
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if (!projectiles[i].active) continue;
        Projectile *p = &projectiles[i];
        
        p->x += p->vx; p->y += p->vy; p->z += p->vz;
        p->life--;
        
        // Floor Collision
        if (p->y < 0) { p->active = 0; continue; }
        if (p->life <= 0) { p->active = 0; continue; }

        // Player Collision (Simple Point Check)
        for(int j=0; j<MAX_CLIENTS; j++) {
            if (!state.players[j].active || j == p->owner_id) continue;
            PlayerState *t = &state.players[j];
            
            float dx = p->x - t->x;
            float dy = p->y - (t->y + 3.0f); // Center mass
            float dz = p->z - t->z;
            
            if (dx*dx + dy*dy + dz*dz < 4.0f) { // 2.0 Radius Sphere
                damage_player(j, p->dmg, p->owner_id);
                p->active = 0;
                break;
            }
        }
    }
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

            // WEAPON TYPE LOGIC
            if (w == WPN_MAGNUM || w == WPN_SNIPER || w == WPN_KNIFE) {
                // HITSCAN
                float rad = -p->yaw * 0.0174533f;
                float rp = p->pitch * 0.0174533f;
                float dx = sinf(rad) * cosf(rp);
                float dy = sinf(rp);
                float dz = -cosf(rad) * cosf(rp);
                
                // AIM FIX: Shoot from EYE HEIGHT (Y+6.0)
                int hit = check_hit(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, id);
                if (hit != -1) damage_player(hit, WPN_STATS[w].dmg, id);
            
            } else {
                // PROJECTILE (AR / Shotgun)
                float speed = 2.0f; // Fast bullet
                if (w == WPN_AR) {
                    fire_projectile(p->x, p->y + EYE_HEIGHT, p->z, p->yaw, p->pitch, speed, 0.05f, WPN_STATS[w].dmg, id);
                } else if (w == WPN_SHOTGUN) {
                    for(int k=0; k<8; k++) {
                        fire_projectile(p->x, p->y + EYE_HEIGHT, p->z, p->yaw, p->pitch, speed, 0.15f, WPN_STATS[w].dmg, id);
                    }
                }
            }
        }
    }
}

void local_update(float fwd, float strafe, float yaw, float pitch, int shoot, int weapon, int jump, int crouch, int reload) {
    state.tick++;
    update_projectiles(); // Move bullets

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

        if (p->y < -50.0f) { p->x=0; p->y=10; p->z=0; p->vx=0; p->vy=0; p->vz=0; }

        resolve_collision(p);
        if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
        update_combat(p, i, i_shoot, i_reload);
    }
}
#endif
