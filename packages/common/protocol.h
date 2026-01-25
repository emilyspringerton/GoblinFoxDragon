#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_CLIENTS 10
#define MAX_WEAPONS 5

#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

#define RELOAD_TIME 60

typedef struct {
    int id; int dmg; int rof; int cnt; float spr; int ammo_max;
} WeaponStats;

// Global Stats Registry
static const WeaponStats WPN_STATS[MAX_WEAPONS] = {
    {WPN_KNIFE,   35, 20, 1, 0.0f,  0},   // Infinite
    {WPN_MAGNUM,  45, 25, 1, 0.0f,  6},   // 6 Rounds
    {WPN_AR,      12, 6,  1, 0.04f, 30},  // 30 Rounds
    {WPN_SHOTGUN, 8,  50, 8, 0.15f, 8},   // 8 Shells
    {WPN_SNIPER,  90, 70, 1, 0.0f,  5}    // 5 Rounds
};

typedef struct {
    float x, y, z;
    float vx, vy, vz;
    float yaw, pitch;
    int on_ground;
    int crouching;
    
    // Combat State
    int current_weapon;
    int ammo[MAX_WEAPONS];
    int reload_timer;
    int attack_cooldown;
    int is_shooting;
    
    int health;
    float recoil_anim;
} PlayerState;

#endif
