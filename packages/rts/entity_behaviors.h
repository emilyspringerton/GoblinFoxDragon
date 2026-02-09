#ifndef ENTITY_BEHAVIORS_H
#define ENTITY_BEHAVIORS_H

#include <math.h>
#include <stdlib.h>
#include "../common/protocol.h"

// ============================================================================
// ENTITY TYPES & CONSTANTS
// ============================================================================

#define MAX_ENTITIES 512
#define MAX_STRUCTURES 128
#define MAX_TOWNS 64

#define CELL_SIZE 50.0f
#define AGGRO_CHECK_INTERVAL 15 // Frames between target acquisition

// Entity Types
typedef enum {
    ENT_NONE = 0,
    // Tier 1
    ENT_MILITIA = 1,
    ENT_SCOUT = 2,
    ENT_SWARMLING = 3,
    ENT_RAVAGER = 4,
    // Tier 2
    ENT_HEXBOUND = 5,
    ENT_BEHEMOTH = 6,
    ENT_SHADE = 7,
    ENT_PYROMANCER = 8,
    // Tier 3
    ENT_WARDEN = 9,
    ENT_TIDECALLER = 10,
    ENT_GOLEM = 11,
    ENT_VOIDREAVER = 12,
    // Tier 4
    ENT_ARCHON = 13,
    ENT_CHAOS = 14,
    ENT_WRAITH = 15,
    ENT_DRAGON = 16,
    // NPCs
    ENT_PEASANT = 50,
    ENT_GUARD = 51,
    ENT_HUNTER = 52,
    ENT_CULTIST = 53,
    ENT_GOBLIN = 54,
    ENT_ORC = 55,
    ENT_PILLAGER_MARAUDER = 56,
    ENT_PILLAGER_DESTROYER = 57,
    ENT_PILLAGER_CORRUPTOR = 58
} EntityType;

typedef enum {
    STRUCT_NONE = 0,
    STRUCT_OUTPOST = 1,
    STRUCT_MANAWELL = 2,
    STRUCT_CORRUPTION_SPIRE = 3,
    STRUCT_GROVE_HEART = 4,
    STRUCT_SIEGE_WORKSHOP = 5,
    STRUCT_SHADOW_SANCTUM = 6,
    STRUCT_INFERNO_TOWER = 7,
    STRUCT_NEXUS_CORE = 8
} StructureType;

typedef enum {
    TOWN_FRONTIER = 0,
    TOWN_HAMLET = 1,
    TOWN_ENCLAVE = 2,
    TOWN_BLIGHTED = 3
} TownType;

// Alignment
typedef enum {
    ALIGN_NEUTRAL = 0,
    ALIGN_PLAYER = 1,
    ALIGN_ENEMY = 2,
    ALIGN_CORRUPTED = 3
} Alignment;

// Behavior Modes
typedef enum {
    BEHAVIOR_IDLE = 0,
    BEHAVIOR_MOVE_TO_TARGET = 1,
    BEHAVIOR_ATTACK_TARGET = 2,
    BEHAVIOR_FLEE = 3,
    BEHAVIOR_PATROL = 4,
    BEHAVIOR_GUARD = 5,
    BEHAVIOR_SUPPORT = 6
} BehaviorMode;

// ============================================================================
// STAT TABLES (Exact Tuning)
// ============================================================================

typedef struct {
    EntityType type;
    int hp_max;
    float speed; // cells per second
    float aggro_range; // in cells
    int damage;
    float attack_speed; // attacks per second
    float attack_range; // in cells
    int is_ranged;
    int is_melee;
    int prefer_structures; // Ravager behavior
    int is_support; // Doesn't attack
    int is_flying; // Ignores terrain
} EntityStats;

static const EntityStats ENTITY_STAT_TABLE[] = {
    // type, hp, speed, aggro, dmg, atk_spd, atk_rng, ranged, melee, pref_struct, support, flying
    {ENT_MILITIA, 100, 0.5f, 5.0f, 10, 1.0f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_SCOUT, 50, 0.8f, 8.0f, 15, 0.66f, 6.0f, 1, 0, 0, 0, 0},
    {ENT_SWARMLING, 20, 1.0f, 3.0f, 5, 1.5f, 1.0f, 0, 1, 0, 0, 0},
    {ENT_RAVAGER, 150, 0.6f, 4.0f, 25, 0.8f, 1.5f, 0, 1, 1, 0, 0},
    {ENT_HEXBOUND, 80, 0.4f, 0.0f, 0, 0.0f, 0.0f, 0, 0, 0, 1, 1},
    {ENT_BEHEMOTH, 300, 0.2f, 6.0f, 8, 0.5f, 2.0f, 0, 1, 0, 0, 0},
    {ENT_SHADE, 60, 0.7f, 7.0f, 30, 1.2f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_PYROMANCER, 70, 0.5f, 6.0f, 40, 0.2f, 6.0f, 1, 0, 0, 0, 0},
    {ENT_WARDEN, 250, 0.3f, 5.0f, 12, 0.8f, 2.0f, 0, 1, 0, 0, 0},
    {ENT_TIDECALLER, 100, 0.4f, 0.0f, 0, 0.0f, 0.0f, 0, 0, 0, 1, 0},
    {ENT_GOLEM, 400, 0.3f, 5.0f, 20, 0.5f, 5.0f, 1, 0, 0, 0, 0},
    {ENT_VOIDREAVER, 150, 0.6f, 6.0f, 20, 1.0f, 2.0f, 0, 1, 0, 0, 0},
    {ENT_ARCHON, 500, 0.5f, 8.0f, 50, 1.5f, 3.0f, 0, 1, 0, 0, 1},
    {ENT_CHAOS, 100, 0.5f, 5.0f, 15, 1.0f, 2.0f, 0, 1, 0, 0, 0},
    {ENT_WRAITH, 300, 0.4f, 6.0f, 40, 0.5f, 2.5f, 0, 1, 0, 0, 0},
    {ENT_DRAGON, 800, 0.6f, 10.0f, 60, 0.33f, 8.0f, 1, 0, 0, 0, 1},
    // NPCs
    {ENT_PEASANT, 30, 0.4f, 3.0f, 5, 1.0f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_GUARD, 60, 0.4f, 4.0f, 12, 1.0f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_HUNTER, 50, 0.5f, 5.0f, 10, 0.8f, 5.0f, 1, 0, 0, 0, 0},
    {ENT_CULTIST, 40, 0.5f, 4.0f, 8, 1.0f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_GOBLIN, 40, 0.6f, 3.0f, 8, 1.2f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_ORC, 100, 0.5f, 5.0f, 15, 0.8f, 2.0f, 0, 1, 0, 0, 0},
    {ENT_PILLAGER_MARAUDER, 80, 0.8f, 6.0f, 12, 1.0f, 1.5f, 0, 1, 0, 0, 0},
    {ENT_PILLAGER_DESTROYER, 120, 0.5f, 5.0f, 20, 0.8f, 2.0f, 0, 1, 1, 0, 0},
    {ENT_PILLAGER_CORRUPTOR, 60, 0.6f, 5.0f, 10, 1.0f, 1.5f, 0, 1, 0, 0, 0}
};

// ============================================================================
// AI WEIGHT TABLES (THE BRAIN)
// ============================================================================

typedef struct {
    float w_closest_enemy;      // Weight for attacking closest enemy
    float w_lowest_hp;          // Weight for attacking lowest HP
    float w_highest_threat;     // Weight for attacking highest DPS
    float w_structure_priority; // Ravager: prioritize structures
    float w_flee_threshold;     // HP% to start fleeing (0.0-1.0)
    float w_kite_distance;      // Scout: maintain this distance
    float w_pack_bonus;         // Damage bonus when near allies (multiplier)
    float w_support_radius;     // Tidecaller/Hexbound: effect radius
} AIWeights;

static const AIWeights AI_WEIGHT_TABLE[] = {
    // closest, low_hp, high_threat, struct_prio, flee, kite, pack, support_rad
    {1.0f, 0.2f, 0.3f, 0.0f, 0.2f, 0.0f, 1.2f, 0.0f}, // MILITIA: Frontline brawler
    {0.5f, 0.3f, 0.5f, 0.0f, 0.4f, 6.0f, 1.0f, 0.0f}, // SCOUT: Kiting harasser
    {1.0f, 0.5f, 0.0f, 0.0f, 0.0f, 0.0f, 1.5f, 0.0f}, // SWARMLING: Zerg rush
    {0.3f, 0.0f, 0.0f, 1.0f, 0.1f, 0.0f, 1.0f, 0.0f}, // RAVAGER: Structure hunter
    {0.0f, 0.0f, 0.0f, 0.0f, 1.0f, 0.0f, 0.0f, 5.0f}, // HEXBOUND: Pure support
    {0.8f, 0.0f, 0.2f, 0.0f, 0.0f, 0.0f, 1.1f, 0.0f}, // BEHEMOTH: Anchor tank
    {0.2f, 0.0f, 1.0f, 0.0f, 0.3f, 0.0f, 1.0f, 0.0f}, // SHADE: Assassin (high threat)
    {0.5f, 0.0f, 0.3f, 0.3f, 0.5f, 0.0f, 1.0f, 2.0f}, // PYROMANCER: AoE caster
    {0.3f, 0.0f, 0.5f, 0.0f, 0.0f, 0.0f, 1.0f, 5.0f}, // WARDEN: Guard duty
    {0.0f, 1.0f, 0.0f, 0.0f, 1.0f, 0.0f, 0.0f, 4.0f}, // TIDECALLER: Heal bot
    {0.7f, 0.0f, 0.3f, 0.2f, 0.0f, 0.0f, 1.0f, 0.0f}, // GOLEM: Slow bruiser
    {0.9f, 0.3f, 0.0f, 0.0f, 0.0f, 0.0f, 1.3f, 0.0f}, // VOIDREAVER: Berserker
    {0.5f, 0.0f, 0.8f, 0.0f, 0.0f, 0.0f, 1.5f, 0.0f}, // ARCHON: Elite killer
    {1.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 1.0f, 0.0f}, // CHAOS: Random
    {0.6f, 0.2f, 0.5f, 0.0f, 0.0f, 0.0f, 1.2f, 0.0f}, // WRAITH: Revenant
    {0.4f, 0.0f, 0.3f, 0.7f, 0.0f, 0.0f, 1.0f, 0.0f}, // DRAGON: Boss logic
    // NPCs (simpler)
    {1.0f, 0.0f, 0.0f, 0.0f, 0.5f, 0.0f, 1.0f, 0.0f}, // PEASANT
    {0.8f, 0.0f, 0.2f, 0.0f, 0.2f, 0.0f, 1.0f, 0.0f}, // GUARD
    {0.6f, 0.3f, 0.0f, 0.0f, 0.4f, 5.0f, 1.0f, 0.0f}, // HUNTER
    {1.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 1.2f, 0.0f}, // CULTIST
    {1.0f, 0.0f, 0.0f, 0.0f, 0.3f, 0.0f, 1.3f, 0.0f}, // GOBLIN
    {0.9f, 0.0f, 0.1f, 0.0f, 0.1f, 0.0f, 1.1f, 0.0f}, // ORC
    {1.0f, 0.0f, 0.0f, 0.0f, 0.2f, 0.0f, 1.2f, 0.0f}, // MARAUDER
    {0.5f, 0.0f, 0.0f, 1.0f, 0.1f, 0.0f, 1.0f, 0.0f}, // DESTROYER
    {0.8f, 0.2f, 0.0f, 0.0f, 0.0f, 0.0f, 1.1f, 0.0f}  // CORRUPTOR
};

// ============================================================================
// ENTITY STRUCTURE
// ============================================================================

typedef struct Entity {
    int active;
    int id;
    EntityType type;
    Alignment alignment;

    // Transform
    float x, y, z;
    float vx, vz; // Velocity (no Y, ground-based)
    float yaw; // Facing direction

    // Stats (runtime)
    int hp;
    int hp_max;
    float speed;
    float aggro_range;
    int damage;
    float attack_speed;
    float attack_range;
    int is_ranged;
    int is_flying;

    // AI State
    BehaviorMode behavior;
    int target_id; // Entity ID being targeted
    int target_struct_id; // Structure ID being targeted
    unsigned int last_aggro_check; // Frame number
    float attack_cooldown; // Frames until next attack

    // Special States
    int revive_charges; // Wraith King
    int is_invisible; // Shade Stalker
    float drain_timer; // Void Reaver self-damage
    int spawn_tick; // Dragon determinism

    // Grid Position (cached)
    int grid_x, grid_z;

} Entity;

typedef struct Structure {
    int active;
    int id;
    StructureType type;
    Alignment alignment;
    float x, z;
    int hp;
    int hp_max;
    int spawn_cooldown; // Frames until next spawn
    int grid_x, grid_z;
} Structure;

typedef struct Town {
    int active;
    int id;
    TownType type;
    Alignment alignment;
    float x, z;
    int morale; // 0-100, flips at 0
    int population;
    int spawn_cooldown;
    int grid_x, grid_z;
} Town;

// ============================================================================
// GLOBAL STATE
// ============================================================================

typedef struct {
    Entity entities[MAX_ENTITIES];
    Structure structures[MAX_STRUCTURES];
    Town towns[MAX_TOWNS];
    unsigned int frame_count;
} RTSGameState;

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

static inline float dist_sq(float x1, float z1, float x2, float z2) {
    float dx = x2 - x1;
    float dz = z2 - z1;
    return dx * dx + dz * dz;
}

static inline float dist(float x1, float z1, float x2, float z2) {
    return sqrtf(dist_sq(x1, z1, x2, z2));
}

static inline float angle_to(float x1, float z1, float x2, float z2) {
    return atan2f(x2 - x1, z2 - z1) * (180.0f / 3.14159265f);
}

const EntityStats *get_entity_stats(EntityType type) {
    for (int i = 0; i < (int)(sizeof(ENTITY_STAT_TABLE) / sizeof(EntityStats)); i++) {
        if (ENTITY_STAT_TABLE[i].type == type) return &ENTITY_STAT_TABLE[i];
    }
    return NULL;
}

const AIWeights *get_ai_weights(EntityType type) {
    int idx = (int)type;
    if (idx >= 1 && idx <= 16) return &AI_WEIGHT_TABLE[idx - 1];
    if (idx >= 50 && idx <= 58) return &AI_WEIGHT_TABLE[16 + (idx - 50)];
    return &AI_WEIGHT_TABLE[0]; // Default
}

// ============================================================================
// ENTITY INITIALIZATION
// ============================================================================

void init_entity(Entity *e, EntityType type, Alignment align, float x, float z) {
    const EntityStats *stats = get_entity_stats(type);
    if (!stats) return;

    e->active = 1;
    e->type = type;
    e->alignment = align;
    e->x = x;
    e->y = 2.0f; // Ground level
    e->z = z;
    e->vx = 0;
    e->vz = 0;
    e->yaw = 0;

    e->hp = stats->hp_max;
    e->hp_max = stats->hp_max;
    e->speed = stats->speed;
    e->aggro_range = stats->aggro_range;
    e->damage = stats->damage;
    e->attack_speed = stats->attack_speed;
    e->attack_range = stats->attack_range;
    e->is_ranged = stats->is_ranged;
    e->is_flying = stats->is_flying;

    e->behavior = BEHAVIOR_IDLE;
    e->target_id = -1;
    e->target_struct_id = -1;
    e->last_aggro_check = 0;
    e->attack_cooldown = 0;

    e->revive_charges = (type == ENT_WRAITH) ? 1 : 0;
    e->is_invisible = 0;
    e->drain_timer = 0;
    e->spawn_tick = 0;

    e->grid_x = (int)(x / CELL_SIZE);
    e->grid_z = (int)(z / CELL_SIZE);
}

// ============================================================================
// TARGET ACQUISITION (THE REAL AI)
// ============================================================================

int find_best_target(Entity *self, RTSGameState *state) {
    const AIWeights *weights = get_ai_weights(self->type);
    const EntityStats *stats = get_entity_stats(self->type);

    int best_target = -1;
    float best_score = -99999.0f;

    for (int i = 0; i < MAX_ENTITIES; i++) {
        Entity *other = &state->entities[i];
        if (!other->active) continue;
        if (other->id == self->id) continue;
        if (other->alignment == self->alignment) continue; // Don't attack allies
        if (other->alignment == ALIGN_NEUTRAL && self->alignment != ALIGN_CORRUPTED) continue; // Respect neutrals

        float d = dist(self->x, self->z, other->x, other->z);
        if (d > self->aggro_range * CELL_SIZE) continue; // Out of aggro range

        // Calculate weighted score
        float score = 0.0f;

        // Distance score (closer = higher)
        float dist_score = 1.0f - (d / (self->aggro_range * CELL_SIZE));
        score += weights->w_closest_enemy * dist_score;

        // HP score (lower = higher)
        float hp_ratio = (float)other->hp / (float)other->hp_max;
        score += weights->w_lowest_hp * (1.0f - hp_ratio);

        // Threat score (higher DPS = higher)
        float threat = (float)other->damage * other->attack_speed;
        score += weights->w_highest_threat * (threat / 50.0f); // Normalize

        if (score > best_score) {
            best_score = score;
            best_target = i;
        }
    }

    // Check structures if Ravager
    if (stats->prefer_structures && weights->w_structure_priority > 0.5f) {
        for (int i = 0; i < MAX_STRUCTURES; i++) {
            Structure *s = &state->structures[i];
            if (!s->active) continue;
            if (s->alignment == self->alignment) continue;

            float d = dist(self->x, self->z, s->x, s->z);
            if (d > self->aggro_range * CELL_SIZE) continue;

            float struct_score = weights->w_structure_priority * 2.0f;
            if (struct_score > best_score) {
                best_score = struct_score;
                self->target_struct_id = i;
                return -2; // Special flag: targeting structure
            }
        }
    }

    return best_target;
}

// ============================================================================
// MOVEMENT & STEERING
// ============================================================================

void move_towards(Entity *e, float target_x, float target_z, float dt) {
    float dx = target_x - e->x;
    float dz = target_z - e->z;
    float d = sqrtf(dx * dx + dz * dz);

    if (d < 0.1f) {
        e->vx = 0;
        e->vz = 0;
        return;
    }

    // Normalize and scale by speed
    float move_speed = e->speed * CELL_SIZE * dt;
    e->vx = (dx / d) * move_speed;
    e->vz = (dz / d) * move_speed;

    // Update facing
    e->yaw = angle_to(e->x, e->z, target_x, target_z);
}

void apply_velocity(Entity *e, float dt) {
    e->x += e->vx;
    e->z += e->vz;

    // Update grid position
    e->grid_x = (int)(e->x / CELL_SIZE);
    e->grid_z = (int)(e->z / CELL_SIZE);
}

// ============================================================================
// COMBAT
// ============================================================================

void deal_damage(Entity *attacker, Entity *target, RTSGameState *state) {
    target->hp -= attacker->damage;

    if (target->hp <= 0) {
        // Death logic
        if (target->type == ENT_WRAITH && target->revive_charges > 0) {
            target->hp = target->hp_max / 2;
            target->revive_charges--;
            // TODO: Spawn 2 militia
        } else {
            target->active = 0;
            // TODO: Drop loot, trigger quests
        }
    }
}

// ============================================================================
// BEHAVIOR UPDATE (MAIN AI LOOP)
// ============================================================================

void update_entity_behavior(Entity *e, RTSGameState *state, float dt) {
    if (!e->active) return;

    const AIWeights *weights = get_ai_weights(e->type);
    const EntityStats *stats = get_entity_stats(e->type);

    // Special: Void Reaver self-drain
    if (e->type == ENT_VOIDREAVER) {
        e->drain_timer += dt;
        if (e->drain_timer >= 1.0f) {
            e->hp -= 10;
            e->damage += 5; // Ramp up
            e->drain_timer = 0;
            if (e->hp <= 0) {
                // Explode logic
                e->active = 0;
            }
        }
    }

    // Cooldown tick
    if (e->attack_cooldown > 0) {
        e->attack_cooldown -= dt * 60.0f; // Assuming 60 FPS
    }

    // Aggro check (every N frames for performance)
    if (state->frame_count - e->last_aggro_check > AGGRO_CHECK_INTERVAL) {
        e->target_id = find_best_target(e, state);
        e->last_aggro_check = state->frame_count;
    }

    // Support units (Tidecaller, Hexbound) - special behavior
    if (stats->is_support) {
        e->behavior = BEHAVIOR_SUPPORT;
        // TODO: Apply buffs to nearby allies
        return;
    }

    // No target = idle/patrol
    if (e->target_id == -1 && e->target_struct_id == -1) {
        e->behavior = BEHAVIOR_IDLE;
        e->vx = 0;
        e->vz = 0;
        return;
    }

    // Target structure (Ravager logic)
    if (e->target_struct_id >= 0) {
        Structure *s = &state->structures[e->target_struct_id];
        if (!s->active) {
            e->target_struct_id = -1;
            return;
        }

        float d = dist(e->x, e->z, s->x, s->z);
        if (d > e->attack_range * CELL_SIZE) {
            move_towards(e, s->x, s->z, dt);
            e->behavior = BEHAVIOR_MOVE_TO_TARGET;
        } else {
            e->behavior = BEHAVIOR_ATTACK_TARGET;
            e->vx = 0;
            e->vz = 0;
            if (e->attack_cooldown <= 0) {
                s->hp -= e->damage;
                if (s->hp <= 0) s->active = 0;
                e->attack_cooldown = 60.0f / e->attack_speed;
            }
        }
        return;
    }

    // Target entity
    Entity *target = &state->entities[e->target_id];
    if (!target->active) {
        e->target_id = -1;
        return;
    }

    float d = dist(e->x, e->z, target->x, target->z);

    // Kiting behavior (Scout)
    if (weights->w_kite_distance > 0 && e->is_ranged) {
        float kite_dist = weights->w_kite_distance * CELL_SIZE;
        if (d < kite_dist) {
            // Move away
            float away_x = e->x - target->x;
            float away_z = e->z - target->z;
            float away_len = sqrtf(away_x * away_x + away_z * away_z);
            if (away_len > 0.1f) {
                move_towards(e,
                             e->x + (away_x / away_len) * kite_dist,
                             e->z + (away_z / away_len) * kite_dist,
                             dt);
            }
            e->behavior = BEHAVIOR_FLEE;
        } else if (d > e->attack_range * CELL_SIZE) {
            move_towards(e, target->x, target->z, dt);
            e->behavior = BEHAVIOR_MOVE_TO_TARGET;
        } else {
            e->behavior = BEHAVIOR_ATTACK_TARGET;
            e->vx = 0;
            e->vz = 0;
        }
    } else {
        // Standard behavior
        if (d > e->attack_range * CELL_SIZE) {
            move_towards(e, target->x, target->z, dt);
            e->behavior = BEHAVIOR_MOVE_TO_TARGET;
        } else {
            e->behavior = BEHAVIOR_ATTACK_TARGET;
            e->vx = 0;
            e->vz = 0;
        }
    }

    // Attack if in range
    if (e->behavior == BEHAVIOR_ATTACK_TARGET && e->attack_cooldown <= 0) {
        deal_damage(e, target, state);
        e->attack_cooldown = 60.0f / e->attack_speed;
    }

    // Apply movement
    apply_velocity(e, dt);
}

// ============================================================================
// DRAGON DETERMINISM SYSTEM
// ============================================================================

// Dragon uses PRNG seeded by spawn tick for all decisions
typedef struct {
    unsigned int seed;
} DragonRNG;

unsigned int dragon_rand(DragonRNG *rng) {
    // LCG: xorshift32
    rng->seed ^= rng->seed << 13;
    rng->seed ^= rng->seed >> 17;
    rng->seed ^= rng->seed << 5;
    return rng->seed;
}

float dragon_randf(DragonRNG *rng) {
    return (float)(dragon_rand(rng) % 10000) / 10000.0f;
}

void update_dragon(Entity *dragon, RTSGameState *state, float dt) {
    if (dragon->type != ENT_DRAGON) return;

    // Initialize RNG if first update
    if (dragon->spawn_tick == 0) {
        dragon->spawn_tick = state->frame_count;
    }

    DragonRNG rng;
    rng.seed = dragon->spawn_tick + state->frame_count;

    // Dragon AI: Patrol between structures, breathe fire occasionally
    if (state->frame_count % 120 == 0) { // Every 2 seconds
        // Pick random enemy structure
        int struct_count = 0;
        int struct_indices[MAX_STRUCTURES];
        for (int i = 0; i < MAX_STRUCTURES; i++) {
            if (state->structures[i].active && state->structures[i].alignment != dragon->alignment) {
                struct_indices[struct_count++] = i;
            }
        }

        if (struct_count > 0) {
            int idx = dragon_rand(&rng) % struct_count;
            dragon->target_struct_id = struct_indices[idx];
        }
    }

    // Move towards target
    if (dragon->target_struct_id >= 0) {
        Structure *s = &state->structures[dragon->target_struct_id];
        if (s->active) {
            float d = dist(dragon->x, dragon->z, s->x, s->z);
            if (d > dragon->attack_range * CELL_SIZE) {
                move_towards(dragon, s->x, s->z, dt);
            } else {
                // Breathe fire (line AoE)
                if (dragon->attack_cooldown <= 0) {
                    // TODO: Implement line AoE damage
                    s->hp -= dragon->damage;
                    if (s->hp <= 0) s->active = 0;
                    dragon->attack_cooldown = 60.0f / dragon->attack_speed;
                }
            }
        }
    }

    apply_velocity(dragon, dt);
}

#endif // ENTITY_BEHAVIORS_H
