#ifndef PROTOCOL_H
#define PROTOCOL_H

#define MAX_CLIENTS 32
#define PACKET_CONNECT 0
#define PACKET_USERCMD 1
#define PACKET_SNAPSHOT 2

// --- GAME MODES ---
typedef enum {
    MODE_DEATHMATCH = 0,
    MODE_TDM        = 1,
    MODE_SURVIVAL   = 2,
    MODE_CTF        = 3,
    MODE_ODDBALL    = 4,
    MODE_LOCAL      = 98,
    MODE_NET        = 99
} GameMode;

#define STATE_ALIVE 0
#define STATE_DEAD  1
#define STATE_SPECTATOR 2
#define MAX_HEALTH 100
#define RESPAWN_DELAY 3000

// --- NETWORK STRUCTURES ---
typedef struct {
    unsigned char type;     // 0=Connect, 1=UserCmd, 2=Snapshot
    unsigned char client_id;
    unsigned short sequence;
    unsigned int timestamp; // Local time
} NetHeader;

typedef struct {
    unsigned int sequence;
    unsigned int timestamp;
    unsigned short msec;
    
    // Inputs
    float fwd;      // -1.0 to 1.0
    float str;      // -1.0 to 1.0
    float yaw;      // Degrees
    float pitch;    // Degrees
    
    // Actions (Bitmask)
    unsigned int buttons; // 1=Jump, 2=Attack, 4=Crouch, 8=Reload
    int weapon_idx;
} UserCmd;

// Input Constants
#define BTN_JUMP   1
#define BTN_ATTACK 2
#define BTN_CROUCH 4
#define BTN_RELOAD 8

// --- WEAPON DEFINITIONS (Restored Phase 410) ---
#define WPN_KNIFE   0
#define WPN_MAGNUM  1
#define WPN_AR      2
#define WPN_SHOTGUN 3
#define WPN_SNIPER  4


// --- GAME STATE ---
typedef struct {
    int active;
    
    // Position & Look
    float x, y, z;
    float vx, vy, vz;
    float yaw, pitch;
    
    // State
    int current_weapon;
    int in_shoot;  // Frame counter for animation/logic
    int in_reload;
    
    // VITALS (Fixed Build 100)
    int health;
    int armor;
    int state; // STATE_ALIVE / STATE_DEAD
    
    // SCORING
    int frags;
    int deaths;
    int team;
    
    // TIMING
    unsigned int last_hit_time;
    unsigned int respawn_time;

    // --- VISUALS & ANIMATION (Restored Phase 410) ---
    int ammo[5];          // Ammo per weapon
    float recoil_anim;    // Visual kickback (0.0 to 1.0)
    int reload_timer;     // Visual reload state
    int hit_feedback;     // Flash screen on hit
    int crouching;        // State flag

} PlayerState;

#define LAG_HISTORY 64
typedef struct {
    int active;
    unsigned int timestamp;
    float x, y, z;
    float vx, vy, vz;
} LagRecord;

typedef struct {
    unsigned int server_tick;
    int game_mode;
    
    // Player Entities
    PlayerState players[MAX_CLIENTS];
    
    // Time Machine
    LagRecord history[MAX_CLIENTS][LAG_HISTORY];
    
    // Mode State
    int team_scores[3];
} ServerState;

#endif
