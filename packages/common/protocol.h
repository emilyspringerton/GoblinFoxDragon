#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_CLIENTS 10
#define MAX_WEAPONS 5

// Weapon IDs
#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

#define RELOAD_TIME 60

typedef struct { float x, y, z; } Vec3;

typedef struct {
    int id; int dmg; int rof; int cnt; float spr; int ammo_max;
} WeaponStats;

// Stats Registry
static const WeaponStats WPN_STATS[MAX_WEAPONS] = {
    {WPN_KNIFE,   35, 20, 1, 0.0f,  0},
    {WPN_MAGNUM,  45, 25, 1, 0.0f,  6},
    {WPN_AR,      12, 6,  1, 0.04f, 30},
    {WPN_SHOTGUN, 8,  50, 8, 0.15f, 8},
    {WPN_SNIPER,  90, 70, 1, 0.0f,  5}
};

typedef struct {
    int active;
    Vec3 pos; // REVERTED TO VEC3
    Vec3 vel;
    float yaw, pitch;
    int on_ground;
    int crouching;
    
    // Combat
    int current_weapon;
    int ammo[MAX_WEAPONS];
    int reload_timer;
    int attack_cooldown;
    int is_shooting;
    
    int health;
    int kills;
    int hit_feedback;
    float recoil_anim;
} PlayerState;

#endif
