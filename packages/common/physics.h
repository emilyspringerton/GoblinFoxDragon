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
#define BUGGY_MAX_SPEED 2.5f    
#define BUGGY_ACCEL 0.08f       
#define BUGGY_FRICTION 0.03f    
#define BUGGY_GRAVITY 0.15f     
#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define HEAD_SIZE 1.94f     
#define HEAD_OFFSET 2.42f   
#define MELEE_RANGE_SQ 250.0f 

typedef struct { float x, y, z, w, h, d; } Box;
static Box map_geo[] = { {0.00,-2.00,0.00,1500.00,4.00,1500.00}, {0.00,30.00,0.00,40.00,60.00,40.00}, {0.00,62.00,0.00,60.00,4.00,60.00} };
static int map_count = 3;

// ... Rest of physics implementation as per Build 178 V1 ...
#endif
