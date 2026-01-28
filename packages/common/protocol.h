#ifndef PROTOCOL_H
#define PROTOCOL_H
#ifdef _WIN32
    #include <winsock2.h>
#else
    #include <netinet/in.h>
#endif
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

typedef struct { unsigned char type, client_id; unsigned short sequence; unsigned int timestamp; unsigned char entity_count; } NetHeader;
typedef struct { unsigned int sequence, timestamp; unsigned short msec; float fwd, str, yaw, pitch; unsigned int buttons; int weapon_idx; } UserCmd;
#define BTN_JUMP 1
#define BTN_ATTACK 2
#define BTN_CROUCH 4
#define BTN_RELOAD 8
#define BTN_USE 16 

typedef struct { int id, dmg, rof, cnt; float spr; int ammo_max; } WeaponStats;
static const WeaponStats WPN_STATS[MAX_WEAPONS] = {
    {WPN_KNIFE, 200, 20, 1, 0.0f, 0}, {WPN_MAGNUM, 45, 25, 1, 0.0f, 6},
    {WPN_AR, 20, 6, 1, 0.04f, 30}, {WPN_SHOTGUN, 64, 50, 8, 0.15f, 8},
    {WPN_SNIPER, 101, 70, 1, 0.0f, 5}
};

typedef struct { int version; float w_aggro, w_strafe, w_jump, w_slide, w_turret, w_repel; } BotGenome;
typedef struct { int id, active, is_bot; float x,y,z,vx,vy,vz,yaw,pitch; int on_ground; float in_fwd, in_strafe; int in_jump, in_shoot, in_reload, crouching, in_use, current_weapon, ammo[5], reload_timer, attack_cooldown, is_shooting, jump_timer, health, shield, shield_regen_timer, state, kills, deaths, hit_feedback; float recoil_anim; int in_vehicle, vehicle_cooldown; float accumulated_reward; BotGenome brain; unsigned int last_hit_time, respawn_time; } PlayerState;
typedef struct { int active; float x,y,z,vx,vy,vz; int owner_id; } Projectile;
typedef struct { unsigned char id; float x,y,z,yaw,pitch,current_weapon,state,health,shield,is_shooting,crouching; float reward_feedback; unsigned char ammo,in_vehicle,hit_feedback; } NetPlayer;
typedef struct { int active; unsigned int timestamp; float x,y,z,vx,vy,vz; } LagRecord;
typedef enum { MODE_DEATHMATCH=0, MODE_LOCAL=98, MODE_NET=99, MODE_EVOLUTION=100 } GameMode;
typedef struct { PlayerState players[MAX_CLIENTS]; Projectile projectiles[MAX_PROJECTILES]; LagRecord history[MAX_CLIENTS][LAG_HISTORY]; int server_tick, game_mode; struct sockaddr_in clients[MAX_CLIENTS]; int client_active[MAX_CLIENTS]; } ServerState;
#endif
