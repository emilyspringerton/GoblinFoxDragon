#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "../common/protocol.h"
#include "../common/physics.h"
#include "../rts/entity_behaviors.h"
#include <string.h>

ServerState local_state;
int was_holding_jump = 0;

#define MAX_CITY_NPCS 96

typedef enum { NPC_IDLE = 0, NPC_WANDER, NPC_FLEE, NPC_CHASE, NPC_PATROL, NPC_GUARD } CityNpcBrain;

typedef struct {
    int active;
    int type;
    int state;
    float x, y, z;
    float vx, vz;
    float yaw;
    float hp;
    float leash_x, leash_z;
    int home_district;
    unsigned int think_tick;
    unsigned int action_tick;
} CityNpc;

typedef struct {
    CityNpc npcs[MAX_CITY_NPCS];
    unsigned int seed;
    unsigned int last_spawn_tick;
} CityNpcManager;

CityNpcManager city_npcs;

static inline unsigned int city_hash2(int a, int b, unsigned int seed) {
    unsigned int x = (unsigned int)(a * 73856093u) ^ (unsigned int)(b * 19349663u) ^ seed;
    x ^= x >> 13;
    x *= 1274126177u;
    x ^= x >> 16;
    return x;
}

static inline int city_district_at(float x, float z) {
    float pitch = CITY_BLOCK_SIZE + CITY_ROAD_SIZE;
    int gx = (int)floorf(x / pitch);
    int gz = (int)floorf(z / pitch);
    return (int)(city_hash2(gx, gz, 1337u) % CITY_DISTRICTS);
}

static inline int city_point_blocked(float x, float y, float z) {
    for (int i = 1; i < map_count; i++) {
        Box b = map_geo[i];
        if (fabsf(x - b.x) < b.w * 0.5f + 2.0f && fabsf(z - b.z) < b.d * 0.5f + 2.0f && y < b.y + b.h * 0.5f) {
            return 1;
        }
    }
    return 0;
}

static inline int city_is_hostile_type(int type) {
    return type == ENT_CULTIST || type == ENT_GOBLIN || type == ENT_ORC || type == ENT_PILLAGER_MARAUDER || type == ENT_PILLAGER_DESTROYER || type == ENT_PILLAGER_CORRUPTOR;
}

static inline int city_pick_npc_type(int district, unsigned int roll) {
    int bucket = (int)(roll % 100);
    if (district == 0) {
        if (bucket < 26) return ENT_PEASANT;
        if (bucket < 46) return ENT_GUARD;
        if (bucket < 56) return ENT_HUNTER;
        if (bucket < 70) return ENT_GOBLIN;
        if (bucket < 82) return ENT_CULTIST;
        if (bucket < 90) return ENT_ORC;
        if (bucket < 96) return ENT_PILLAGER_MARAUDER;
        return ENT_PILLAGER_CORRUPTOR;
    }
    if (district == 3) {
        if (bucket < 10) return ENT_PEASANT;
        if (bucket < 18) return ENT_GUARD;
        if (bucket < 28) return ENT_HUNTER;
        if (bucket < 52) return ENT_GOBLIN;
        if (bucket < 68) return ENT_CULTIST;
        if (bucket < 82) return ENT_ORC;
        if (bucket < 93) return ENT_PILLAGER_MARAUDER;
        return ENT_PILLAGER_DESTROYER;
    }
    if (district == 4) {
        if (bucket < 14) return ENT_PEASANT;
        if (bucket < 26) return ENT_GUARD;
        if (bucket < 34) return ENT_HUNTER;
        if (bucket < 52) return ENT_CULTIST;
        if (bucket < 68) return ENT_GOBLIN;
        if (bucket < 80) return ENT_ORC;
        if (bucket < 90) return ENT_PILLAGER_MARAUDER;
        return ENT_PILLAGER_CORRUPTOR;
    }
    if (bucket < 22) return ENT_PEASANT;
    if (bucket < 36) return ENT_GUARD;
    if (bucket < 48) return ENT_HUNTER;
    if (bucket < 64) return ENT_GOBLIN;
    if (bucket < 76) return ENT_CULTIST;
    if (bucket < 86) return ENT_ORC;
    if (bucket < 94) return ENT_PILLAGER_MARAUDER;
    if (bucket < 98) return ENT_PILLAGER_DESTROYER;
    return ENT_PILLAGER_CORRUPTOR;
}

static inline void city_reset_npcs() {
    memset(&city_npcs, 0, sizeof(city_npcs));
    city_npcs.seed = 0xC17BEEFu;
}

static inline int city_target_population(int district) {
    if (district == 0) return 44;
    if (district == 3) return 38;
    if (district == 4) return 34;
    return 30;
}

static void city_npc_update(unsigned int now_tick) {
    PlayerState *focus = &local_state.players[0];
    if (!focus->active || focus->scene_id != SCENE_CITY) return;
    phys_set_scene(SCENE_CITY);

    int player_district = city_district_at(focus->x, focus->z);
    int target_pop = city_target_population(player_district);
    int alive = 0;
    for (int i = 0; i < MAX_CITY_NPCS; i++) {
        if (city_npcs.npcs[i].active) alive++;
    }

    if (alive < target_pop && now_tick - city_npcs.last_spawn_tick > 130) {
        city_npcs.last_spawn_tick = now_tick;
        for (int i = 0; i < MAX_CITY_NPCS; i++) {
            CityNpc *n = &city_npcs.npcs[i];
            if (n->active) continue;
            unsigned int h = city_hash2((int)now_tick + i * 31, i * 17, city_npcs.seed);
            float ang = (float)(h % 6283) * 0.001f;
            float dist = 80.0f + (float)((h >> 8) % 120);
            float sx = focus->x + sinf(ang) * dist;
            float sz = focus->z + cosf(ang) * dist;
            float sy = 2.0f;
            int ok = 0;
            for (int tries = 0; tries < 6; tries++) {
                if (!city_point_blocked(sx, sy, sz)) { ok = 1; break; }
                sx += sinf(ang + tries) * 10.0f;
                sz += cosf(ang + tries) * 10.0f;
            }
            if (!ok) continue;

            n->active = 1;
            n->x = sx; n->y = sy; n->z = sz;
            n->vx = 0.0f; n->vz = 0.0f;
            n->yaw = ang * (180.0f / 3.14159f);
            n->home_district = city_district_at(sx, sz);
            n->type = city_pick_npc_type(n->home_district, h);
            n->state = city_is_hostile_type(n->type) ? NPC_CHASE : NPC_WANDER;
            if (n->type == ENT_GUARD) n->state = NPC_PATROL;
            n->hp = (float)get_entity_stats((EntityType)n->type)->hp_max;
            n->leash_x = sx; n->leash_z = sz;
            n->think_tick = now_tick + (h % 20);
            n->action_tick = now_tick;
            break;
        }
    }

    for (int i = 0; i < MAX_CITY_NPCS; i++) {
        CityNpc *n = &city_npcs.npcs[i];
        if (!n->active) continue;
        float pdx = focus->x - n->x;
        float pdz = focus->z - n->z;
        float pd = sqrtf(pdx * pdx + pdz * pdz);
        if (pd > 350.0f) { n->active = 0; continue; }

        if (n->think_tick <= now_tick) {
            n->think_tick = now_tick + 15 + (i % 11);
            if (n->type == ENT_PEASANT) {
                n->state = (focus->in_shoot && pd < 80.0f) ? NPC_FLEE : NPC_WANDER;
            } else if (n->type == ENT_GUARD) {
                n->state = (pd < 90.0f && city_is_hostile_type(ENT_GOBLIN)) ? NPC_GUARD : NPC_PATROL;
            } else if (n->type == ENT_HUNTER) {
                n->state = (pd < 120.0f) ? NPC_CHASE : NPC_WANDER;
            } else if (n->type == ENT_CULTIST) {
                n->state = (pd < 130.0f) ? NPC_CHASE : NPC_WANDER;
                if (now_tick - n->action_tick > 600 && pd < 170.0f) {
                    n->action_tick = now_tick;
                    for (int j = 0; j < MAX_CITY_NPCS; j++) {
                        if (!city_npcs.npcs[j].active) {
                            city_npcs.npcs[j] = *n;
                            city_npcs.npcs[j].type = ENT_GOBLIN;
                            city_npcs.npcs[j].x += 4.0f;
                            city_npcs.npcs[j].z -= 4.0f;
                            city_npcs.npcs[j].hp = 40.0f;
                            break;
                        }
                    }
                }
            }
        }

        float tx = n->leash_x;
        float tz = n->leash_z;
        float speed = 0.35f;

        if (n->state == NPC_FLEE) { tx = n->x - pdx; tz = n->z - pdz; speed = 0.55f; }
        else if (n->state == NPC_CHASE || n->state == NPC_GUARD) { tx = focus->x; tz = focus->z; speed = (n->type == ENT_GOBLIN) ? 0.72f : 0.48f; }
        else if (n->state == NPC_PATROL || n->state == NPC_WANDER) {
            if ((now_tick + i) % 40 == 0) {
                float wa = (float)((city_hash2(i, now_tick, city_npcs.seed) % 6283)) * 0.001f;
                n->leash_x += sinf(wa) * 18.0f;
                n->leash_z += cosf(wa) * 18.0f;
            }
            tx = n->leash_x; tz = n->leash_z;
            speed = (n->type == ENT_PEASANT) ? 0.24f : 0.30f;
        }

        float dx = tx - n->x;
        float dz = tz - n->z;
        float dl = sqrtf(dx * dx + dz * dz);
        if (dl > 0.01f) {
            dx /= dl; dz /= dl;
            n->vx = n->vx * 0.8f + dx * speed * 0.2f;
            n->vz = n->vz * 0.8f + dz * speed * 0.2f;
        }

        float best_x = n->vx;
        float best_z = n->vz;
        float dirs[4][2] = {{n->vx,n->vz},{n->vz,-n->vx},{-n->vx,n->vz},{-n->vx,-n->vz}};
        for (int d = 0; d < 4; d++) {
            float sx = n->x + dirs[d][0] * 6.0f;
            float sz = n->z + dirs[d][1] * 6.0f;
            if (!city_point_blocked(sx, 2.0f, sz)) { best_x = dirs[d][0]; best_z = dirs[d][1]; break; }
        }
        n->vx = best_x;
        n->vz = best_z;
        n->x += n->vx;
        n->z += n->vz;
        n->yaw = atan2f(n->vx, n->vz) * (180.0f / 3.14159f);

        if (n->type == ENT_PILLAGER_CORRUPTOR && pd < 22.0f && (now_tick % 18 == 0) && focus->health > 1) {
            focus->health -= 1;
        }
    }
}

void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, int use, int bike, int ability, void *server_context, unsigned int cmd_time);
void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time);
void local_init_match(int num_players, int mode);

float rand_weight() { return ((float)(rand()%2000)/1000.0f) - 1.0f; } 
float rand_pos() { return ((float)(rand()%1000)/1000.0f); } 

void init_genome(BotGenome *g) {
    g->version = 1;
    g->w_aggro = 0.5f + rand_weight() * 0.5f;
    g->w_strafe = rand_weight();
    g->w_jump = 0.05f + rand_pos() * 0.1f; 
    g->w_slide = 0.01f + rand_pos() * 0.05f;
    g->w_turret = 5.0f + rand_pos() * 10.0f;
    g->w_repel = 1.0f + rand_pos();
}

void evolve_bot(PlayerState *loser, PlayerState *winner) {
    loser->brain = winner->brain;
    loser->brain.w_aggro += rand_weight() * 0.1f;
    loser->brain.w_strafe += rand_weight() * 0.1f;
    loser->brain.w_jump += rand_weight() * 0.01f;
    loser->brain.w_slide += rand_weight() * 0.01f;
}

PlayerState* get_best_bot() {
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

static inline void scene_load(int scene_id) {
    local_state.scene_id = scene_id;
    phys_set_scene(scene_id);
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (!local_state.players[i].active) continue;
        local_state.players[i].scene_id = scene_id;
        scene_force_spawn(&local_state.players[i]);
    }
    if (scene_id == SCENE_CITY) city_reset_npcs();
}

static inline void scene_request_transition(int scene_id) {
    if (local_state.transition_timer > 0) return;
    local_state.pending_scene = scene_id;
    local_state.transition_timer = 12;
}

static inline void scene_tick_transition() {
    if (local_state.transition_timer <= 0) return;
    local_state.transition_timer--;
    if (local_state.transition_timer == 0 && local_state.pending_scene >= 0) {
        scene_load(local_state.pending_scene);
        local_state.pending_scene = -1;
    }
}

// --- BOT AI ---
void bot_think(int bot_idx, PlayerState *players, float *out_fwd, float *out_yaw, int *out_buttons) {
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
        
        float turn_speed = (me->brain.w_turret > 1.0f) ? me->brain.w_turret : 10.0f;
        float diff = angle_diff(target_yaw, *out_yaw);
        if (diff > turn_speed) diff = turn_speed;
        if (diff < -turn_speed) diff = -turn_speed;
        *out_yaw += diff;
        
        *out_buttons |= BTN_ATTACK;
        
        if (min_dist > 15.0f) *out_fwd = me->brain.w_aggro;
        else if (min_dist < 5.0f) *out_fwd = -me->brain.w_aggro; 
        else *out_fwd = 0.2f; 
        
        *out_yaw += me->brain.w_strafe * 10.0f;
        if (me->on_ground && (rand()%1000 < (me->brain.w_jump * 1000.0f))) *out_buttons |= BTN_JUMP;
        if (me->on_ground && (rand()%1000 < (me->brain.w_slide * 1000.0f))) *out_buttons |= BTN_CROUCH;
        if (me->ammo[me->current_weapon] <= 0) *out_buttons |= BTN_RELOAD;
    } else {
        *out_yaw += 2.0f;
        *out_fwd = 0.5f;
    }
}

// --- UPDATE LOOP ---
void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time) {
    if (!p->active) return;
    if (p->state == STATE_DEAD) return;

    phys_set_scene(p->scene_id);

    if (cmd_time < p->stunned_until_ms) {
        p->in_fwd = 0.0f;
        p->in_strafe = 0.0f;
        p->in_jump = 0;
        p->in_shoot = 0;
        p->in_reload = 0;
        p->in_use = 0;
        p->in_bike = 0;
        p->in_ability = 0;
        p->vx = 0.0f;
        p->vz = 0.0f;
    }

    apply_friction(p);
    float g = p->in_vehicle ? ((p->vehicle_type == VEH_BIKE) ? BIKE_GRAVITY : BUGGY_GRAVITY) : ((p->in_jump) ? GRAVITY_FLOAT : GRAVITY_DROP);
    p->vy -= g; 
    p->y += p->vy;
    
    resolve_collision(p);
    p->x += p->vx;
    p->z += p->vz;

    if (p->recoil_anim > 0) p->recoil_anim -= 0.1f;
    if (p->recoil_anim < 0) p->recoil_anim = 0;
    if (p->hit_feedback > 0) p->hit_feedback--;

    update_weapons(p, local_state.players, local_state.projectiles, p->in_shoot > 0, p->in_reload > 0, p->in_ability > 0);
    scene_safety_check(p);
}

static void apply_projectile_damage(PlayerState *owner, PlayerState *target, int damage, unsigned int now_ms) {
    if (!target->active || target->state == STATE_DEAD) return;
    target->shield_regen_timer = SHIELD_REGEN_DELAY;
    if (target->shield > 0) {
        if (target->shield >= damage) { target->shield -= damage; damage = 0; }
        else { damage -= target->shield; target->shield = 0; }
    }
    target->health -= damage;
    if (target->health <= 0) {
        if (owner) { owner->kills++; owner->accumulated_reward += 500.0f; }
        target->deaths++;
        phys_respawn(target, now_ms);
    }

    if (now_ms >= target->stun_immune_until_ms) {
        unsigned int stun_end = now_ms + 100;
        if (stun_end > target->stunned_until_ms) target->stunned_until_ms = stun_end;
        target->stun_immune_until_ms = now_ms + 250;
    }
}

static void update_projectiles(unsigned int now_ms) {
    for (int i=0; i<MAX_PROJECTILES; i++) {
        Projectile *p = &local_state.projectiles[i];
        if (!p->active) continue;

        phys_set_scene(p->scene_id);

        float next_x = p->x + p->vx;
        float next_y = p->y + p->vy;
        float next_z = p->z + p->vz;

        float hit_x, hit_y, hit_z, nx, ny, nz;
        if (trace_map(p->x, p->y, p->z, next_x, next_y, next_z, &hit_x, &hit_y, &hit_z, &nx, &ny, &nz)) {
            if (p->bounces_left > 0) {
                reflect_vector(&p->vx, &p->vy, &p->vz, nx, ny, nz);
                p->x = hit_x; p->y = hit_y; p->z = hit_z;
                p->bounces_left--;
            } else {
                p->active = 0;
            }
        } else {
            p->x = next_x; p->y = next_y; p->z = next_z;
        }

        if (p->active) {
            for (int t = 0; t < MAX_CLIENTS; t++) {
                PlayerState *target = &local_state.players[t];
                if (!target->active || target->state == STATE_DEAD) continue;
                if (t == p->owner_id) continue;
                if (target->scene_id != p->scene_id) continue;
                float dx = target->x - p->x;
                float dy = (target->y + EYE_HEIGHT) - p->y;
                float dz = target->z - p->z;
                float dist_sq = dx * dx + dy * dy + dz * dz;
                if (dist_sq < 4.0f) {
                    PlayerState *owner = NULL;
                    if (p->owner_id >= 0 && p->owner_id < MAX_CLIENTS) {
                        owner = &local_state.players[p->owner_id];
                    }
                    apply_projectile_damage(owner, target, p->damage, now_ms);
                    p->active = 0;
                    break;
                }
            }
        }

        if (p->x > 4000 || p->x < -4000 || p->z > 4000 || p->z < -4000 || p->y > 2000) p->active = 0;
    }
}

void local_update(float fwd, float str, float yaw, float pitch, int shoot, int weapon_req, int jump, int crouch, int reload, int use, int bike, int ability, void *server_context, unsigned int cmd_time) {
    PlayerState *p0 = &local_state.players[0];
    scene_tick_transition();
    if (local_state.transition_timer > 0) {
        fwd = 0.0f;
        str = 0.0f;
        shoot = 0;
        jump = 0;
        crouch = 0;
        reload = 0;
        ability = 0;
    }
    p0->yaw = yaw; p0->pitch = pitch;
    if (weapon_req >= 0 && weapon_req < MAX_WEAPONS) p0->current_weapon = weapon_req;
    p0->in_use = use;
    p0->in_bike = bike;
    float rad = yaw * 3.14159f / 180.0f;
    float wish_x = sinf(rad) * fwd + cosf(rad) * str;
    float wish_z = cosf(rad) * fwd - sinf(rad) * str;
    float max_spd = MAX_SPEED;
    float acc = ACCEL;

    if (p0->in_vehicle && p0->vehicle_type == VEH_BIKE) {
        wish_x = sinf(rad) * fwd;
        wish_z = cosf(rad) * fwd;

        float vmag = sqrtf(p0->vx * p0->vx + p0->vz * p0->vz);
        if (p0->bike_gear < 1) p0->bike_gear = 1;
        if (p0->bike_gear < BIKE_GEARS && vmag > BIKE_GEAR_MAX[p0->bike_gear] * 0.92f) p0->bike_gear++;
        if (p0->bike_gear > 1 && vmag < BIKE_GEAR_MAX[p0->bike_gear - 1] * 0.55f) p0->bike_gear--;

        max_spd = BIKE_GEAR_MAX[p0->bike_gear];
        acc = BIKE_GEAR_ACCEL[p0->bike_gear];
    } else if (p0->in_vehicle) {
        wish_x = sinf(rad) * fwd;
        wish_z = cosf(rad) * fwd;
        max_spd = BUGGY_MAX_SPEED;
        acc = BUGGY_ACCEL;
    }

    float wish_speed = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_speed > 1.0f) { wish_speed = 1.0f; wish_x/=wish_speed; wish_z/=wish_speed; }
    wish_speed *= max_spd;
    accelerate(p0, wish_x, wish_z, wish_speed, acc);
    int fresh_jump_press = (jump && !was_holding_jump);
    // --- PHASE 485: TUNED SLIDE JUMP ---
    if (jump && p0->on_ground) {
        float speed = sqrtf(p0->vx*p0->vx + p0->vz*p0->vz);
        if (p0->crouching && speed > 0.5f && fresh_jump_press) {
            float boost_mult = 1.0f + (0.25f / speed);
            if (boost_mult > 1.4f) boost_mult = 1.4f;
            if (boost_mult < 1.02f) boost_mult = 1.02f;
            p0->vx *= boost_mult;
            p0->vz *= boost_mult;
        }
        p0->y += 0.1f;
        p0->vy += JUMP_FORCE;
    }
    p0->in_shoot = shoot; p0->in_reload = reload; p0->crouching = crouch;
    p0->in_jump = jump; 
    p0->in_ability = ability;
    was_holding_jump = jump;
    
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active) continue;
        if (i > 0 && p->active && p->state != STATE_DEAD) {
            float b_fwd=0, b_yaw=p->yaw;
            int b_btns=0;
            bot_think(i, local_state.players, &b_fwd, &b_yaw, &b_btns);
            p->yaw = b_yaw;
            float brad = b_yaw * 3.14159f / 180.0f;
            float bx = sinf(brad) * b_fwd;
            float bz = cosf(brad) * b_fwd;
            accelerate(p, bx, bz, MAX_SPEED, ACCEL);
            p->in_shoot = (b_btns & BTN_ATTACK);
            p->in_jump = (b_btns & BTN_JUMP);
            p->in_reload = (b_btns & BTN_RELOAD);
            p->crouching = (b_btns & BTN_CROUCH);
            p->in_ability = 0;
            if ((b_btns & BTN_JUMP) && p->on_ground) { p->y += 0.1f; p->vy += JUMP_FORCE; }
        }
        phys_set_scene(p->scene_id);
        update_entity(p, 0.016f, server_context, cmd_time);
    }
    update_projectiles(cmd_time);
    if (local_state.scene_id == SCENE_CITY) {
        city_npc_update(cmd_time);
    }
}

void local_init_match(int num_players, int mode) {
    memset(&local_state, 0, sizeof(ServerState));
    local_state.game_mode = mode;
    local_state.scene_id = SCENE_GARAGE_OSAKA;
    local_state.pending_scene = -1;
    local_state.transition_timer = 0;
    phys_set_scene(local_state.scene_id);
    city_reset_npcs();
    local_state.players[0].active = 1;
    local_state.players[0].scene_id = local_state.scene_id;
    phys_respawn(&local_state.players[0], 0);
    for(int i=1; i<num_players; i++) {
        local_state.players[i].active = 1;
        local_state.players[i].scene_id = local_state.scene_id;
        phys_respawn(&local_state.players[i], i*100);
        init_genome(&local_state.players[i].brain);
    }
}
#endif
