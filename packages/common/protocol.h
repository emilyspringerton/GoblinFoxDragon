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
#define RELOAD_TIME_FULL 60
#define RELOAD_TIME_TACTICAL 42
#define SHIELD_REGEN_DELAY 180

typedef struct { unsigned char type; unsigned char client_id; unsigned short sequence; unsigned int timestamp; unsigned char entity_count; } NetHeader;
typedef struct { unsigned int sequence; unsigned int timestamp; unsigned short msec; float fwd; float str; float yaw; float pitch; unsigned int buttons; int weapon_idx; } UserCmd;
#define BTN_JUMP 1
#define BTN_ATTACK 2
#define BTN_CROUCH 4
#define BTN_RELOAD 8
#define BTN_USE 16

typedef struct { int id; int dmg; int rof; int cnt; float spr; int ammo_max; } WeaponStats;
static const WeaponStats WPN_STATS[5] = { {0,200,20,1,0.0f,0}, {1,45,25,1,0.0f,6}, {2,20,6,1,0.04f,30}, {3,64,50,8,0.15f,8}, {4,101,70,1,0.0f,5} };
typedef struct { int active; float x,y,z; float vx,vy,vz; int owner_id; } Projectile;
typedef struct { unsigned char id; float x,y,z; float yaw,pitch; unsigned char current_weapon; unsigned char state; unsigned char health, shield, is_shooting, crouching; float reward_feedback; unsigned char ammo, in_vehicle, hit_feedback; } NetPlayer;
typedef struct { int id, active, is_bot; float x,y,z,vx,vy,vz,yaw,pitch; int on_ground; float in_fwd, in_strafe; int in_jump, in_shoot, in_reload, crouching, in_use, current_weapon, ammo[5], reload_timer, attack_cooldown, is_shooting, jump_timer, health, shield, shield_regen_timer, state, kills, deaths, hit_feedback; float recoil_anim; int in_vehicle, vehicle_cooldown; float accumulated_reward; unsigned int respawn_time; } PlayerState;
typedef enum { MODE_DEATHMATCH=0, MODE_LOCAL=98, MODE_NET=99 } GameMode;
typedef struct { PlayerState players[70]; Projectile projectiles[1024]; int server_tick, game_mode; } ServerState;
#endif
