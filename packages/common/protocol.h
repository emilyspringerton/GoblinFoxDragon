#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_CLIENTS 70
#define MAX_WEAPONS 5
#define MAX_PROJECTILES 1024

#define PACKET_CONNECT 0
#define PACKET_USERCMD 1
#define PACKET_SNAPSHOT 2

#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

#define RELOAD_TIME 60
#define SHIELD_REGEN_DELAY 180 

typedef struct {
    unsigned char type;
    unsigned char client_id;
    unsigned short sequence;
    unsigned int timestamp;
} NetHeader;

typedef struct {
    unsigned int sequence;
    unsigned int timestamp;
    unsigned short msec;
    float fwd;      
    float str;      
    float yaw;      
    float pitch;    
    unsigned int buttons;
    int weapon_idx;
} UserCmd;

#define BTN_JUMP   1
#define BTN_ATTACK 2
#define BTN_CROUCH 4
#define BTN_RELOAD 8

typedef struct {
    int id; int dmg; int rof; int cnt; float spr; int ammo_max;
} WeaponStats;

static const WeaponStats WPN_STATS[MAX_WEAPONS] = {
    {WPN_KNIFE,   200, 20, 1, 0.0f,  0},   
    {WPN_MAGNUM,  45, 25, 1, 0.0f,  6},   
    {WPN_AR,      20, 6,  1, 0.04f, 30},  
    {WPN_SHOTGUN, 64,  50, 8, 0.15f, 8},   
    {WPN_SNIPER,  101, 70, 1, 0.0f,  5}    
};

typedef struct {
    int active;
    float x, y, z;
    float vx, vy, vz;
    int owner_id;
} Projectile;

typedef struct {
    int id;
    int active;
    int is_bot;
    
    float x, y, z;
    float vx, vy, vz;
    float yaw, pitch;
    int on_ground;
    
    float in_fwd;
    float in_strafe;
    int in_jump;
    int in_shoot;
    int in_reload;
    int crouching;
    
    int current_weapon;
    int ammo[MAX_WEAPONS];
    int reload_timer;
    int attack_cooldown;
    int is_shooting;
    int jump_timer;
    
    int health;
    int shield;             
    int shield_regen_timer; 
    
    int kills;
    int deaths; // Added for scoreboard
    int hit_feedback;       
    float recoil_anim;
    
    float accumulated_reward; 
    
    // Legacy support fields to prevent breakages
    unsigned int last_hit_time;
    unsigned int respawn_time;
} PlayerState;

// Game Modes
typedef enum {
    MODE_DEATHMATCH = 0,
    MODE_TDM        = 1,
    MODE_SURVIVAL   = 2,
    MODE_CTF        = 3,
    MODE_ODDBALL    = 4,
    MODE_LOCAL      = 98,
    MODE_NET        = 99
} GameMode;

typedef struct {
    PlayerState players[MAX_CLIENTS];
    Projectile projectiles[MAX_PROJECTILES];
    int server_tick;
    int game_mode;
} ServerState;

#endif
