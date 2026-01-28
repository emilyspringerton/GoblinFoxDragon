#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include "protocol.h"

// --- TUNING ---
#define GRAVITY_FLOAT 0.025f 
#define GRAVITY_DROP 0.075f  
#define JUMP_FORCE 0.95f     
#define MAX_SPEED 0.95f      
#define FRICTION 0.15f      
#define ACCEL 0.6f          
#define STOP_SPEED 0.1f     
#define SLIDE_FRICTION 0.01f 
#define CROUCH_SPEED 0.35f  

// --- BUGGY TUNING (Phase 486) ---
#define BUGGY_MAX_SPEED 2.5f    // Fast!
#define BUGGY_ACCEL 0.08f       // Slow build up (Momentum)
#define BUGGY_FRICTION 0.03f    // Very slippery (Drift)
#define BUGGY_GRAVITY 0.15f     // Heavy

#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define HEAD_SIZE 1.94f     
#define HEAD_OFFSET 2.42f   
#define MELEE_RANGE_SQ 250.0f 

void evolve_bot(PlayerState *loser, PlayerState *winner);
PlayerState* get_best_bot();

typedef struct { float x, y, z, w, h, d; } Box;

// --- THE PHI-FORTRESS (Procedural) ---
// (Keeping the fractal map from Phase 483)
// Re-generating for brevity/consistency
static Box map_geo[] = {
    {0, -2, 0, 3000, 4, 3000}, // Floor
    {0, 30, 0, 40, 60, 40},    // Spire
    {0, 62, 0, 60, 4, 60},     // Deck
    {0, 10, 0, 60, 20, 60},    // Base
    {0, 5, 200, 100, 10, 100}, // N Steppes
    {0, 15, 250, 80, 30, 80},
    {0, 25, 300, 60, 50, 60},
    {0, 40, 350, 40, 80, 40},
    {300, 20, 0, 20, 2, 20},   // E Islands
    {350, 30, 50, 20, 2, 20},
    {400, 40, 0, 20, 2, 20},
    {350, 50, -50, 20, 2, 20},
    {300, 60, 0, 10, 1, 10},
    {-300, 2, 0, 100, 4, 200}, // W Ramps
    {-320, 6, 0, 100, 4, 200},
    {-340, 10, 0, 100, 4, 200},
    {-360, 14, 0, 100, 4, 200},
    {-400, 30, 0, 50, 60, 50},
    {150, 62, 0, 300, 2, 10}   // Bridge
};
static int map_count = sizeof(map_geo) / sizeof(Box);

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }

static inline float angle_diff(float a, float b) {
    float d = a - b;
    while (d < -180) d += 360;
    while (d > 180) d -= 360;
    return d;
}

int check_hit_location(float ox, float oy, float oz, float dx, float dy, float dz, PlayerState *target) {
    if (!target->active) return 0;
    float tx = target->x; float tz = target->z;
    
    // Buggy Hitbox (Larger)
    float h_size = target->in_vehicle ? 4.0f : HEAD_SIZE;
    float h_off = target->in_vehicle ? 2.0f : HEAD_OFFSET;
    
    float head_y = target->y + h_off; 
    float vx = tx - ox, vy = head_y - oy, vz = tz - oz;
    float t = vx*dx + vy*dy + vz*dz;
    if (t > 0) {
        float cx = ox + dx*t, cy = oy + dy*t, cz = oz + dz*t;
        float dist_sq = (tx-cx)*(tx-cx) + (head_y-cy)*(head_y-cy) + (tz-cz)*(tz-cz);
        if (dist_sq < (h_size*h_size)) return 2; 
    }
    // Body Hitbox
    float body_y = target->y + 2.0f;
    vx = tx - ox; vy = body_y - oy; vz = tz - oz;
    t = vx*dx + vy*dy + vz*dz;
    if (t > 0) {
        float cx = ox + dx*t, cy = oy + dy*t, cz = oz + dz*t;
        float dist_sq = (tx-cx)*(tx-cx) + (body_y-cy)*(body_y-cy) + (tz-cz)*(tz-cz);
        if (dist_sq < 7.2f) return 1; 
    }
    return 0;
}

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    
    float drop = 0;
    
    if (p->in_vehicle) {
        // DRIFT PHYSICS: Low friction, but high mass feeling
        drop = speed * BUGGY_FRICTION;
    } 
    else if (p->on_ground) {
        if (p->crouching) {
            if (speed > 0.75f) drop = speed * SLIDE_FRICTION; 
            else drop = speed * (FRICTION * 3.0f); 
        } else {
            float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
            drop = control * FRICTION; 
        }
    }
    
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vx *= newspeed; p->vz *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    
    // --- BUGGY PHYSICS ---
    if (p->in_vehicle) {
        // Halo Steering: Forward/Back only follows Camera Look
        // Strafing is disabled (wish_z ignored effectively by caller or here)
        
        // Only allow acceleration if we are somewhat grounded (or mid-air control is low)
        // For fun: Full air control for sick jumps
        
        float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
        float add_speed = wish_speed - current_speed;
        if (add_speed <= 0) return;
        
        float acc_speed = accel * BUGGY_MAX_SPEED;
        if (acc_speed > add_speed) acc_speed = add_speed;
        
        p->vx += acc_speed * wish_x; 
        p->vz += acc_speed * wish_z;
        return;
    }

    // --- FOOT PHYSICS ---
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (p->crouching && speed > 0.75f && p->on_ground) return;
    if (p->crouching && p->on_ground && speed < 0.75f && wish_speed > CROUCH_SPEED) wish_speed = CROUCH_SPEED;

    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * MAX_SPEED; 
    if (acc_speed > add_speed) acc_speed = add_speed;
    
    p->vx += acc_speed * wish_x; 
    p->vz += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = p->in_vehicle ? 3.0f : PLAYER_WIDTH; // Wide car
    float ph = p->in_vehicle ? 3.0f : (p->crouching ? (PLAYER_HEIGHT / 2.0f) : PLAYER_HEIGHT);
    
    p->on_ground = 0;
    if (p->y < 0) { p->y = 0; p->vy = 0; p->on_ground = 1; }
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->x + pw > b.x - b.w/2 && p->x - pw < b.x + b.w/2 &&
            p->z + pw > b.z - b.d/2 && p->z - pw < b.z + b.d/2) {
            if (p->y < b.y + b.h/2 && p->y + ph > b.y - b.h/2) {
                float prev_y = p->y - p->vy;
                if (prev_y >= b.y + b.h/2) {
                    p->y = b.y + b.h/2; p->vy = 0; p->on_ground = 1;
                } else {
                    float dx = p->x - b.x; float dz = p->z - b.z;
                    float w = (b.w > 0.1f) ? b.w : 1.0f;
                    float d = (b.d > 0.1f) ? b.d : 1.0f;
                    if (fabs(dx)/w > fabs(dz)/d) { 
                        p->vx = 0; 
                        p->x = (dx > 0) ? b.x + b.w/2 + pw : b.x - b.w/2 - pw;
                    } else { 
                        p->vz = 0; 
                        p->z = (dz > 0) ? b.z + b.d/2 + pw : b.z - b.d/2 - pw;
                    }
                }
            }
        }
    }
}

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1;
    p->state = STATE_ALIVE;
    p->health = 100;
    p->shield = 100;
    p->respawn_time = 0;
    p->in_vehicle = 0; // Reset vehicle
    
    if (rand()%2 == 0) { p->x = 0; p->z = 0; p->y = 80; } 
    else { float ang = phys_rand_f() * 6.28f; p->x = sinf(ang) * 500; p->z = cosf(ang) * 500; p->y = 20; }
    
    p->current_weapon = WPN_MAGNUM;
    for(int i=0; i<MAX_WEAPONS; i++) p->ammo[i] = WPN_STATS[i].ammo_max;
    
    if (p->is_bot) {
        PlayerState *winner = get_best_bot();
        if (winner && winner != p) evolve_bot(p, winner);
    }
}

void update_weapons(PlayerState *p, PlayerState *targets, int shoot, int reload) {
    // No shooting in vehicle (for now)
    if (p->in_vehicle) return; 

    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;

    int w = p->current_weapon;
    if (reload && p->reload_timer == 0 && w != WPN_KNIFE) {
        if (p->ammo[w] < WPN_STATS[w].ammo_max) {
            if (p->ammo[w] > 0) p->reload_timer = RELOAD_TIME_TACTICAL; 
            else p->reload_timer = RELOAD_TIME_FULL; 
        }
    }
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (w != WPN_KNIFE && p->ammo[w] <= 0) p->reload_timer = RELOAD_TIME_FULL; 
        else {
            p->is_shooting = 5; p->recoil_anim = 1.0f;
            p->attack_cooldown = WPN_STATS[w].rof;
            if (w != WPN_KNIFE) p->ammo[w]--;
            
            float r = -p->yaw * 0.0174533f;
            float rp = p->pitch * 0.0174533f;
            float dx = sinf(r) * cosf(rp);
            float dy = sinf(rp);
            float dz = -cosf(r) * cosf(rp);
            
            if (WPN_STATS[w].spr > 0) {
                dx += phys_rand_f() * WPN_STATS[w].spr;
                dy += phys_rand_f() * WPN_STATS[w].spr;
                dz += phys_rand_f() * WPN_STATS[w].spr;
            }

            for(int i=0; i<MAX_CLIENTS; i++) {
                if (p == &targets[i]) continue;
                if (!targets[i].active) continue;
                if (w == WPN_KNIFE) {
                    float kx = p->x - targets[i].x; float ky = p->y - targets[i].y; float kz = p->z - targets[i].z;
                    if ((kx*kx + ky*ky + kz*kz) > MELEE_RANGE_SQ + 22.0f ) continue; 
                }
                int hit_type = check_hit_location(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, &targets[i]);
                if (hit_type > 0) {
                    printf("🔫 HIT! Dmg: %d on Target %d\n", WPN_STATS[w].dmg, i);
                    int damage = WPN_STATS[w].dmg;
                    p->accumulated_reward += 10.0f;
                    targets[i].shield_regen_timer = SHIELD_REGEN_DELAY;
                    if (hit_type == 2 && targets[i].shield <= 0) { damage *= 3; p->hit_feedback = 20; } else { p->hit_feedback = 10; }
                    if (targets[i].shield > 0) {
                        if (targets[i].shield >= damage) { targets[i].shield -= damage; damage = 0; } 
                        else { damage -= targets[i].shield; targets[i].shield = 0; }
                    }
                    targets[i].health -= damage;
                    if(targets[i].health <= 0) {
                        p->kills++; targets[i].deaths++; 
                        p->accumulated_reward += 1000.0f;
                        phys_respawn(&targets[i], 0);
                    }
                }
            }
        }
    }
}

void phys_store_history(ServerState *server, int client_id, unsigned int now) {
    if (client_id < 0 || client_id >= MAX_CLIENTS) return;
    int slot = (now / 16) % LAG_HISTORY; 
    server->history[client_id][slot].active = 1;
    server->history[client_id][slot].timestamp = now;
    server->history[client_id][slot].x = server->players[client_id].x;
    server->history[client_id][slot].y = server->players[client_id].y;
    server->history[client_id][slot].z = server->players[client_id].z;
}

int phys_resolve_rewind(ServerState *server, int client_id, unsigned int target_time, float *out_pos) {
    LagRecord *hist = server->history[client_id];
    for(int i=0; i<LAG_HISTORY; i++) {
        if (!hist[i].active) continue;
        if (hist[i].timestamp == target_time) { 
            out_pos[0] = hist[i].x; out_pos[1] = hist[i].y; out_pos[2] = hist[i].z;
            return 1;
        }
    }
    return 0; 
}
#endif
