#ifndef SHANK_H
#define SHANK_H

#include <SDL2/SDL.h>
#include <SDL2/SDL_opengl.h>
#include <stdbool.h>
#include <math.h>
#include "../../../packages/protocol/protocol.h" // Include Shared Protocol

// --- Constants ---
#define WINDOW_WIDTH 1280
#define WINDOW_HEIGHT 720
#define FOV 70.0f

// --- Core Entities ---
typedef struct {
    Vec3 pos;
    Vec3 vel;
    float yaw;
    float pitch;
    int current_weapon;
    int health;
    int ammo[6];
} Player;

typedef struct {
    bool running;
    SDL_Window* window;
    SDL_GLContext gl_context;
    Player local_player;
    
    // NEW: Network Data
    ServerState last_server_state;
} GameContext;

#endif
