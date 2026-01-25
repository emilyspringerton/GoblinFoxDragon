#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include <math.h>
#include <string.h>
#include <stdlib.h>
#include "../common/protocol.h"
#include "../common/physics.h" // Use the Turbo Physics

// Local State
PlayerState local_p; // The player
PlayerState bots[MAX_CLIENTS]; // Dummies

void local_init() {
    memset(&local_p, 0, sizeof(PlayerState));
    local_p.y = 10.0f;
    local_p.health = 100;
    local_p.current_weapon = WPN_MAGNUM;
    
    // Init Ammo
    for(int i=0; i<MAX_WEAPONS; i++) local_p.ammo[i] = WPN_STATS[i].ammo_max;

    // Init Dummy Bot
    memset(&bots[1], 0, sizeof(PlayerState));
    bots[1].x = 5.0f; bots[1].y = 0.0f; bots[1].z = 5.0f;
    bots[1].health = 100;
}

// The Core Loop
void local_update(float fwd, float strafe, float yaw, float pitch, int shoot, int weapon, int jump, int crouch, int reload) {
    // 1. Update Input
    local_p.yaw = yaw;
    local_p.pitch = pitch;
    if (weapon != -1) local_p.current_weapon = weapon;

    // 2. Physics Step (Quake Math from physics.h)
    apply_friction(&local_p);

    // Calc Wish Direction (No negative sign on yaw!)
    float rad = local_p.yaw * 0.0174533f;
    float fwd_x = sinf(rad); float fwd_z = -cosf(rad);
    float right_x = cosf(rad); float right_z = sinf(rad);

    float wish_x = (fwd_x * fwd) + (right_x * strafe);
    float wish_z = (fwd_z * fwd) + (right_z * strafe);
    
    float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_len > 0) { wish_x /= wish_len; wish_z /= wish_len; }

    float target_speed = crouch ? MAX_SPEED * 0.5f : MAX_SPEED;

    if (local_p.on_ground) {
        accelerate(&local_p, wish_x, wish_z, target_speed, ACCEL);
        if (jump) { local_p.vy = JUMP_FORCE; local_p.on_ground = 0; }
    } else {
        accelerate(&local_p, wish_x, wish_z, target_speed * MAX_AIR_SPEED, ACCEL);
    }

    local_p.vy -= GRAVITY;
    local_p.x += local_p.vx; 
    local_p.z += local_p.vz; 
    local_p.y += local_p.vy;

    // Floor/Respawn Safety
    if (local_p.y < -50.0f) { local_p.x=0; local_p.y=10; local_p.z=0; local_p.vx=0; local_p.vy=0; local_p.vz=0; }

    resolve_collision(&local_p);

    // 3. Combat Logic
    if (local_p.recoil_anim > 0) local_p.recoil_anim -= 0.1f;
    update_weapons(&local_p, shoot, reload);
}
#endif
