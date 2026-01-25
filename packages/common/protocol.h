#ifndef PROTOCOL_H
#define PROTOCOL_H

// --- CONSTANTS ---
#define MAX_WEAPONS 5
#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

typedef struct {
    int id; int dmg; int rof; int cnt; float spr; int ammo_max;
} WeaponStats;

// DEFINITIVE WEAPON STATS (From Phase 85)
static const WeaponStats WPN_STATS[MAX_WEAPONS] = {
    {WPN_KNIFE,   35, 20, 1, 0.0f,  0},   // Knife: Fast, Infinite
    {WPN_MAGNUM,  45, 25, 1, 0.0f,  6},   // Magnum: High Dmg
    {WPN_AR,      12, 6,  1, 0.04f, 30},  // AR: Fast, Spread
    {WPN_SHOTGUN, 8,  50, 8, 0.15f, 8},   // Shotgun: 8 pellets
    {WPN_SNIPER,  90, 70, 1, 0.0f,  5}    // Sniper: 1-shot
};

typedef struct {
    float x, y, z;
    float vx, vy, vz;
    float yaw, pitch;
    int on_ground;
    int crouching;
    int current_weapon;
    int health;
    float recoil_anim;
} PlayerState;

#endif
