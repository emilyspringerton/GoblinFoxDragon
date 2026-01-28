#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_CLIENTS 70
#define MAX_WEAPONS 5
#define MAX_PROJECTILES 1024
#define LAG_HISTORY 64

#define PACKET_CONNECT 0
#define PACKET_USERCMD 1
#define PACKET_SNAPSHOT 2
#define PACKET_WELCOME 3

#define STATE_ALIVE 0
#define STATE_DEAD 1

#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

#define BTN_JUMP 1
#define BTN_ATTACK 2
#define BTN_CROUCH 4
#define BTN_RELOAD 8
#define BTN_USE 16

typedef struct { unsigned char type, client_id; unsigned short sequence; unsigned int timestamp; unsigned char entity_count; } NetHeader;
typedef struct { unsigned int sequence, timestamp; unsigned short msec; float fwd, str, yaw, pitch; unsigned int buttons; int weapon_idx; } UserCmd;
typedef struct { int version; float w_aggro, w_strafe, w_jump, w_slide, w_turret, w_repel; } BotGenome;

typedef struct {
    int id, active, is_bot;
    float x, y, z; float vx, vy, vz; float yaw, pitch; int on_ground;
    float in_fwd, in_strafe;
    int in_jump, in_shoot, in_reload, crouching, in_use, current_weapon, ammo[MAX_WEAPONS];
    int reload_timer, attack_cooldown, is_shooting, jump_timer, health, shield, shield_regen_timer, state;
    int kills, deaths, hit_feedback; float recoil_anim;
    int in_vehicle, vehicle_cooldown;
    float accumulated_reward; 
    BotGenome brain;
    unsigned int last_hit_time, respawn_time;
} PlayerState;

typedef struct { int active; float x, y, z, vx, vy, vz; int owner_id; } Projectile;
typedef struct { unsigned char id; float x, y, z, yaw, pitch; unsigned char current_weapon, state, health, shield, is_shooting, crouching; float reward_feedback; unsigned char ammo, in_vehicle, hit_feedback; } NetPlayer;
typedef enum { MODE_DEATHMATCH=0, MODE_LOCAL=98, MODE_NET=99, MODE_EVOLUTION=100 } GameMode;

typedef struct {
    PlayerState players[MAX_CLIENTS];
    Projectile projectiles[MAX_PROJECTILES];
    int server_tick, game_mode;
} ServerState;

#endif
