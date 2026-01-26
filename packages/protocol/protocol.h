#ifndef PROTOCOL_H
#define PROTOCOL_H

#include <stdint.h>

#define PORT 5314 
#define MAX_CLIENTS 64
#define MAX_WEAPONS 5

// PACKET TYPES
#define PACKET_INPUT 1
#define PACKET_STATE 2
#define PACKET_LOBBY_CMD 3
#define PACKET_LOBBY_INFO 4

// LOBBY COMMANDS
#define CMD_SEARCH 1
#define CMD_CREATE 2
#define CMD_JOIN   3
#define CMD_NAME   5  // <-- NEW: Set Name

// Weapon IDs
#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

typedef struct { float x, y, z; } Vec3;

typedef struct {
    int id; int dmg; int rof; int cnt; float spr; int ammo_max; float range;
} WeaponStats;

static WeaponStats WPN_STATS[MAX_WEAPONS] = {
    {0, 110, 20, 1, 0.0f, 0, 3.5f}, {1, 32, 12, 1, 0.0f, 9, 200.0f}, 
    {2, 16, 6, 1, 0.04f, 30, 200.0f}, {3, 50, 50, 12, 0.22f, 8, 70.0f}, {4, 101, 70, 1, 0.0f, 5, 500.0f}
};

typedef struct {
    int active;
    char name[32]; // <-- NEW: Player Name
    Vec3 pos; Vec3 vel;
    float yaw; float pitch;
    int health; int kills;
    int current_weapon; int ammo[MAX_WEAPONS];
    int attack_cooldown; int is_shooting;
    int hit_feedback; int color; int reload_timer; 
} PlayerState;

typedef struct {
    int server_tick;
    int player_count;
    char status_msg[64];
    PlayerState players[MAX_CLIENTS];
} ServerState;

typedef struct {
    int type;
    int owner_id; 
    int cmd_id; 
    char data[4096];
} Packet;

typedef struct {
    float fwd; float strafe; float yaw; float pitch;
    int jump; int crouch; int shoot; int reload;
    int weapon_req; 
    int zoom; 
} ClientInput;

#endif

