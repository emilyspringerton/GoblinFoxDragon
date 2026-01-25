#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include "protocol.h"

// --- TURBO TUNING ---
#define GRAVITY 0.025f
#define JUMP_FORCE 0.55f
#define MAX_SPEED 0.75f
#define FRICTION 0.82f
#define ACCEL 1.5f
#define STOP_SPEED 0.1f
#define MAX_AIR_SPEED 0.1f
#define EYE_HEIGHT 4.0f

typedef struct { float x, y, z, w, h, d; } Box;

// MAP GEOMETRY
static Box map_geo[] = {
    // 0: Floor (900x300)
    {0, -1, 0, 900, 2, 300},
    
    // --- CENTRAL ZIGGURAT (Indices 1-3) ---
    {0, 2.0, 0, 20, 4, 20},       
    {0, 5.0, 0, 10, 2, 10},       
    {0, 8.0, 0, 4, 4, 4},
    
    // --- EAST CANYON (Indices 4-8) ---
    {200, 10, 80, 200, 20, 20},   
    {200, 10, -80, 200, 20, 20},  
    {350, 4, 0, 10, 8, 10},       
    {250, 2, 40, 4, 4, 4},        
    {150, 2, -40, 4, 4, 4},       

    // --- WEST CANYON (Indices 9-13) ---
    {-200, 10, 80, 200, 20, 20},  
    {-200, 10, -80, 200, 20, 20}, 
    {-350, 4, 0, 10, 8, 10},      
    {-250, 2, -40, 4, 4, 4},      
    {-150, 2, 40, 4, 4, 4},       

    // --- PARKOUR BRIDGES (Index 14) ---
    {0, 12, 0, 800, 1, 2},        

    // --- CONTAINMENT WALLS (Indices 15-18) ---
    {0, 25, 200, 1200, 50, 100},   // North Wall (Z+) - Index 15
    {0, 25, -200, 1200, 50, 100},  // South Wall (Z-) - Index 16
    {550, 25, 0, 200, 50, 600},    // East Wall (X+)  - Index 17 (Was skipped!)
    {-550, 25, 0, 200, 50, 600}    // West Wall (X-)  - Index 18 (Was skipped!)
};

// FIX: COUNT IS 19, NOT 17
static int map_count = 19; 

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }

int check_hit(float ox, float oy, float oz, float dx, float dy, float dz, PlayerState *target) {
    if (!target->active) return 0;
    float tx = target->x; float ty = target->y + 2.0f; float tz = target->z;
    float vx = tx - ox; float vy = ty - oy; float vz = tz - oz;
    float t = vx*dx + vy*dy + vz*dz;
    if (t < 0) return 0;
    float cx = ox + dx*t; float cy = oy + dy*t; float cz = oz + dz*t;
    float dist_sq = (tx-cx)*(tx-cx) + (ty-cy)*(ty-cy) + (tz-cz)*(tz-cz);
    return (dist_sq < 2.5f);
}

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    if (!p->on_ground) return; 
    float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
    float drop = control * FRICTION;
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vx *= newspeed; p->vz *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * wish_speed;
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vx += acc_speed * wish_x; p->vz += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = 0.6f; float ph = p->crouching ? 2.0f : 4.0f; 
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
                    // Push out to nearest side
                    if (fabs(dx)/b.w > fabs(dz)/b.d) { 
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

void update_weapons(PlayerState *p, PlayerState *targets, int shoot, int reload) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;

    int w = p->current_weapon;
    if (reload && p->reload_timer == 0 && w != WPN_KNIFE) {
        if (p->ammo[w] < WPN_STATS[w].ammo_max) p->reload_timer = RELOAD_TIME;
    }
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (w == WPN_KNIFE || p->ammo[w] > 0) {
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

                if (check_hit(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, &targets[i])) {
                    targets[i].health -= WPN_STATS[w].dmg;
                    p->hit_feedback = 10;
                    if(targets[i].health <= 0) {
                        p->kills++;
                        targets[i].health = 100;
                        targets[i].x = (rand()%800)-400; 
                        targets[i].z = (rand()%200)-100; 
                        targets[i].y = 10;
                    }
                }
            }
        } else {
            p->attack_cooldown = 10; 
        }
    }
}
#endif
