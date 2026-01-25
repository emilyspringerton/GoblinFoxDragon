#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_WEAPONS 5
#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

typedef struct {
    float x, y, z;      // Position
    float vx, vy, vz;   // Velocity
    float yaw, pitch;   // View Angles
    int on_ground;      // Flags
    int crouching;
    int current_weapon;
    int health;
    float recoil_anim;  // Visual kickback 0.0 - 1.0
} PlayerState;

#endif
