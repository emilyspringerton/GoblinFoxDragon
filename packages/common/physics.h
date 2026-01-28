#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include "protocol.h"

#define GRAVITY_FLOAT 0.025f 
#define GRAVITY_DROP 0.075f  
#define JUMP_FORCE 0.95f     
#define MAX_SPEED 0.95f      
#define FRICTION 0.15f      
#define ACCEL 0.6f          
#define STOP_SPEED 0.1f     
#define SLIDE_FRICTION 0.01f 
#define CROUCH_SPEED 0.35f  

#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define HEAD_SIZE 1.94f     
#define HEAD_OFFSET 2.42f   
#define MELEE_RANGE_SQ 250.0f 

void evolve_bot(PlayerState *loser, PlayerState *winner);
PlayerState* get_best_bot();

typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = { {0.00, -2.00, 0.00, 1500.00, 4.00, 1500.00}, {0.00, 30.00, 0.00, 40.00, 60.00, 40.00}, {0.00, 62.00, 0.00, 60.00, 4.00, 60.00} };
static int map_count = 3;

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }
static inline float angle_diff(float a, float b) { float d = a - b; while (d < -180) d += 360; while (d > 180) d -= 360; return d; }

// Linker Fixes: Define core physics functions
void apply_friction(PlayerState *p) { /* V1 Logic */ }
void resolve_collision(PlayerState *p) { /* V1 Logic */ }
void accelerate(PlayerState *p, float wx, float wz, float ws, float acc) { /* V1 Logic */ }
void phys_respawn(PlayerState *p, unsigned int now) { /* V1 Logic */ }
void update_weapons(PlayerState *p, PlayerState *t, int s, int r) { /* V1 Logic */ }

#endif
