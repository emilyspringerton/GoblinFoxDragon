#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "../common/protocol.h"
#include "../common/physics.h"
#include <string.h>

ServerState local_state;
int was_holding_jump = 0;

void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time);

// --- HELPER: SPAWN ENTITY ---
void spawn_entity(int type, float x, float y, float z) {
    for(int i=0; i<MAX_ENTITIES; i++) {
        if (!local_state.entities[i].active) {
            local_state.entities[i].active = 1;
            local_state.entities[i].id = i;
            local_state.entities[i].type = type;
            local_state.entities[i].x = x;
            local_state.entities[i].y = y;
            local_state.entities[i].z = z;
            local_state.entities[i].owner_id = -1;
            
            if (type == ENT_BUGGY) {
                local_state.entities[i].w = 4.0f; local_state.entities[i].h = 3.0f; local_state.entities[i].d = 6.0f;
            } else if (type == ENT_FLAG_RED || type == ENT_FLAG_BLUE) {
                local_state.entities[i].w = 1.0f; local_state.entities[i].h = 4.0f; local_state.entities[i].d = 1.0f;
            }
            return;
        }
    }
}

// --- INIT MATCH ---
void local_init_match(int num_players, int mode) {
    memset(&local_state, 0, sizeof(ServerState));
    local_state.game_mode = mode;
    local_state.players[0].active = 1;
    phys_respawn(&local_state.players[0], 0);
    
    // SPAWN ENTITIES
    // 1. Buggies
    spawn_entity(ENT_BUGGY, 100, 20, 100);
    spawn_entity(ENT_BUGGY, -100, 20, -100);
    spawn_entity(ENT_BUGGY, 0, 20, 200);
    
    // 2. Flags
    if (mode == MODE_CTF) {
        spawn_entity(ENT_FLAG_RED, 0, 5, 800);
        spawn_entity(ENT_FLAG_BLUE, 0, 5, -800);
    }
}

// --- INTERACTION LOGIC ---
void check_interactions(PlayerState *p) {
    if (p->interaction_cooldown > 0) p->interaction_cooldown--;
    
    if (p->in_use && p->interaction_cooldown == 0) {
        // EXIT VEHICLE
        if (p->in_vehicle != -1) {
            int eid = p->in_vehicle;
            local_state.entities[eid].owner_id = -1; // Free it
            local_state.entities[eid].x = p->x;      // Leave it here
            local_state.entities[eid].y = p->y;
            local_state.entities[eid].z = p->z;
            p->in_vehicle = -1;
            p->interaction_cooldown = 30;
            p->y += 4.0f; // Hop out top
            return;
        }
        
        // FIND NEAREST ENTITY
        int best_id = -1;
        float min_dist = INTERACT_RANGE;
        
        for(int i=0; i<MAX_ENTITIES; i++) {
            if (!local_state.entities[i].active) continue;
            float dx = local_state.entities[i].x - p->x;
            float dy = local_state.entities[i].y - p->y;
            float dz = local_state.entities[i].z - p->z;
            float d = sqrtf(dx*dx + dy*dy + dz*dz);
            if (d < min_dist) {
                min_dist = d;
                best_id = i;
            }
        }
        
        if (best_id != -1) {
            Entity *e = &local_state.entities[best_id];
            
            // ENTER BUGGY
            if (e->type == ENT_BUGGY && e->owner_id == -1) {
                p->in_vehicle = best_id;
                e->owner_id = p->id;
                p->interaction_cooldown = 30;
                // Snap player to car (or car to player, handled in update)
            }
            
            // PICKUP FLAG
            if ((e->type == ENT_FLAG_RED || e->type == ENT_FLAG_BLUE) && e->owner_id == -1) {
                p->carrying_flag = best_id;
                e->owner_id = p->id;
                p->interaction_cooldown = 30;
            }
        }
    }
}

// --- UPDATE LOOP ---
void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time) {
    if (!p->active || p->state == STATE_DEAD) return;

    check_interactions(p);
    apply_friction(p);
    
    float g = (p->in_vehicle != -1) ? BUGGY_GRAVITY : (p->in_jump ? GRAVITY_FLOAT : GRAVITY_DROP);
    p->vy -= g; 
    p->y += p->vy;
    
    resolve_collision(p, local_state.entities);
    p->x += p->vx;
    p->z += p->vz;
    
    // SYNC ENTITIES TO PLAYER
    if (p->in_vehicle != -1) {
        Entity *e = &local_state.entities[p->in_vehicle];
        e->x = p->x; e->y = p->y; e->z = p->z;
        e->yaw = p->yaw; // Or velocity based
    }
    if (p->carrying_flag != -1) {
        Entity *e = &local_state.entities[p->carrying_flag];
        e->x = p->x; e->y = p->y + 4.0f; e->z = p->z; // On back
    }

    if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
    if (p->recoil_anim < 0) p->recoil_anim = 0;
    if (p->hit_feedback > 0) p->hit_feedback--;

    update_weapons(p, local_state.players, p->in_shoot > 0, p->in_reload > 0);
}

void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, int use, void *server_context, unsigned int cmd_time) {
    
    PlayerState *p0 = &local_state.players[0];
    p0->yaw = yaw; p0->pitch = pitch;
    if (weapon_req >= 0 && weapon_req < MAX_WEAPONS) p0->current_weapon = weapon_req;
    
    float rad = yaw * 3.14159f / 180.0f;
    float wish_x = 0, wish_z = 0;
    float max_spd = MAX_SPEED; float acc = ACCEL;

    if (p0->in_vehicle != -1) {
        wish_x = sinf(rad) * fwd; wish_z = cosf(rad) * fwd;
        max_spd = BUGGY_MAX_SPEED; acc = BUGGY_ACCEL;
    } else {
        wish_x = sinf(rad) * fwd + cosf(rad) * str;
        wish_z = cosf(rad) * fwd - sinf(rad) * str;
    }
    
    float wish_speed = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_speed > 1.0f) { wish_speed = 1.0f; wish_x/=wish_speed; wish_z/=wish_speed; }
    wish_speed *= max_spd;
    
    accelerate(p0, wish_x, wish_z, wish_speed, acc);
    
    int fresh_jump_press = (jump && !was_holding_jump);
    if (jump && p0->on_ground) {
        float speed = sqrtf(p0->vx*p0->vx + p0->vz*p0->vz);
        if (p0->crouching && speed > 0.5f && fresh_jump_press) {
            float boost_mult = 1.0f + (0.25f / speed);
            if (boost_mult > 1.4f) boost_mult = 1.4f; 
            if (boost_mult < 1.02f) boost_mult = 1.02f;
            p0->vx *= boost_mult; p0->vz *= boost_mult;
        }
        p0->y += 0.1f; p0->vy += JUMP_FORCE;
    }
    
    p0->in_shoot = shoot; p0->in_reload = reload; p0->crouching = crouch; p0->in_jump = jump; p0->in_use = use;
    was_holding_jump = jump;
    
    update_entity(p0, 0.016f, server_context, cmd_time);
}
#endif
