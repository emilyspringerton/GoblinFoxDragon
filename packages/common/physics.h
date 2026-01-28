#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include "protocol.h"

// --- MOON TUNING (PHASE 455) ---
#define GRAVITY_FLOAT 0.025f 
#define GRAVITY_DROP 0.075f  
#define JUMP_FORCE 0.95f     
#define MAX_SPEED 0.95f      
#define FRICTION 0.15f      
#define ACCEL 0.6f          
#define STOP_SPEED 0.1f     
#define SLIDE_FRICTION 0.01f 
#define CROUCH_SPEED 0.35f  

// --- GOLDEN RATIO SCALE (1.618) ---
#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define HEAD_SIZE 1.94f     
#define HEAD_OFFSET 2.42f   
#define MELEE_RANGE_SQ 250.0f 

// Forward decl
void evolve_bot(PlayerState *loser, PlayerState *winner);
PlayerState* get_best_bot();

typedef struct { float x, y, z, w, h, d; } Box;

// --- THE PHI-FORTRESS (Procedurally Generated) ---
static Box map_geo[] = {
    {0.00, -2.00, 0.00, 3000.00, 4.00, 3000.00},
    {0.00, 10.50, 0.00, 144.00, 21.00, 144.00},
    {72.00, 21.00, 72.00, 10.00, 2.00, 10.00},
    {-72.00, 21.00, -72.00, 10.00, 2.00, 10.00},
    {0.00, 29.25, 0.00, 89.00, 16.51, 89.00},
    {44.50, 37.51, 44.50, 10.00, 2.00, 10.00},
    {-44.50, 37.51, -44.50, 10.00, 2.00, 10.00},
    {0.00, 44.00, 0.00, 55.00, 12.98, 55.00},
    {27.50, 50.49, 27.50, 10.00, 2.00, 10.00},
    {-27.50, 50.49, -27.50, 10.00, 2.00, 10.00},
    {0.00, 55.59, 0.00, 33.99, 10.20, 33.99},
    {17.00, 60.69, 17.00, 10.00, 2.00, 10.00},
    {-17.00, 60.69, -17.00, 10.00, 2.00, 10.00},
    {0.00, 64.70, 0.00, 21.01, 8.02, 21.01},
    {10.50, 68.71, 10.50, 10.00, 2.00, 10.00},
    {-10.50, 68.71, -10.50, 10.00, 2.00, 10.00},
    {0.00, 71.87, 0.00, 12.98, 6.31, 12.98},
    {6.49, 75.02, 6.49, 10.00, 2.00, 10.00},
    {-6.49, 75.02, -6.49, 10.00, 2.00, 10.00},
    {0.00, 77.50, 0.00, 8.02, 4.96, 8.02},
    {4.01, 79.98, 4.01, 10.00, 2.00, 10.00},
    {-4.01, 79.98, -4.01, 10.00, 2.00, 10.00},
    {0.00, 5.00, 200.00, 6.00, 2.00, 6.00},
    {0.00, 10.00, 209.00, 10.00, 2.00, 10.00},
    {0.00, 15.00, 224.00, 16.00, 2.00, 16.00},
    {0.00, 20.00, 248.00, 26.00, 2.00, 26.00},
    {0.00, 25.00, 287.00, 42.00, 2.00, 42.00},
    {0.00, 30.00, 350.00, 68.00, 2.00, 68.00},
    {0.00, 35.00, 452.00, 110.00, 2.00, 110.00},
    {0.00, 40.00, 617.00, 178.00, 2.00, 178.00},
    {400.00, 10.00, 0.00, 15.00, 2.00, 15.00},
    {215.21, 12.00, 77.69, 15.50, 2.00, 15.50},
    {311.33, 14.00, -129.51, 16.00, 2.00, 16.00},
    {388.27, 16.00, 115.04, 16.50, 2.00, 16.50},
    {142.43, 18.00, -27.78, 17.00, 2.00, 17.00},
    {447.59, 20.00, -94.03, 17.50, 2.00, 17.50},
    {250.82, 22.00, 183.53, 18.00, 2.00, 18.00},
    {205.34, 24.00, -181.84, 18.50, 2.00, 18.50},
    {506.73, 26.00, 75.24, 19.00, 2.00, 19.00},
    {82.89, 28.00, 89.93, 19.50, 2.00, 19.50},
    {405.65, 30.00, -226.58, 20.00, 2.00, 20.00},
    {379.69, 32.00, 252.73, 20.50, 2.00, 20.50},
    {57.51, 34.00, -140.00, 21.00, 2.00, 21.00},
    {588.01, 36.00, -63.85, 21.50, 2.00, 21.50},
    {122.19, 38.00, 253.94, 22.00, 2.00, 22.00},
    {257.58, 40.00, -322.22, 22.50, 2.00, 22.50},
    {560.46, 42.00, 218.55, 23.00, 2.00, 23.00},
    {-54.66, 44.00, 15.48, 23.50, 2.00, 23.50},
    {561.63, 46.00, -261.63, 24.00, 2.00, 24.00},
    {283.21, 48.00, 384.63, 24.50, 2.00, 24.50},
    {42.88, 50.00, -306.42, 25.00, 2.00, 25.00},
    {711.45, 52.00, 54.17, 25.50, 2.00, 25.50},
    {-52.24, 54.00, 246.64, 26.00, 2.00, 26.00},
    {396.32, 56.00, -434.45, 26.50, 2.00, 26.50},
    {530.00, 58.00, 398.37, 27.00, 2.00, 27.00},
    {-153.02, 60.00, -142.84, 27.50, 2.00, 27.50},
    {744.09, 62.00, -207.08, 28.00, 2.00, 28.00},
    {106.74, 64.00, 466.56, 28.50, 2.00, 28.50},
    {122.15, 66.00, -488.64, 29.00, 2.00, 29.00},
    {774.55, 68.00, 247.04, 29.50, 2.00, 29.50},
    {-300.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-340.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-380.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-420.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-460.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-500.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-540.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-580.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-620.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-660.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-700.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-740.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-780.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-820.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-860.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-900.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-940.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-980.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-1020.00, 40.00, 0.00, 60.00, 4.00, 20.00},
    {-1060.00, 45.00, 0.00, 10.00, 1.00, 10.00},
    {-90.00, 10.00, -490.00, 60.00, 20.00, 60.00},
    {-90.00, 22.00, -490.00, 30.00, 10.00, 30.00},
    {-90.00, 10.00, -400.00, 60.00, 20.00, 60.00},
    {-90.00, 22.00, -400.00, 30.00, 10.00, 30.00},
    {-90.00, 10.00, -310.00, 60.00, 20.00, 60.00},
    {-90.00, 22.00, -310.00, 30.00, 10.00, 30.00},
    {0.00, 10.00, -490.00, 60.00, 20.00, 60.00},
    {0.00, 22.00, -490.00, 30.00, 10.00, 30.00},
    {0.00, 10.00, -310.00, 60.00, 20.00, 60.00},
    {0.00, 22.00, -310.00, 30.00, 10.00, 30.00},
    {90.00, 10.00, -490.00, 60.00, 20.00, 60.00},
    {90.00, 22.00, -490.00, 30.00, 10.00, 30.00},
    {90.00, 10.00, -400.00, 60.00, 20.00, 60.00},
    {90.00, 22.00, -400.00, 30.00, 10.00, 30.00},
    {90.00, 10.00, -310.00, 60.00, 20.00, 60.00},
    {90.00, 22.00, -310.00, 30.00, 10.00, 30.00}
};
static int map_count = sizeof(map_geo) / sizeof(Box);

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }

static inline float angle_diff(float a, float b) {
    float d = a - b;
    while (d < -180) d += 360;
    while (d > 180) d -= 360;
    return d;
}

int check_hit_location(float ox, float oy, float oz, float dx, float dy, float dz, PlayerState *target) {
    if (!target->active) return 0;
    float tx = target->x; float tz = target->z;
    float head_y = target->y + HEAD_OFFSET; float h_size = HEAD_SIZE;
    float vx = tx - ox, vy = head_y - oy, vz = tz - oz;
    float t = vx*dx + vy*dy + vz*dz;
    if (t > 0) {
        float cx = ox + dx*t, cy = oy + dy*t, cz = oz + dz*t;
        float dist_sq = (tx-cx)*(tx-cx) + (head_y-cy)*(head_y-cy) + (tz-cz)*(tz-cz);
        if (dist_sq < (h_size*h_size)) return 2; 
    }
    float body_y = target->y + 2.0f;
    vx = tx - ox; vy = body_y - oy; vz = tz - oz;
    t = vx*dx + vy*dy + vz*dz;
    if (t > 0) {
        float cx = ox + dx*t, cy = oy + dy*t, cz = oz + dz*t;
        float dist_sq = (tx-cx)*(tx-cx) + (body_y-cy)*(body_y-cy) + (tz-cz)*(tz-cz);
        if (dist_sq < 7.2f) return 1; 
    }
    return 0;
}

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    
    float drop = 0;
    if (p->on_ground) {
        if (p->crouching) {
            if (speed > 0.75f) drop = speed * SLIDE_FRICTION; 
            else drop = speed * (FRICTION * 3.0f); 
        } else {
            float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
            drop = control * FRICTION; 
        }
    } else { drop = 0; }
    
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vx *= newspeed; p->vz *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (p->crouching && speed > 0.75f && p->on_ground) return;
    if (p->crouching && p->on_ground && speed < 0.75f && wish_speed > CROUCH_SPEED) wish_speed = CROUCH_SPEED;

    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * MAX_SPEED; 
    if (acc_speed > add_speed) acc_speed = add_speed;
    
    p->vx += acc_speed * wish_x; 
    p->vz += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = PLAYER_WIDTH; 
    float ph = p->crouching ? (PLAYER_HEIGHT / 2.0f) : PLAYER_HEIGHT; 
    p->on_ground = 0;
    if (p->y < 0) { p->y = 0; p->vy = 0; p->on_ground = 1; }
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (p->x + pw > b.x - b.w/2 && p->x - pw < b.x + b.w/2 &&
            p->z + pw > b.z - b.d/2 && p->z - pw < b.z + b.d/2) {
            if (p->y < b.y + b.h/2 && p->y + ph > b.y - b.h/2) {
                float prev_y = p->y - p->vy;
                if (prev_y >= b.y + b.h/2) {
                    p->y = b.y + b.h/2; p->vy = 0; p->on_ground = 1;
                } else {
                    float dx = p->x - b.x; float dz = p->z - b.z;
                    float w = (b.w > 0.1f) ? b.w : 1.0f;
                    float d = (b.d > 0.1f) ? b.d : 1.0f;
                    if (fabs(dx)/w > fabs(dz)/d) { 
                        p->vx = 0; 
                        p->x = (dx > 0) ? b.x + b.w/2 + pw : b.x - b.w/2 - pw;
                    } else { 
                        p->vz = 0; 
                        p->z = (dz > 0) ? b.z + b.d/2 + pw : b.z - b.d/2 - pw;
                    }
                }
            }
        }
    }
}

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1;
    p->state = STATE_ALIVE;
    p->health = 100;
    p->shield = 100;
    p->respawn_time = 0;
    
    // Spawn High on the Spire or Outskirts
    if (rand()%2 == 0) {
        p->x = 0; p->z = 0; p->y = 80; // Top of Spire
    } else {
        float ang = phys_rand_f() * 6.28f;
        p->x = sinf(ang) * 500; p->z = cosf(ang) * 500; p->y = 20;
    }
    
    p->current_weapon = WPN_MAGNUM;
    for(int i=0; i<MAX_WEAPONS; i++) p->ammo[i] = WPN_STATS[i].ammo_max;
    
    if (p->is_bot) {
        PlayerState *winner = get_best_bot();
        if (winner && winner != p) evolve_bot(p, winner);
    }
}

void update_weapons(PlayerState *p, PlayerState *targets, int shoot, int reload) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;

    int w = p->current_weapon;
    if (reload && p->reload_timer == 0 && w != WPN_KNIFE) {
        if (p->ammo[w] < WPN_STATS[w].ammo_max) {
            if (p->ammo[w] > 0) p->reload_timer = RELOAD_TIME_TACTICAL; 
            else p->reload_timer = RELOAD_TIME_FULL; 
        }
    }
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (w != WPN_KNIFE && p->ammo[w] <= 0) p->reload_timer = RELOAD_TIME_FULL; 
        else {
            p->is_shooting = 5; p->recoil_anim = 1.0f;
            p->attack_cooldown = WPN_STATS[w].rof;
            if (w != WPN_KNIFE) p->ammo[w]--;
            
            float r = -p->yaw * 0.0174533f;
            float rp = p->pitch * 0.0174533f;
            float dx = sinf(r) * cosf(rp);
            float dy = sinf(rp);
            float dz = -cosf(r) * cosf(rp);
            
            if (WPN_STATS[w].spr > 0) {
                dx += phys_rand_f() * WPN_STATS[w].spr;
                dy += phys_rand_f() * WPN_STATS[w].spr;
                dz += phys_rand_f() * WPN_STATS[w].spr;
            }

            for(int i=0; i<MAX_CLIENTS; i++) {
                if (p == &targets[i]) continue;
                if (!targets[i].active) continue;
                if (w == WPN_KNIFE) {
                    float kx = p->x - targets[i].x; float ky = p->y - targets[i].y; float kz = p->z - targets[i].z;
                    if ((kx*kx + ky*ky + kz*kz) > MELEE_RANGE_SQ + 22.0f ) continue; 
                }
                int hit_type = check_hit_location(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, &targets[i]);
                if (hit_type > 0) {
                    printf("🔫 HIT! Dmg: %d on Target %d\n", WPN_STATS[w].dmg, i);
                    int damage = WPN_STATS[w].dmg;
                    p->accumulated_reward += 10.0f;
                    targets[i].shield_regen_timer = SHIELD_REGEN_DELAY;
                    if (hit_type == 2 && targets[i].shield <= 0) { damage *= 3; p->hit_feedback = 20; } else { p->hit_feedback = 10; }
                    if (targets[i].shield > 0) {
                        if (targets[i].shield >= damage) { targets[i].shield -= damage; damage = 0; } 
                        else { damage -= targets[i].shield; targets[i].shield = 0; }
                    }
                    targets[i].health -= damage;
                    if(targets[i].health <= 0) {
                        p->kills++; targets[i].deaths++; 
                        p->accumulated_reward += 1000.0f;
                        phys_respawn(&targets[i], 0);
                    }
                }
            }
        }
    }
}

void phys_store_history(ServerState *server, int client_id, unsigned int now) {
    if (client_id < 0 || client_id >= MAX_CLIENTS) return;
    int slot = (now / 16) % LAG_HISTORY; 
    server->history[client_id][slot].active = 1;
    server->history[client_id][slot].timestamp = now;
    server->history[client_id][slot].x = server->players[client_id].x;
    server->history[client_id][slot].y = server->players[client_id].y;
    server->history[client_id][slot].z = server->players[client_id].z;
}

int phys_resolve_rewind(ServerState *server, int client_id, unsigned int target_time, float *out_pos) {
    LagRecord *hist = server->history[client_id];
    for(int i=0; i<LAG_HISTORY; i++) {
        if (!hist[i].active) continue;
        if (hist[i].timestamp == target_time) { 
            out_pos[0] = hist[i].x; out_pos[1] = hist[i].y; out_pos[2] = hist[i].z;
            return 1;
        }
    }
    return 0; 
}
#endif
