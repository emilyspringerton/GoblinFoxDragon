#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H
#include <math.h>
#include <stdlib.h>
#include <string.h>
#include "protocol.h" 
#include "map.h"

extern ServerState local_state; 
extern int local_pid;
static GameMap local_level;

#define GRAVITY 0.02f
#define JUMP_POWER 0.42f
#define MAX_SPEED 0.8f
#define FRICTION 0.96f  // <-- VERY SLIPPERY (Was 0.82)
#define ACCEL 0.04f     // <-- SLOW ACCEL (Drifty feel)
#define RELOAD_TIME 60 

float loc_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }

Vec3 loc_get_aim(float yaw, float pitch, float spread) {
    float r = -yaw * (3.14159f/180.0f); float rp = pitch * (3.14159f/180.0f);
    Vec3 fwd = { sin(r)*cos(rp), sin(rp), -cos(r)*cos(rp) };
    if (spread > 0) { fwd.x+=loc_rand_f()*spread; fwd.y+=loc_rand_f()*spread; fwd.z+=loc_rand_f()*spread; }
    return fwd;
}

float loc_dist_sq(Vec3 a, Vec3 b) { return (a.x-b.x)*(a.x-b.x) + (a.y-b.y)*(a.y-b.y) + (a.z-b.z)*(a.z-b.z); }

int loc_ray_hit(Vec3 origin, Vec3 dir, Vec3 target, float radius) {
    Vec3 oc = { target.x-origin.x, target.y-origin.y, target.z-origin.z };
    float t = (oc.x*dir.x + oc.y*dir.y + oc.z*dir.z); if (t < 0) return 0;
    Vec3 p = { origin.x + dir.x*t, origin.y + dir.y*t, origin.z + dir.z*t };
    return loc_dist_sq(p, target) < (radius*radius);
}

void local_init() {
    map_init(&local_level);
    memset(&local_state, 0, sizeof(local_state));
    local_state.players[0].active = 1; 
    local_state.players[0].health = 100; 
    local_state.players[0].pos.y = 5.0f; 
    local_state.players[0].current_weapon = 1;
    for(int j=0; j<5; j++) local_state.players[0].ammo[j] = WPN_STATS[j].ammo_max;
}

void local_reload() {
    PlayerState *p = &local_state.players[local_pid]; if (p->current_weapon == 0) return;
    if (p->reload_timer <= 0 && p->ammo[p->current_weapon] < WPN_STATS[p->current_weapon].ammo_max) p->reload_timer = RELOAD_TIME;
}

void local_shoot() {
    PlayerState *p = &local_state.players[local_pid]; int w = p->current_weapon;
    if (p->attack_cooldown > 0 || p->reload_timer > 0) return;
    if (w != 0 && p->ammo[w] <= 0) return;
    if(w != 0) p->ammo[w]--;
    p->attack_cooldown = WPN_STATS[w].rof; p->is_shooting = 5;
    int pellets = WPN_STATS[w].cnt; float range = WPN_STATS[w].range;
    Vec3 origin = { p->pos.x, p->pos.y + 1.5f, p->pos.z };
    for (int k=0; k<pellets; k++) {
        Vec3 dir = loc_get_aim(p->yaw, p->pitch, WPN_STATS[w].spr);
        if (map_ray_cast(&local_level, origin, dir, range)) continue;
        for(int i=0; i<MAX_CLIENTS; i++) {
            if (!local_state.players[i].active || i == local_pid) continue;
            if (loc_dist_sq(origin, local_state.players[i].pos) > (range * range)) continue;
            if (loc_ray_hit(origin, dir, local_state.players[i].pos, 1.5f)) {
                p->hit_feedback = 2; local_state.players[i].health -= WPN_STATS[w].dmg;
                if(local_state.players[i].health <= 0) {
                    p->kills++; local_state.players[i].health = 100;
                    local_state.players[i].pos.x = ((rand()%20)-10); local_state.players[i].pos.z = (rand()%20);
                    local_state.players[i].pos.y = 10.0f;
                }
            }
        }
    }
}

void apply_friction(PlayerState *p, float t) {
    p->vel.x *= FRICTION;
    p->vel.z *= FRICTION;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    // Basic Additive Acceleration
    p->vel.x += wish_x * accel;
    p->vel.z += wish_z * accel;
    
    // Clamp Speed (Soft)
    float s = sqrtf(p->vel.x*p->vel.x + p->vel.z*p->vel.z);
    if (s > MAX_SPEED) {
        float scale = MAX_SPEED / s;
        p->vel.x *= scale;
        p->vel.z *= scale;
    }
}

void local_update(float dt, float fwd, float strafe, float yaw, float pitch, int shoot, int weapon, int jump, int crouch, int reload) {
    local_state.server_tick++; PlayerState *p = &local_state.players[local_pid];
    p->yaw = yaw; p->pitch = pitch;
    if (weapon >= 0 && weapon < 5) p->current_weapon = weapon;
    
    apply_friction(p, dt);
    
    float r = -yaw * (3.14159f/180.0f);
    float fwd_x = sinf(r); float fwd_z = -cosf(r); 
    float right_x = cosf(r); float right_z = sinf(r);
    float wish_x = (fwd_x * fwd) + (right_x * strafe); 
    float wish_z = (fwd_z * fwd) + (right_z * strafe);
    float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_len > 0) { wish_x /= wish_len; wish_z /= wish_len; }

    float target_speed = crouch ? (MAX_SPEED * 0.5f) : MAX_SPEED;
    
    // Movement
    if (wish_len > 0) accelerate(p, wish_x, wish_z, target_speed, ACCEL);

    // Gravity
    p->vel.y -= GRAVITY;
    
    // Jump input (simplified for client prediction)
    if (jump && fabs(p->vel.y) < 0.05f) p->vel.y = JUMP_POWER;

    // Integrate
    p->pos.x += p->vel.x; 
    p->pos.z += p->vel.z; 
    p->pos.y += p->vel.y;
    
    // Collision
    map_resolve_collision(&local_level, &p->pos, &p->vel);
    
    if (reload) local_reload();
    if (p->reload_timer > 0) { p->reload_timer--; if (p->reload_timer == 0) p->ammo[p->current_weapon] = WPN_STATS[p->current_weapon].ammo_max; } 
    else { if(shoot) local_shoot(); }
    if (p->attack_cooldown > 0) p->attack_cooldown--; if (p->is_shooting > 0) p->is_shooting--;
    for(int i=0; i<MAX_CLIENTS; i++) {
        if(!local_state.players[i].active || i == local_pid) continue;
        PlayerState *curr = &local_state.players[i];
        if (curr->attack_cooldown > 0) curr->attack_cooldown--; if (curr->is_shooting > 0) curr->is_shooting--; curr->hit_feedback = 0;
    }
}
#endif
