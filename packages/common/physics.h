#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include "protocol.h"

// --- QUAKE / HALO HYBRID TUNING ---
#define GRAVITY 0.02f
#define JUMP_FORCE 0.50f
#define MAX_SPEED 0.6f
#define MAX_AIR_SPEED 0.1f
#define FRICTION 0.82f     // Slidey
#define ACCEL 1.0f         // Snappy acceleration
#define STOP_SPEED 0.1f

// --- MAP GEOMETRY ---
typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = {
    {0, -1, 0, 200, 2, 200},      // Floor
    {15, 2.5, 15, 10, 5, 10},     // Red Box
    {-15, 1.5, -15, 10, 3, 10},   // Blue Box
    {-20, 4.0, 5, 6, 8, 6},       // Pillar
    {0, 5.0, -25, 4, 10, 4},      // Sniper Perch
    {10, 1.0, -10, 4, 2, 4}       // Step
};
static int map_count = 6;

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    
    if (!p->on_ground) return; // No air friction

    float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
    float drop = control * FRICTION;
    
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;

    p->vx *= newspeed;
    p->vz *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    
    float acc_speed = accel * wish_speed;
    if (acc_speed > add_speed) acc_speed = add_speed;
    
    p->vx += acc_speed * wish_x;
    p->vz += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = 0.6f; // Fat hitbox
    float ph = p->crouching ? 2.0f : 4.0f; 

    p->on_ground = 0;
    
    // Floor Safety
    if (p->y < 0) { p->y = 0; p->vy = 0; p->on_ground = 1; }

    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->x + pw > b.x - b.w/2 && p->x - pw < b.x + b.w/2 &&
            p->z + pw > b.z - b.d/2 && p->z - pw < b.z + b.d/2) {
            
            if (p->y < b.y + b.h/2 && p->y + ph > b.y - b.h/2) {
                // Landing on top
                float prev_y = p->y - p->vy;
                if (prev_y >= b.y + b.h/2) {
                    p->y = b.y + b.h/2;
                    p->vy = 0;
                    p->on_ground = 1;
                } else {
                    // Wall Slide logic
                    float dx = p->x - b.x;
                    float dz = p->z - b.z;
                    if (fabs(dx) > fabs(dz)) { 
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
#endif
