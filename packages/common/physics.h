#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include "protocol.h"

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
#define MELEE_RANGE_SQ 250.0f 

typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = {
    {0.00, -2.00, 0.00, 1500.00, 4.00, 1500.00},
    {0.00, 30.00, 0.00, 40.00, 60.00, 40.00},
    {0.00, 62.00, 0.00, 60.00, 4.00, 60.00},
    {526.69, 32.01, -384.38, 22.98, 27.00, 35.12},
    {509.84, 14.36, -537.03, 20.82, 28.71, 37.77},
    {-346.11, 37.50, -415.62, 12.34, 24.39, 39.66}
    // ... Additional map data re-hydrated from Build 178
};
static int map_count = sizeof(map_geo) / sizeof(Box);

void resolve_collision(PlayerState *p) {
    float pw = PLAYER_WIDTH;
    float ph = (p->crouching ? (PLAYER_HEIGHT / 2.0f) : PLAYER_HEIGHT);
    p->on_ground = 0;
    if (p->y < 0) { p->y = 0; p->vy = 0; p->on_ground = 1; }
    for(int i=0; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->x + pw > b.x - b.w/2 && p->x - pw < b.x + b.w/2 &&
            p->z + pw > b.z - b.d/2 && p->z - pw < b.z + b.d/2) {
            if (p->y < b.y + b.h/2 && p->y + ph > b.y - b.h/2) {
                float prev_y = p->y - p->vy;
                if (prev_y >= b.y + b.h/2) { p->y = b.y + b.h/2; p->vy = 0; p->on_ground = 1; }
            }
        }
    }
}

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1; p->health = 100; p->shield = 100;
    p->x = 0; p->z = 0; p->y = 20;
    p->current_weapon = 1; // Magnum
}

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    float drop = speed * FRICTION;
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vx *= newspeed; p->vz *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * MAX_SPEED; 
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vx += acc_speed * wish_x; p->vz += acc_speed * wish_z;
}
#endif
