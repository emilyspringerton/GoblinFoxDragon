#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include "protocol.h"

#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define MELEE_RANGE_SQ 250.0f 
#define SHIELD_REGEN_DELAY 180

typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = { {0.00,-2.00,0.00,1500.00,4.00,1500.00}, {0.00,30.00,0.00,40.00,60.00,40.00}, {0.00,62.00,0.00,60.00,4.00,60.00} };
static int map_count = 3;

// --- MATH HELPERS ---
static inline float angle_diff(float a, float b) {
    float d = a - b;
    while (d < -180) d += 360;
    while (d > 180) d -= 360;
    return d;
}

// --- COLLISION ---
void resolve_collision(PlayerState *p) {
    p->on_ground = 0;
    if (p->y < 0) { p->y = 0; p->vy = 0; p->on_ground = 1; }
    for(int i=0; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->x + PLAYER_WIDTH > b.x - b.w/2 && p->x - PLAYER_WIDTH < b.x + b.w/2 &&
            p->z + PLAYER_WIDTH > b.z - b.d/2 && p->z - PLAYER_WIDTH < b.z + b.d/2) {
            if (p->y < b.y + b.h/2 && p->y + PLAYER_HEIGHT > b.y - b.h/2) {
                float prev_y = p->y - p->vy;
                if (prev_y >= b.y + b.h/2) { p->y = b.y + b.h/2; p->vy = 0; p->on_ground = 1; }
            }
        }
    }
}

// --- WEAPONS ---
void update_weapons(PlayerState *p, PlayerState *targets, int shoot, int reload) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;
    
    int w = p->current_weapon;
    if (reload && p->reload_timer == 0 && w != 0) p->reload_timer = 60;
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;
    
    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        p->is_shooting = 5; p->recoil_anim = 1.0f;
        p->attack_cooldown = WPN_STATS[w].rof;
        if (w != 0) p->ammo[w]--;
        // Hitscan logic would follow...
    }
}

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1; p->health = 100; p->shield = 100;
    p->x = 0; p->z = 0; p->y = 20;
    p->current_weapon = 1;
    for(int i=0; i<5; i++) p->ammo[i] = WPN_STATS[i].ammo_max;
}

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    p->vx *= 0.85f; p->vz *= 0.85f;
}

void accelerate(PlayerState *p, float wx, float wz, float ws, float acc) {
    float current_speed = (p->vx * wx) + (p->vz * wz);
    float add_speed = ws - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = acc * 0.95f; 
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vx += acc_speed * wx; p->vz += acc_speed * wz;
}
#endif
