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

#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define HEAD_SIZE 1.94f     
#define HEAD_OFFSET 2.42f   
#define MELEE_RANGE_SQ 250.0f 

typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = {
    {0.00, -2.00, 0.00, 1500.00, 4.00, 1500.00},
    {0.00, 30.00, 0.00, 40.00, 60.00, 40.00},
    {0.00, 62.00, 0.00, 60.00, 4.00, 60.00},
    {526.69, 32.01, -384.38, 22.98, 27.00, 35.12},
    {509.84, 14.36, -537.03, 20.82, 28.71, 37.77},
    {-346.11, 37.50, -415.62, 12.34, 24.39, 39.66},
    {464.05, 11.00, -362.24, 13.90, 22.01, 10.07},
    {8.55, 4.95, 579.26, 12.30, 9.91, 19.07},
    {-574.14, 42.92, -311.29, 10.49, 26.63, 38.53},
    {145.40, 12.01, -590.30, 15.34, 24.01, 21.10}
};
static int map_count = sizeof(map_geo) / sizeof(Box);

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }
static inline float angle_diff(float a, float b) {
    float d = a - b; while (d < -180) d += 360; while (d > 180) d -= 360; return d;
}

// --- RICOCHET MATH ---
void reflect_vector(float *vx, float *vy, float *vz, float nx, float ny, float nz) {
    float dot = (*vx * nx) + (*vy * ny) + (*vz * nz);
    *vx = *vx - 2.0f * dot * nx;
    *vy = *vy - 2.0f * dot * ny;
    *vz = *vz - 2.0f * dot * nz;
}

// Returns 1 if collision occurred, sets normal in nx/ny/nz
int trace_map(float x1, float y1, float z1, float x2, float y2, float z2, float *out_x, float *out_y, float *out_z, float *nx, float *ny, float *nz) {
    // Simplified ray-box intersection for simulation
    // This is a placeholder for the full sweep logic, checking basic containment for ricochet testing
    // In a real engine, we'd use Slab method or AABB sweep.
    // For now, we reuse resolve_collision logic concepts but applied to a point.
    
    // Checking endpoints against boxes
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (x2 > b.x - b.w/2 && x2 < b.x + b.w/2 &&
            z2 > b.z - b.d/2 && z2 < b.z + b.d/2 &&
            y2 > b.y - b.h/2 && y2 < b.y + b.h/2) {
            
            // Determine normal based on previous position (where did we come from?)
            float dx = x1 - b.x; float dz = z1 - b.z;
            float w = b.w; float d = b.d;
            
            if (fabs(dx)/w > fabs(dz)/d) {
                *nx = (dx > 0) ? 1 : -1; *ny = 0; *nz = 0;
                *out_x = (dx > 0) ? b.x + b.w/2 + 0.1f : b.x - b.w/2 - 0.1f;
                *out_y = y2; *out_z = z2;
            } else {
                *nx = 0; *ny = 0; *nz = (dz > 0) ? 1 : -1;
                *out_x = x2; *out_y = y2;
                *out_z = (dz > 0) ? b.z + b.d/2 + 0.1f : b.z - b.d/2 - 0.1f;
            }
            return 1;
        }
    }
    // Floor check
    if (y2 < 0) {
        *nx = 0; *ny = 1; *nz = 0;
        *out_x = x2; *out_y = 0.1f; *out_z = z2;
        return 1;
    }
    return 0;
}

void resolve_collision(PlayerState *p) {
    float pw = PLAYER_WIDTH;
    float ph = (p->crouching ? (PLAYER_HEIGHT / 2.0f) : PLAYER_HEIGHT);
    p->on_ground = 0;
    if (p->y < 0) { p->y = 0; p->vy = 0; p->on_ground = 1; }
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->x + pw > b.x - b.w/2 && p->x - pw < b.x + b.w/2 &&
            p->z + pw > b.z - b.d/2 && p->z - pw < b.z + b.d/2) {
            if (p->y < b.y + b.h/2 && p->y + ph > b.y - b.h/2) {
                float prev_y = p->y - p->vy;
                if (prev_y >= b.y + b.h/2) { p->y = b.y + b.h/2; p->vy = 0; p->on_ground = 1; }
                else {
                    float dx = p->x - b.x; float dz = p->z - b.z;
                    float w = (b.w > 0.1f) ? b.w : 1.0f;
                    float d = (b.d > 0.1f) ? b.d : 1.0f;
                    if (fabs(dx)/w > fabs(dz)/d) { 
                        p->vx = 0; p->x = (dx > 0) ? b.x + b.w/2 + pw : b.x - b.w/2 - pw;
                    } else { 
                        p->vz = 0; p->z = (dz > 0) ? b.z + b.d/2 + pw : b.z - b.d/2 - pw;
                    }
                }
            }
        }
    }
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

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1; p->health = 100; p->shield = 100;
    p->x = 0; p->z = 0; p->y = 20; p->current_weapon = 1;
    p->storm_charges = 0; p->ability_cooldown = 0;
    for(int i=0; i<5; i++) p->ammo[i] = WPN_STATS[i].ammo_max;
}
#endif
