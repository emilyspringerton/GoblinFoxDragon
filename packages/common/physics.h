#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>

// Phase 403 Forward Decl
int phys_resolve_rewind(ServerState *server, int client_id, unsigned int target_time, float *out_pos);

#include <stdlib.h>
#include "protocol.h"

// --- TURBO TUNING ---
#define GRAVITY 0.018f
#define JUMP_FORCE 0.61f
#define MAX_SPEED 0.75f
#define FRICTION 0.42f
#define ACCEL 1.618f
#define STOP_SPEED 0.1f
#define MAX_AIR_SPEED 0.2f
#define EYE_HEIGHT 4.0f
#define HEAD_SIZE 0.8f
#define HEAD_OFFSET 4.5f

// KNIFE RANGE LIMIT
#define MELEE_RANGE_SQ 100.0f // 10.0 * 10.0

typedef struct { float x, y, z, w, h, d; } Box;

// MAP GEOMETRY (Standard 40-count map + auto-calc)
static Box map_geo[] = {
    {0, -1, 0, 900, 2, 300}, // Floor
    {0, 2.0, 0, 24, 4, 24}, {0, 5.0, 0, 14, 2, 14}, {0, 8.0, 0, 6, 4, 6}, // Zig
    {300, 10, 0, 10, 20, 10}, {310, 2, 0, 4, 2, 4}, {308, 5, 5, 4, 2, 4}, // East Spire
    {-300, 10, 0, 10, 20, 10}, {-310, 2, 0, 4, 2, 4}, {-308, 5, -5, 4, 2, 4}, // West Spire
    {100, 2, 0, 40, 4, 2}, {-100, 2, 0, 40, 4, 2}, // Spine
    {150, 2, 40, 6, 4, 6}, {160, 2, 35, 2, 4, 10}, {140, 1, 45, 4, 2, 4}, // E Ruins
    {150, 2, -40, 6, 4, 6}, {160, 2, -35, 2, 4, 10}, {180, 4, 0, 8, 8, 8},
    {-150, 2, 40, 6, 4, 6}, {-160, 2, 35, 2, 4, 10}, {-140, 1, 45, 4, 2, 4}, // W Ruins
    {-150, 2, -40, 6, 4, 6}, {-160, 2, -35, 2, 4, 10}, {-180, 4, 0, 8, 8, 8},
    {250, 5, 0, 4, 1, 4}, {200, 7, 10, 4, 1, 4}, {150, 9, -10, 4, 1, 4}, // E Parkour
    {-250, 5, 0, 4, 1, 4}, {-200, 7, -10, 4, 1, 4}, {-150, 9, 10, 4, 1, 4}, // W Parkour
    {50, 2, 80, 4, 4, 4}, {50, 2, -80, 4, 4, 4}, {-50, 2, 80, 4, 4, 4}, {-50, 2, -80, 4, 4, 4}, 
    {220, 1, 50, 4, 2, 8}, {-220, 1, -50, 4, 2, 8}, // Cover
    
    // BASES (Alpha)
    {350, 5, 20, 40, 10, 2}, {350, 5, -20, 40, 10, 2}, {370, 5, 0, 2, 10, 40}, {330, 5, 12, 2, 10, 16}, {330, 5, -12, 2, 10, 16},
    {350, 10, 15, 40, 1, 10}, {350, 10, -15, 40, 1, 10}, {365, 10, 0, 10, 1, 20}, {335, 10, 0, 10, 1, 20},
    {325, 2, 25, 6, 2, 6}, {328, 4, 25, 6, 2, 6}, {331, 6, 25, 6, 2, 6}, {335, 8, 25, 6, 2, 6}, {360, 1, 0, 4, 2, 4},
    
    // BASES (Omega)
    {-350, 5, 20, 40, 10, 2}, {-350, 5, -20, 40, 10, 2}, {-370, 5, 0, 2, 10, 40}, {-330, 5, 12, 2, 10, 16}, {-330, 5, -12, 2, 10, 16},
    {-350, 10, 15, 40, 1, 10}, {-350, 10, -15, 40, 1, 10}, {-365, 10, 0, 10, 1, 20}, {-335, 10, 0, 10, 1, 20},
    {-325, 2, -25, 6, 2, 6}, {-328, 4, -25, 6, 2, 6}, {-331, 6, -25, 6, 2, 6}, {-335, 8, -25, 6, 2, 6}, {-360, 1, 0, 4, 2, 4},

    // WALLS
    {0, 25, 250, 1200, 50, 200}, {0, 25, -250, 1200, 50, 200}, // N/S Walls
    {550, 25, 0, 200, 50, 800}, {-550, 25, 0, 200, 50, 800} // E/W Walls
};
static int map_count = sizeof(map_geo) / sizeof(Box);

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }

// Return: 0=Miss, 1=Body, 2=Head
int check_hit_location(float ox, float oy, float oz, float dx, float dy, float dz, PlayerState *target) {
    if (!target->active) return 0;
    
    float tx = target->x; 
    float tz = target->z;
    
    // 1. Check Head
    float head_y = target->y + HEAD_OFFSET;
    float h_size = HEAD_SIZE;
    
    float vx = tx - ox, vy = head_y - oy, vz = tz - oz;
    float t = vx*dx + vy*dy + vz*dz;
    if (t > 0) {
        float cx = ox + dx*t, cy = oy + dy*t, cz = oz + dz*t;
        float dist_sq = (tx-cx)*(tx-cx) + (head_y-cy)*(head_y-cy) + (tz-cz)*(tz-cz);
        if (dist_sq < (h_size*h_size)) return 2; 
    }

    // 2. Check Body
    float body_y = target->y + 2.0f;
    vx = tx - ox; vy = body_y - oy; vz = tz - oz;
    t = vx*dx + vy*dy + vz*dz;
    if (t > 0) {
        float cx = ox + dx*t, cy = oy + dy*t, cz = oz + dz*t;
        float dist_sq = (tx-cx)*(tx-cx) + (body_y-cy)*(body_y-cy) + (tz-cz)*(tz-cz);
        if (dist_sq < 12.0f) return 1; // Temporarily boosted for review 
    }
    
    return 0;
}

void apply_friction(PlayerState *p) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }
    if (!p->on_ground) return; 
    float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
    float drop = control * FRICTION;
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vx *= newspeed; p->vz *= newspeed;
}

void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * wish_speed;
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vx += acc_speed * wish_x; p->vz += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = 0.6f; float ph = p->crouching ? 2.0f : 4.0f; 
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

void update_weapons(PlayerState *p, PlayerState *targets, int shoot, int reload, void *server_ptr, unsigned int cmd_time) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;

    int w = p->current_weapon;
    if (reload && p->reload_timer == 0 && w != WPN_KNIFE) {
        if (p->ammo[w] < WPN_STATS[w].ammo_max) p->reload_timer = RELOAD_TIME;
    }
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (w == WPN_KNIFE || p->ammo[w] > 0) {
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

                // --- FIX: KNIFE RANGE CHECK ---
                if (w == WPN_KNIFE) {
                    float kx = p->x - targets[i].x;
                    float ky = p->y - targets[i].y;
                    float kz = p->z - targets[i].z;
                    float kdist = kx*kx + ky*ky + kz*kz;
                    if (kdist > MELEE_RANGE_SQ + 22.0f ) continue; // Too far!
                }

                
                // --- TIME MACHINE (Phase 317) ---
                float original_pos[3];
                int rewound = 0;
                
                // If we are the server (server_ptr provided) and have a valid history time
                if (server_ptr && cmd_time > 0) {
                    ServerState *s = (ServerState*)server_ptr;
                    // Save current state (The Present)
                    original_pos[0] = targets[i].x; 
                    original_pos[1] = targets[i].y; 
                    original_pos[2] = targets[i].z;
                    
                    // Travel to the Past
                    float past_pos[3];
                    if (phys_resolve_rewind(s, i, cmd_time, past_pos)) {
                        targets[i].x = past_pos[0];
                        targets[i].y = past_pos[1];
                        targets[i].z = past_pos[2];
                        rewound = 1;
                    }
                }

                int hit_type = check_hit_location(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, &targets[i]);

                // Restore Reality
                if (rewound) {
                    targets[i].x = original_pos[0];
                    targets[i].y = original_pos[1];
                    targets[i].z = original_pos[2];
                }
                // --------------------------------

                
                if (hit_type > 0) {
                    int damage = WPN_STATS[w].dmg;
                    targets[i].shield_regen_timer = SHIELD_REGEN_DELAY;
                    
                    if (hit_type == 2 && targets[i].shield <= 0) {
                        damage *= 3; 
                        p->hit_feedback = 20; 
                    } else {
                        p->hit_feedback = 10; 
                    }
                    
                    if (targets[i].shield > 0) {
                        if (targets[i].shield >= damage) {
                            targets[i].shield -= damage;
                            damage = 0;
                        } else {
                            damage -= targets[i].shield;
                            targets[i].shield = 0;
                        }
                    }
                    
                    targets[i].health -= damage;
                    
                    if(targets[i].health <= 0) {
                        p->kills++;
                        targets[i].health = 100;
                        targets[i].shield = 100; 
                        targets[i].x = (rand()%800)-400; 
                        targets[i].z = (rand()%200)-100; 
                        targets[i].y = 10;
                    }
                }
            }
        } else {
            p->attack_cooldown = 10; 
        }
    }
}

// --- PHASE 310: TIME MACHINE LOGIC ---
void phys_store_history(ServerState *server, int client_id, unsigned int now) {
    if (client_id < 0 || client_id >= MAX_CLIENTS) return;
    if (!server->players[client_id].active) return;

    // Use a circular buffer index based on tick or simple modulus of timestamp?
    // Using tick count is safer, but for now we'll use a rolling index if we track it.
    // Simpler: Just shift? No, slow.
    // Let's use server_tick % LAG_HISTORY.
    int slot = server->server_tick % LAG_HISTORY;
    
    server->history[client_id][slot].active = 1;
    server->history[client_id][slot].timestamp = now;
    server->history[client_id][slot].x = server->players[client_id].x;
    server->history[client_id][slot].y = server->players[client_id].y;
    server->history[client_id][slot].z = server->players[client_id].z;
}

// Returns 1 if we successfully rewound and wrote to out_pos
int phys_resolve_rewind(ServerState *server, int client_id, unsigned int target_time, float *out_pos) {
    LagRecord *hist = server->history[client_id];
    
    LagRecord *prev = NULL;
    LagRecord *next = NULL;
    
    // Scan history (Optimization: Start from newest)
    // Since it's a ring buffer, we iterate 0..63. Ideally we'd know the head.
    // For this alpha, a linear scan is acceptable (64 items is nothing for a CPU).
    for (int i=0; i<LAG_HISTORY; i++) {
        if (!hist[i].active) continue;
        
        if (hist[i].timestamp <= target_time) {
            if (!prev || hist[i].timestamp > prev->timestamp) prev = &hist[i];
        }
        if (hist[i].timestamp >= target_time) {
            if (!next || hist[i].timestamp < next->timestamp) next = &hist[i];
        }
    }
    
    if (!prev || !next) return 0; // No valid history for this time
    
    // Exact match
    if (prev == next) {
        out_pos[0] = prev->x; out_pos[1] = prev->y; out_pos[2] = prev->z;
        return 1;
    }
    
    // Interpolate
    float total = (float)(next->timestamp - prev->timestamp);
    if (total <= 0.0001f) return 0;
    
    float dt = (float)(target_time - prev->timestamp);
    float frac = dt / total;
    
    out_pos[0] = prev->x + (next->x - prev->x) * frac;
    out_pos[1] = prev->y + (next->y - prev->y) * frac;
    out_pos[2] = prev->z + (next->z - prev->z) * frac;
    
    return 1;
}

#endif
