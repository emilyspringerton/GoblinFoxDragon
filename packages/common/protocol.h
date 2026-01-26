#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_CLIENTS 70
#define MAX_WEAPONS 5
#define MAX_PROJECTILES 1024

#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

#define RELOAD_TIME 60
#define SHIELD_REGEN_DELAY 180 // 3 Seconds @ 60 FPS


// --- PHASE 405: GAME MODES ---
typedef enum {
    MODE_DEATHMATCH = 0, // Default: FFA, Instant Respawn
    MODE_TDM        = 1, // Red vs Blue, Kill Limit
    MODE_SURVIVAL   = 2, // CS Style: No Respawn, Round Based
    MODE_CTF        = 3, // Capture The Flag
    MODE_ODDBALL    = 4  // Halo Style: Hold the Ball for time
} GameMode;

#define TEAM_RED  1
#define TEAM_BLUE 2
#define TEAM_FFA  0

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

    int team;           // TEAM_RED, TEAM_BLUE, TEAM_FFA
    int object_held;    // 0=None, 1=Oddball, 2=Flag

    float x, y, z;
    float vx, vy, vz;
    int owner_id;
} Projectile;

typedef struct {
    int id;
    int active;

    int team;           // TEAM_RED, TEAM_BLUE, TEAM_FFA
    int object_held;    // 0=None, 1=Oddball, 2=Flag

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
    int shield;             // <--- NEW
    int shield_regen_timer; // <--- NEW
    
    int kills;
    int hit_feedback;       // 10 = Body, 20 = Head
    float recoil_anim;
    
    float accumulated_reward; 
} PlayerState;


// --- PHASE 310: LAG COMPENSATION ---
#define LAG_HISTORY 64

typedef struct {
    int active;

    int team;           // TEAM_RED, TEAM_BLUE, TEAM_FFA
    int object_held;    // 0=None, 1=Oddball, 2=Flag

    unsigned int timestamp;
    float x, y, z;
    // We can add hitbox bounds here later if we implement crouching hitboxes
} LagRecord;

typedef struct {
    LagRecord history[MAX_CLIENTS][LAG_HISTORY]; // Time Machine
    
    // --- MODE STATE (Phase 405) ---
    int game_mode;          // See GameMode enum
    int round_state;        // 0=Warmup, 1=Active, 2=End
    unsigned int round_timer; // Time remaining
    
    // Scoring
    int team_scores[3];     // Index 0=FFA, 1=Red, 2=Blue
    
    // Oddball
    int oddball_holder;     // Client ID of carrier (-1 if dropped)
    float oddball_pos[3];   // World position if dropped
    
    // CTF
    int flag_carrier_red;   // Who has the Red flag? (Blue player)
    int flag_carrier_blue;  // Who has the Blue flag? (Red player)
    float flag_pos_red[3];
    float flag_pos_blue[3];

    PlayerState players[MAX_CLIENTS];
    Projectile projectiles[MAX_PROJECTILES];
    int server_tick;
} ServerState;


// --- PHASE 300: SERVER AUTHORITY PROTOCOL ---

// Button Bits (for UserCmd)
#define BTN_ATTACK  (1 << 0)
#define BTN_JUMP    (1 << 1)
#define BTN_CROUCH  (1 << 2)
#define BTN_RELOAD  (1 << 3)
#define BTN_USE     (1 << 4)

// 1. Client Intent (The Input)
typedef struct {
    unsigned int sequence;  // Monotonic counter (1, 2, 3...)
    unsigned int timestamp; // Client time
    
    float fwd;      // -127 to 127 (normalized to float for sim)
    float str;      // -127 to 127
    float yaw;      // View Angle
    float pitch;    // View Angle
    
    int weapon_idx; // Desired weapon
    unsigned char buttons; // Bitmask of BTN_*
    
    unsigned short msec; // Duration of this command (usually 16ms)
} UserCmd;

// 2. The Header (Traffic Control)
typedef enum {
    PACKET_CONNECT = 0,
    PACKET_USERCMD,
    PACKET_SNAPSHOT,
    PACKET_DISCONNECT
} NetPacketType;

typedef struct {
    unsigned int type; // NetPacketType
    unsigned int client_id;
} NetHeader;

// 3. The "Truth" (Server Update)
// This will eventually wrap ServerState for delta compression.

#endif
