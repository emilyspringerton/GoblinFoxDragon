#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "../common/protocol.h"
#include "../common/physics.h"
#include <string.h>
#include <math.h>

// --- STATE ---
static ServerState local_state;
static int was_holding_jump = 0;

// --- PROTOTYPES ---
static void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, int ability, void *server_context, int cmd_time);
static void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time);
static void update_weapons(PlayerState *p, int shoot, int reload, int ability_press);
static void update_projectiles();
static void fire_projectile(PlayerState *p, int damage, int bounces, float speed_mult);
static void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons);
static void local_init_match(int num_players, int mode);

// --- HELPERS ---
static float rand_weight() { return ((float)(rand()%2000)/1000.0f) - 1.0f; } 
static float rand_pos() { return ((float)(rand()%1000)/1000.0f); } 

static void init_genome(BotGenome *g) {
    g->version = 1;
    g->w_aggro = 0.5f + rand_weight() * 0.5f;
    g->w_strafe = rand_weight();
    g->w_jump = 0.05f + rand_pos() * 0.1f; 
    g->w_slide = 0.01f + rand_pos() * 0.05f;
    g->w_turret = 5.0f + rand_pos() * 10.0f;
    g->w_repel = 1.0f + rand_pos();
}

static void evolve_bot(PlayerState *loser, PlayerState *winner) {
    loser->brain = winner->brain;
    loser->brain.w_aggro += rand_weight() * 0.1f;
    loser->brain.w_strafe += rand_weight() * 0.1f;
    loser->brain.w_jump += rand_weight() * 0.01f;
    loser->brain.w_slide += rand_weight() * 0.01f;
}

static PlayerState* get_best_bot() {
    PlayerState *best = NULL;
    float max_score = -99999.0f;
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (!local_state.players[i].active) continue;
        if (local_state.players[i].accumulated_reward > max_score) {
            max_score = local_state.players[i].accumulated_reward;
            best = &local_state.players[i];
        }
    }
    return best;
}

// --- AI IMPLEMENTATION ---
static void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
    PlayerState *me = &players[bot_idx];
    if (me->state == STATE_DEAD) { *out_buttons = 0; return; }

    int target_idx = -1;
    float min_dist = 9999.0f;

    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (i == bot_idx) continue;
        if (!players[i].active) continue;
        if (players[i].state == STATE_DEAD) continue;
        
        float dx = players[i].x - me->x;
        float dz = players[i].z - me->z;
        float dist = sqrtf(dx*dx + dz*dz);
        
        if (i == 0 || dist < min_dist) { 
            if (i == 0) dist *= 0.5f;
            if (dist < min_dist) { min_dist = dist; target_idx = i; }
        }
    }

    if (target_idx != -1) {
        PlayerState *t = &players[target_idx];
        float dx = t->x - me->x;
        float dz = t->z - me->z;
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        float diff = angle_diff(target_yaw, me->yaw);
        float turn_speed = (me->brain.w_turret > 1.0f) ? me->brain.w_turret : 10.0f;
        
        if (diff > turn_speed) diff = turn_speed;
        if (diff < -turn_speed) diff = -turn_speed;
        *out_yaw = me->yaw + diff;
        
        *out_buttons |= BTN_ATTACK;
        
        if (min_dist > 15.0f) *out_fwd = me->brain.w_aggro;
        else if (min_dist < 5.0f) *out_fwd = -me->brain.w_aggro; 
        else *out_fwd = 0.2f; 
        
        *out_yaw += me->brain.w_strafe * 10.0f;
        if (me->on_ground && (rand()%1000 < (me->brain.w_jump * 1000.0f))) *out_buttons |= BTN_JUMP;
        if (me->on_ground && (rand()%1000 < (me->brain.w_slide * 1000.0f))) *out_buttons |= BTN_CROUCH;
        if (me->ammo[me->current_weapon] <= 0) *out_buttons |= BTN_RELOAD;
    } else {
        *out_yaw = me->yaw + 2.0f;
        *out_fwd = 0.5f;
    }
}

// --- PROJECTILES ---
static void fire_projectile(PlayerState *p, int damage, int bounces, float speed_mult) {
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if (!local_state.projectiles[i].active) {
            Projectile *proj = &local_state.projectiles[i];
            proj->active = 1;
            proj->owner_id = p->id;
            proj->damage = damage;
            proj->bounces_left = bounces;
            
            float rad_y = p->yaw * 0.01745f;
            float rad_p = p->pitch * 0.01745f;
            float speed = 4.0f * speed_mult; 
            
            proj->vx = sinf(rad_y) * cosf(rad_p) * speed;
            proj->vy = sinf(rad_p) * speed;
            proj->vz = -cosf(rad_y) * cosf(rad_p) * speed;
            
            proj->x = p->x; proj->y = p->y + EYE_HEIGHT; proj->z = p->z;
            break;
        }
    }
}

static void update_projectiles() {
    for (int i=0; i<MAX_PROJECTILES; i++) {
        Projectile *p = &local_state.projectiles[i];
        if (!p->active) continue;

        float next_x = p->x + p->vx;
        float next_y = p->y + p->vy;
        float next_z = p->z + p->vz;

        float hit_x, hit_y, hit_z, nx, ny, nz;
        // Uses trace_map from physics.h
        if (trace_map(p->x, p->y, p->z, next_x, next_y, next_z, &hit_x, &hit_y, &hit_z, &nx, &ny, &nz)) {
            if (p->bounces_left > 0) {
                // RICOCHET
                reflect_vector(&p->vx, &p->vy, &p->vz, nx, ny, nz);
                p->x = hit_x; p->y = hit_y; p->z = hit_z;
                p->bounces_left--;
            } else {
                p->active = 0; // Destroy
            }
        } else {
            p->x = next_x; p->y = next_y; p->z = next_z;
        }
        
        if (p->x > 4000 || p->x < -4000 || p->z > 4000 || p->z < -4000 || p->y > 2000) p->active = 0;
    }
}

// --- WEAPONS ---
static void update_weapons(PlayerState *p, int shoot, int reload, int ability_press) {
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->ability_cooldown > 0) p->ability_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;

    // STORM ARROW ACTIVATION
    if (ability_press && p->ability_cooldown == 0 && p->storm_charges == 0) {
        p->storm_charges = 5;
        p->ability_cooldown = 480; // 8s
    }

    int w = p->current_weapon;
    int rof = WPN_STATS[w].rof;
    int damage = WPN_STATS[w].dmg;
    
    // STORM MODE OVERRIDES
    if (p->storm_charges > 0 && w == WPN_SNIPER) {
        rof = 8; 
        damage = (int)(damage * 0.7f); 
    }

    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (p->storm_charges > 0 && w == WPN_SNIPER) {
            fire_projectile(p, damage, 1, 1.5f);
            p->storm_charges--;
            p->attack_cooldown = rof;
            p->recoil_anim = 0.5f;
        } else {
            p->is_shooting = 5; 
            p->recoil_anim = 1.0f;
            p->attack_cooldown = rof;
            if (w != WPN_KNIFE) p->ammo[w]--;
            
            // Hitscan logic (Simplified for bots/local)
            for(int i=0; i<MAX_CLIENTS; i++) {
                if(&local_state.players[i] == p || !local_state.players[i].active) continue;
                // Basic distance check for hitscan
                float dx = local_state.players[i].x - p->x;
                float dz = local_state.players[i].z - p->z;
                if(sqrtf(dx*dx + dz*dz) < 5.0f && fabsf(p->yaw - atan2f(dx, dz)*57.29f) < 20.0f) {
                    local_state.players[i].health -= damage;
                    p->hit_feedback = 10;
                    if(local_state.players[i].health <= 0) {
                        p->kills++;
                        phys_respawn(&local_state.players[i], 0);
                    }
                }
            }
        }
    }
}

// --- ENTITY UPDATE ---
static void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time) {
    if (!p->active) return;
    if (p->state == STATE_DEAD) return;

    apply_friction(p);
    float g = (p->in_jump) ? GRAVITY_FLOAT : GRAVITY_DROP;
    p->vy -= g; 
    p->y += p->vy;
    
    resolve_collision(p);
    p->x += p->vx;
    p->z += p->vz;

    if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
    if (p->recoil_anim < 0) p->recoil_anim = 0;
    if (p->hit_feedback > 0) p->hit_feedback--;
}

// --- MAIN LOOP ---
static void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, int ability, void *server_context, int cmd_time) {
    PlayerState *p0 = &local_state.players[0];
    p0->yaw = yaw; p0->pitch = pitch;
    if (weapon_req >= 0 && weapon_req < MAX_WEAPONS) p0->current_weapon = weapon_req;
    
    float rad = yaw * 0.01745f;
    float wx = sinf(rad) * fwd + cosf(rad) * str;
    float wz = -cosf(rad) * fwd + sinf(rad) * str; // Fixed Z axis sign
    
    accelerate(p0, wx, wz, MAX_SPEED, ACCEL);
    
    if (jump && p0->on_ground) { 
        // Slide Jump Logic
        float speed = sqrtf(p0->vx*p0->vx + p0->vz*p0->vz);
        if (p0->crouching && speed > 0.5f && !was_holding_jump) {
             float boost = 1.4f;
             p0->vx *= boost; p0->vz *= boost;
        }
        p0->vy = JUMP_FORCE; 
        p0->on_ground = 0; 
    }
    p0->crouching = crouch;
    was_holding_jump = jump;
    
    update_entity(p0, 0.016f, server_context, cmd_time);
    update_weapons(p0, shoot, reload, ability);
    
    // Bots
    for(int i=1; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active) continue;
        
        float b_fwd=0, b_yaw=p->yaw; int b_btns=0;
        bot_think(i, local_state.players, &b_fwd, &b_yaw, &b_btns);
        
        p->yaw = b_yaw;
        float brad = b_yaw * 0.01745f;
        float bx = sinf(brad) * b_fwd;
        float bz = cosf(brad) * b_fwd; // Bots move simple
        
        accelerate(p, bx, bz, MAX_SPEED, ACCEL);
        if ((b_btns & BTN_JUMP) && p->on_ground) p->vy = JUMP_FORCE;
        
        update_entity(p, 0.016f, server_context, cmd_time);
        update_weapons(p, (b_btns & BTN_ATTACK), 0, 0);
    }
    
    update_projectiles();
}

static void local_init_match(int num_players, int mode) {
    memset(&local_state, 0, sizeof(ServerState));
    local_state.game_mode = mode;
    local_state.players[0].active = 1;
    phys_respawn(&local_state.players[0], 0);
    local_state.players[0].current_weapon = WPN_SNIPER; 
    local_state.players[0].ammo[WPN_SNIPER] = 100;
    
    for(int i=1; i<num_players; i++) {
        local_state.players[i].active = 1;
        phys_respawn(&local_state.players[i], i*100);
        init_genome(&local_state.players[i].brain);
    }
}

#endif
