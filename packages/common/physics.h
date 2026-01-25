#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include "protocol.h"

#define GRAVITY 0.025f
#define JUMP_POWER 0.45f
#define MAX_SPEED 0.75f
#define FRICTION 0.82f
#define ACCEL 1.5f
#define STOP_SPEED 0.1f

typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = {
    {0, -1, 0, 200, 2, 200},
    {15, 2.5, 15, 10, 5, 10},
    {-15, 1.5, -15, 10, 3, 10},
    {-20, 4.0, 5, 6, 8, 6},
    {0, 5.0, -25, 4, 10, 4},
    {10, 1.0, -10, 4, 2, 4}
};
static int map_count = 6;

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vel.x*p->vel.x + p->vel.z*p->vel.z);
    if (speed < 0.001f) { p->vel.x = 0; p->vel.z = 0; return; }
    if (p->pos.y > 0.001f) return; // Only ground

    float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
    float drop = control * FRICTION;
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vel.x *= newspeed; p->vel.z *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float current_speed = (p->vel.x * wish_x) + (p->vel.z * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * wish_speed;
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vel.x += acc_speed * wish_x; p->vel.z += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = 0.6f; float ph = p->crouching ? 2.0f : 4.0f; 
    p->on_ground = 0;
    if (p->pos.y < 0) { p->pos.y = 0; p->vel.y = 0; p->on_ground = 1; }

    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->pos.x + pw > b.x - b.w/2 && p->pos.x - pw < b.x + b.w/2 &&
            p->pos.z + pw > b.z - b.d/2 && p->pos.z - pw < b.z + b.d/2) {
            
            if (p->pos.y < b.y + b.h/2 && p->pos.y + ph > b.y - b.h/2) {
                float prev_y = p->pos.y - p->vel.y;
                if (prev_y >= b.y + b.h/2) {
                    p->pos.y = b.y + b.h/2; p->vel.y = 0; p->on_ground = 1;
                } else {
                    float dx = p->pos.x - b.x; float dz = p->pos.z - b.z;
                    if (fabs(dx) > fabs(dz)) { 
                        p->vel.x = 0; p->pos.x = (dx > 0) ? b.x + b.w/2 + pw : b.x - b.w/2 - pw;
                    } else { 
                        p->vel.z = 0; p->pos.z = (dz > 0) ? b.z + b.d/2 + pw : b.z - b.d/2 - pw;
                    }
                }
            }
        }
    }
}

void update_weapons(PlayerState *p, int shoot, int reload) {
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
        } else {
            p->attack_cooldown = 10; 
        }
    }
}
#endif
