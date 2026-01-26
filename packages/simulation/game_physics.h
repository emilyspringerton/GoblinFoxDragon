#ifndef GAME_PHYSICS_H
#define GAME_PHYSICS_H

#undef ADD_WALL 
#include <math.h>
#include <string.h> 
#include <stdlib.h> 
#include "../protocol/protocol.h"
#include "../map/map.h"

// CONSTANTS
#define GRAVITY 0.025f
#define JUMP_POWER 0.45f
#define MAX_SPEED 0.75f  
#define FRICTION 0.64f
#define ACCEL 1.618f       
#define RELOAD_TIME 42
#define GAME_DURATION 7200 
#define KILL_LIMIT 20

// HELPER MATH
float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }
Vec3 phys_get_aim(float yaw, float pitch, float spread) {
    float r = -yaw * (M_PI/180.0f);
    float rp = pitch * (M_PI/180.0f);
    Vec3 fwd = { sin(r)*cos(rp), sin(rp), -cos(r)*cos(rp) }; 
    if (spread > 0) { fwd.x+=phys_rand_f()*spread; fwd.y+=phys_rand_f()*spread; fwd.z+=phys_rand_f()*spread; }
    return fwd;
}
int phys_ray_hit(Vec3 origin, Vec3 dir, Vec3 target, float radius) {
    Vec3 oc = { target.x-origin.x, target.y-origin.y, target.z-origin.z };
    float t = (oc.x*dir.x + oc.y*dir.y + oc.z*dir.z);
    if (t < 0) return 0;
    Vec3 p = { origin.x + dir.x*t, origin.y + dir.y*t, origin.z + dir.z*t };
    return (powf(p.x-target.x, 2) + powf(p.y-target.y, 2) + powf(p.z-target.z, 2)) < (radius*radius);
}

// MAP INIT
void phys_init_map(GameMap *map) {
    map->wall_count = 0;
    #define ADD_WALL(x, y, z, w, h, d, cr, cg, cb) map->walls[map->wall_count++] = (Wall){x, y, z, w, h, d, cr, cg, cb}
    ADD_WALL(5, 0.25f, 5, 1.0f, 0.25f, 1.0f, 1.0f, 0.5f, 0.0f);
    ADD_WALL(6, 0.75f, 5, 1.0f, 0.75f, 1.0f, 1.0f, 0.6f, 0.0f); 
}

// MOVE LOGIC
void phys_apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vel.x*p->vel.x + p->vel.z*p->vel.z);
    if (speed < 0.001f || p->pos.y > 0.001f) return;
    float drop = ((speed < 0.1f) ? 0.1f : speed) * FRICTION;
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vel.x *= newspeed; p->vel.z *= newspeed;
}

void phys_accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float current_speed = (p->vel.x * wish_x) + (p->vel.z * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * wish_speed * 0.1f;
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vel.x += acc_speed * wish_x; p->vel.z += acc_speed * wish_z;
}

void phys_slide_move(GameMap *map, Vec3 *pos, Vec3 *vel, int is_crouching) {
    float w_sz = 0.3f; float h_sz = is_crouching ? 1.0f : 1.8f; 
    for (int i=0; i < map->wall_count; i++) {
        Wall *w = &map->walls[i];
        if (pos->x + w_sz > w->x - w->sx && pos->x - w_sz < w->x + w->sx &&
            pos->z + w_sz > w->z - w->sz && pos->z - w_sz < w->z + w->sz &&
            pos->y < w->y + w->sy && (pos->y + h_sz) > w->y - w->sy) 
        {
            float dx1 = (w->x + w->sx) - (pos->x - w_sz); float dx2 = (pos->x + w_sz) - (w->x - w->sx);
            float dz1 = (w->z + w->sz) - (pos->z - w_sz); float dz2 = (pos->z + w_sz) - (w->z - w->sz);
            float dy1 = (w->y + w->sy) - pos->y; float dy2 = (pos->y + h_sz) - (w->y - w->sy); 
            float ox = (fabs(dx1) < fabs(dx2)) ? dx1 : -dx2; float oz = (fabs(dz1) < fabs(dz2)) ? dz1 : -dz2;
            if (dy1 > 0 && dy1 < 0.5f && vel->y <= 0) { pos->y += dy1; vel->y = 0; } 
            else if (dy2 > 0 && dy2 < 0.5f && vel->y > 0) { pos->y -= dy2; vel->y = 0; }
            else { if (fabs(ox) < fabs(oz)) { pos->x += ox; vel->x = 0; } else { pos->z += oz; vel->z = 0; } }
        }
    }
}

void phys_update_player(GameMap *map, PlayerState *p, float fwd, float strafe, float yaw, float pitch, int jump, int crouch) {
    p->yaw = yaw; p->pitch = pitch;
    phys_apply_friction(p);

    float r = -yaw * (M_PI/180.0f);
    float fwd_x = sinf(r); float fwd_z = -cosf(r); 
    float right_x = cosf(r); float right_z = sinf(r); 
    float wish_x = (fwd_x * fwd) + (right_x * strafe);
    float wish_z = (fwd_z * fwd) + (right_z * strafe);
    float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_len > 0) { wish_x /= wish_len; wish_z /= wish_len; }

    float target_speed = crouch ? (MAX_SPEED * 0.5f) : MAX_SPEED;
    int grounded = (p->pos.y <= 0.001f);
    
    if (grounded) { if (wish_len > 0) phys_accelerate(p, wish_x, wish_z, target_speed, ACCEL); } 
    else { if (wish_len > 0) phys_accelerate(p, wish_x, wish_z, target_speed, ACCEL * 0.1f); }

    p->vel.y -= GRAVITY;
    if (p->pos.y <= 0) { p->pos.y = 0; p->vel.y = 0; grounded = 1; }
    if (p->pos.y < -10.0f) { p->pos.x = 0; p->pos.y = 5.0f; p->pos.z = 0; p->vel.x=0; p->vel.y=0; p->vel.z=0; }

    phys_slide_move(map, &p->pos, &p->vel, crouch);
    if (p->vel.y == 0 && p->pos.y > 0) grounded = 1; 

    if (jump && grounded) { p->vel.y = JUMP_POWER; p->pos.y += 0.05f; }

    p->pos.x += p->vel.x; p->pos.z += p->vel.z; p->pos.y += p->vel.y;
    
    if (p->reload_timer > 0) {
        p->reload_timer--;
        if (p->reload_timer == 0) p->ammo[p->current_weapon] = WPN_STATS[p->current_weapon].ammo_max;
    }
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;
    p->hit_feedback = 0; 
}
#endif

