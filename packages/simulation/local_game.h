#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "../common/protocol.h"
#include "../common/physics.h"
#include "../rts/entity_behaviors.h"
#include <string.h>

ServerState local_state;
int was_holding_jump = 0;

#define MAX_CITY_NPCS 128
#define NPC_TEAM_NEUTRAL 0

typedef enum { NPC_IDLE = 0, NPC_WANDER, NPC_FLEE, NPC_CHASE, NPC_PATROL, NPC_GUARD } CityNpcBrain;

typedef struct {
    int active;
    int type;
    int state;
    int scene_id;
    int team_id;
    float x, y, z;
    float vx, vz;
    float yaw;
    float hp;
    float hp_max;
    float leash_x, leash_z;
    int home_district;
    int spawn_zone;
    int last_hit_by_client;
    int last_hit_by_team;
    unsigned int last_hit_time_ms;
    unsigned int think_tick;
    unsigned int action_tick;
    unsigned int respawn_time_ms;
} CityNpc;

typedef struct {
    float x;
    float z;
    float radius;
    int scene_id;
    int district_tag;
    int max_alive;
    unsigned int respawn_delay_ms;
} NpcSpawnZone;

typedef struct {
    int base_xp;
    int assist_xp;
    float team_share_pct;
    float xp_radius;
} CreepReward;

typedef struct {
    CityNpc npcs[MAX_CITY_NPCS];
    unsigned int seed;
} CityNpcManager;

CityNpcManager city_npcs;

static const NpcSpawnZone NPC_SPAWN_ZONES[] = {
    { 0.0f,  360.0f, 170.0f, SCENE_STADIUM, 0, 4, 3500 },
    { 0.0f, -360.0f, 170.0f, SCENE_STADIUM, 3, 4, 3500 },
    { 360.0f,  0.0f, 170.0f, SCENE_STADIUM, 1, 4, 3200 },
    {-360.0f,  0.0f, 170.0f, SCENE_STADIUM, 4, 4, 3200 },
    { 640.0f,  640.0f, 260.0f, SCENE_CITY, 0, 8, 5500 },
    {-640.0f,  640.0f, 260.0f, SCENE_CITY, 1, 8, 5500 },
    { 640.0f, -640.0f, 260.0f, SCENE_CITY, 2, 8, 5200 },
    {-640.0f, -640.0f, 260.0f, SCENE_CITY, 3, 8, 5000 },
    { 1200.0f, 0.0f, 320.0f, SCENE_CITY, 4, 10, 6200 }
};

static unsigned int city_hash2(int a, int b, unsigned int seed) {
    unsigned int x = (unsigned int)(a * 73856093u) ^ (unsigned int)(b * 19349663u) ^ seed;
    x ^= x >> 13;
    x *= 1274126177u;
    x ^= x >> 16;
    return x;
}

static int city_district_at(float x, float z) {
    float pitch = CITY_BLOCK_SIZE + CITY_ROAD_SIZE;
    int gx = (int)floorf(x / pitch);
    int gz = (int)floorf(z / pitch);
    return (int)(city_hash2(gx, gz, 1337u) % CITY_DISTRICTS);
}

static int city_point_blocked(float x, float y, float z) {
    for (int i = 1; i < map_count; i++) {
        Box b = map_geo[i];
        if (fabsf(x - b.x) < b.w * 0.5f + 2.0f && fabsf(z - b.z) < b.d * 0.5f + 2.0f && y < b.y + b.h * 0.5f) return 1;
    }
    return 0;
}

static int city_is_hostile_type(int type) {
    return type == ENT_CULTIST || type == ENT_GOBLIN || type == ENT_ORC || type == ENT_PILLAGER_MARAUDER || type == ENT_PILLAGER_DESTROYER || type == ENT_PILLAGER_CORRUPTOR;
}

static CreepReward creep_reward_for_type(int type) {
    CreepReward r = {10, 3, 0.5f, 110.0f};
    if (type == ENT_PEASANT) r = (CreepReward){5, 1, 0.4f, 100.0f};
    else if (type == ENT_GOBLIN) r = (CreepReward){12, 3, 0.5f, 115.0f};
    else if (type == ENT_CULTIST) r = (CreepReward){15, 4, 0.5f, 120.0f};
    else if (type == ENT_GUARD) r = (CreepReward){18, 5, 0.5f, 120.0f};
    else if (type == ENT_ORC) r = (CreepReward){30, 7, 0.5f, 125.0f};
    else if (type == ENT_PILLAGER_MARAUDER) r = (CreepReward){35, 8, 0.5f, 130.0f};
    else if (type == ENT_PILLAGER_DESTROYER) r = (CreepReward){55, 10, 0.5f, 135.0f};
    else if (type == ENT_PILLAGER_CORRUPTOR) r = (CreepReward){45, 9, 0.5f, 130.0f};
    return r;
}

static int xp_needed_for_level(int level) {
    return 50 + (level - 1) * 25;
}

static int city_pick_npc_type(int district, unsigned int roll) {
    int bucket = (int)(roll % 100);
    if (district == 0) {
        if (bucket < 24) return ENT_GUARD;
        if (bucket < 36) return ENT_HUNTER;
        if (bucket < 50) return ENT_PEASANT;
        if (bucket < 64) return ENT_GOBLIN;
        if (bucket < 76) return ENT_CULTIST;
        if (bucket < 88) return ENT_ORC;
        return ENT_PILLAGER_MARAUDER;
    }
    if (district == 3) {
        if (bucket < 8) return ENT_PEASANT;
        if (bucket < 24) return ENT_GOBLIN;
        if (bucket < 42) return ENT_CULTIST;
        if (bucket < 58) return ENT_ORC;
        if (bucket < 78) return ENT_PILLAGER_MARAUDER;
        return ENT_PILLAGER_DESTROYER;
    }
    if (district == 4) {
        if (bucket < 12) return ENT_GUARD;
        if (bucket < 24) return ENT_HUNTER;
        if (bucket < 44) return ENT_CULTIST;
        if (bucket < 58) return ENT_GOBLIN;
        if (bucket < 72) return ENT_ORC;
        if (bucket < 88) return ENT_PILLAGER_MARAUDER;
        return ENT_PILLAGER_CORRUPTOR;
    }
    if (bucket < 20) return ENT_PEASANT;
    if (bucket < 36) return ENT_GUARD;
    if (bucket < 46) return ENT_HUNTER;
    if (bucket < 64) return ENT_GOBLIN;
    if (bucket < 78) return ENT_CULTIST;
    if (bucket < 88) return ENT_ORC;
    if (bucket < 96) return ENT_PILLAGER_MARAUDER;
    return ENT_PILLAGER_CORRUPTOR;
}

static void city_reset_npcs() {
    memset(&city_npcs, 0, sizeof(city_npcs));
    city_npcs.seed = 0xC17BEEFu;
}

static void award_player_xp(PlayerState *p, int xp_award) {
    if (!p || !p->active || xp_award <= 0) return;
    p->xp += xp_award;
    p->accumulated_reward += (float)xp_award;
    while (p->xp >= p->xp_to_next) {
        p->xp -= p->xp_to_next;
        p->level++;
        p->xp_to_next = xp_needed_for_level(p->level);
        p->health = 100;
        p->shield = 100;
        p->storm_charges += 1;
    }
}

static void award_creep_xp(const CityNpc *n, int killer_client_id, int killer_team_id) {
    if (!n || killer_client_id < 0 || killer_client_id >= MAX_CLIENTS || killer_team_id <= 0) return;
    CreepReward rw = creep_reward_for_type(n->type);
    for (int i = 0; i < MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active || p->scene_id != n->scene_id) continue;
        if (p->team_id != killer_team_id) continue;
        float dx = p->x - n->x;
        float dz = p->z - n->z;
        if ((dx * dx + dz * dz) > rw.xp_radius * rw.xp_radius) continue;
        if (i == killer_client_id) award_player_xp(p, rw.base_xp);
        else award_player_xp(p, (int)floorf((float)rw.base_xp * rw.team_share_pct));
    }
}

static void city_spawn_npc(CityNpc *n, const NpcSpawnZone *zone, unsigned int now_tick, unsigned int h) {
    if (!n || !zone) return;
    float ang = (float)(h % 6283) * 0.001f;
    float dist = 20.0f + (float)((h >> 8) % (int)(zone->radius));
    float sx = zone->x + sinf(ang) * dist;
    float sz = zone->z + cosf(ang) * dist;
    float sy = 2.0f;
    for (int tries = 0; tries < 6 && city_point_blocked(sx, sy, sz); tries++) {
        sx += sinf(ang + tries * 0.7f) * 12.0f;
        sz += cosf(ang + tries * 0.7f) * 12.0f;
    }
    if (city_point_blocked(sx, sy, sz)) return;

    n->active = 1;
    n->scene_id = zone->scene_id;
    n->x = sx; n->y = sy; n->z = sz;
    n->vx = 0.0f; n->vz = 0.0f;
    n->yaw = ang * (180.0f / 3.14159f);
    n->home_district = zone->scene_id == SCENE_CITY ? city_district_at(sx, sz) : zone->district_tag;
    n->type = city_pick_npc_type(n->home_district, h);
    n->state = city_is_hostile_type(n->type) ? NPC_CHASE : NPC_WANDER;
    if (n->type == ENT_GUARD) n->state = NPC_GUARD;
    n->hp_max = (float)get_entity_stats((EntityType)n->type)->hp_max;
    n->hp = n->hp_max;
    n->leash_x = sx; n->leash_z = sz;
    n->spawn_zone = (int)(zone - NPC_SPAWN_ZONES);
    n->team_id = NPC_TEAM_NEUTRAL;
    n->last_hit_by_client = -1;
    n->last_hit_by_team = 0;
    n->last_hit_time_ms = 0;
    n->think_tick = now_tick + (h % 20);
    n->action_tick = now_tick;
    n->respawn_time_ms = 0;
}

static void city_npc_update(unsigned int now_tick) {
    PlayerState *focus = &local_state.players[0];
    if (!focus->active) return;

    for (int z = 0; z < (int)(sizeof(NPC_SPAWN_ZONES)/sizeof(NPC_SPAWN_ZONES[0])); z++) {
        const NpcSpawnZone *zone = &NPC_SPAWN_ZONES[z];
        if (zone->scene_id != local_state.scene_id) continue;

        int alive = 0;
        for (int i = 0; i < MAX_CITY_NPCS; i++) {
            CityNpc *n = &city_npcs.npcs[i];
            if (n->active && n->spawn_zone == z && n->scene_id == zone->scene_id) alive++;
        }

        if (alive < zone->max_alive) {
            for (int i = 0; i < MAX_CITY_NPCS; i++) {
                CityNpc *n = &city_npcs.npcs[i];
                if (n->active) continue;
                if (n->respawn_time_ms != 0 && now_tick < n->respawn_time_ms) continue;
                unsigned int h = city_hash2(i * 13 + z * 79, (int)now_tick, city_npcs.seed);
                city_spawn_npc(n, zone, now_tick, h);
                if (n->active) break;
            }
        }
    }

    for (int i = 0; i < MAX_CITY_NPCS; i++) {
        CityNpc *n = &city_npcs.npcs[i];
        if (!n->active) continue;
        if (n->scene_id != local_state.scene_id) continue;

        float best_dx = 0.0f, best_dz = 0.0f;
        float best_d = 1e9f;
        PlayerState *target = NULL;
        for (int p = 0; p < MAX_CLIENTS; p++) {
            PlayerState *pl = &local_state.players[p];
            if (!pl->active || pl->state == STATE_DEAD || pl->scene_id != n->scene_id) continue;
            float dx = pl->x - n->x, dz = pl->z - n->z;
            float d = sqrtf(dx*dx + dz*dz);
            if (d < best_d) { best_d = d; best_dx = dx; best_dz = dz; target = pl; }
        }
        if (!target) continue;

        if (n->think_tick <= now_tick) {
            n->think_tick = now_tick + 14 + (i % 9);
            if (n->type == ENT_PEASANT) n->state = (target->in_shoot && best_d < 90.0f) ? NPC_FLEE : NPC_WANDER;
            else if (n->type == ENT_GUARD) n->state = (best_d < 120.0f) ? NPC_GUARD : NPC_PATROL;
            else if (n->type == ENT_HUNTER) n->state = (best_d < 190.0f) ? NPC_CHASE : NPC_WANDER;
            else if (city_is_hostile_type(n->type)) n->state = (best_d < 220.0f) ? NPC_CHASE : NPC_WANDER;
        }

        float tx = n->leash_x, tz = n->leash_z, speed = 0.26f;
        if (n->state == NPC_FLEE) { tx = n->x - best_dx; tz = n->z - best_dz; speed = 0.58f; }
        else if (n->state == NPC_CHASE || n->state == NPC_GUARD) {
            tx = target->x; tz = target->z;
            speed = (n->type == ENT_GOBLIN) ? 0.76f : (n->type == ENT_ORC ? 0.38f : 0.48f);
            if (n->type == ENT_HUNTER && best_d < 40.0f) { tx = n->x - best_dx; tz = n->z - best_dz; speed = 0.52f; }
            if (n->type == ENT_PILLAGER_MARAUDER) {
                tx += -best_dz * 0.35f;
                tz += best_dx * 0.35f;
            }
        } else if (n->state == NPC_PATROL || n->state == NPC_WANDER) {
            if ((now_tick + i) % 40 == 0) {
                float wa = (float)((city_hash2(i, now_tick, city_npcs.seed) % 6283)) * 0.001f;
                n->leash_x += sinf(wa) * 16.0f;
                n->leash_z += cosf(wa) * 16.0f;
            }
            tx = n->leash_x; tz = n->leash_z;
            speed = (n->type == ENT_PEASANT) ? 0.20f : 0.28f;
        }

        float dx = tx - n->x, dz = tz - n->z;
        float dl = sqrtf(dx * dx + dz * dz);
        if (dl > 0.01f) { dx /= dl; dz /= dl; }
        n->vx = n->vx * 0.82f + dx * speed * 0.18f;
        n->vz = n->vz * 0.82f + dz * speed * 0.18f;

        if (n->type == ENT_GOBLIN) {
            float zig = sinf((now_tick + i * 17) * 0.07f) * 0.18f;
            n->vx += -dz * zig;
            n->vz += dx * zig;
        }

        float dirs[4][2] = {{n->vx,n->vz},{n->vz,-n->vx},{-n->vx,n->vz},{-n->vx,-n->vz}};
        for (int d = 0; d < 4; d++) {
            float sx = n->x + dirs[d][0] * 6.0f;
            float sz = n->z + dirs[d][1] * 6.0f;
            if (!city_point_blocked(sx, 2.0f, sz)) { n->vx = dirs[d][0]; n->vz = dirs[d][1]; break; }
        }

        n->x += n->vx;
        n->z += n->vz;
        n->yaw = atan2f(n->vx, n->vz) * (180.0f / 3.14159f);

        if (best_d < ((n->type == ENT_ORC || n->type == ENT_PILLAGER_DESTROYER) ? 9.0f : 6.0f) && (now_tick % 20 == 0)) {
            int dmg = (n->type == ENT_ORC) ? 10 : (n->type == ENT_PILLAGER_DESTROYER ? 14 : 5);
            if (target->health > 1) target->health -= dmg;
        }
        if (n->type == ENT_HUNTER && best_d < 150.0f && best_d > 22.0f && (now_tick % 34 == 0) && target->health > 1) {
            target->health -= 6;
        }
        if (n->type == ENT_PILLAGER_CORRUPTOR && best_d < 24.0f && (now_tick % 18 == 0) && target->health > 1) {
            target->health -= 2;
        }
    }
}

static int city_npc_apply_damage(float x, float y, float z, int damage, int attacker_id, unsigned int now_ms, int scene_id) {
    int best = -1;
    float best_d2 = 16.0f;
    for (int i = 0; i < MAX_CITY_NPCS; i++) {
        CityNpc *n = &city_npcs.npcs[i];
        if (!n->active || n->scene_id != scene_id) continue;
        float dx = n->x - x;
        float dy = (n->y + 2.5f) - y;
        float dz = n->z - z;
        float d2 = dx * dx + dy * dy + dz * dz;
        if (d2 < best_d2) { best = i; best_d2 = d2; }
    }
    if (best < 0) return 0;

    CityNpc *n = &city_npcs.npcs[best];
    n->hp -= (float)damage;
    n->last_hit_by_client = attacker_id;
    n->last_hit_by_team = (attacker_id >= 0 && attacker_id < MAX_CLIENTS) ? local_state.players[attacker_id].team_id : 0;
    n->last_hit_time_ms = now_ms;
    if (n->hp <= 0.0f) {
        n->active = 0;
        n->respawn_time_ms = now_ms + NPC_SPAWN_ZONES[n->spawn_zone].respawn_delay_ms;
        int killer = (now_ms - n->last_hit_time_ms <= 5000) ? n->last_hit_by_client : -1;
        int kteam = (killer >= 0 && killer < MAX_CLIENTS) ? local_state.players[killer].team_id : 0;
        award_creep_xp(n, killer, kteam);
    }
    return 1;
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
    if (scene_id == SCENE_CITY || scene_id == SCENE_STADIUM) city_reset_npcs();
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
            if (p->active) {
                if (city_npc_apply_damage(p->x, p->y, p->z, p->damage, p->owner_id, now_ms, p->scene_id)) {
                    p->active = 0;
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
    if (local_state.scene_id == SCENE_CITY || local_state.scene_id == SCENE_STADIUM) {
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
    local_state.players[0].team_id = 1;
    local_state.players[0].level = 1;
    local_state.players[0].xp = 0;
    local_state.players[0].xp_to_next = xp_needed_for_level(1);
    phys_respawn(&local_state.players[0], 0);
    for(int i=1; i<num_players; i++) {
        local_state.players[i].active = 1;
        local_state.players[i].scene_id = local_state.scene_id;
        local_state.players[i].team_id = (mode == MODE_TDM) ? ((i % 2) ? 2 : 1) : 1;
        local_state.players[i].level = 1;
        local_state.players[i].xp = 0;
        local_state.players[i].xp_to_next = xp_needed_for_level(1);
        phys_respawn(&local_state.players[i], i*100);
        init_genome(&local_state.players[i].brain);
    }
}
#endif
