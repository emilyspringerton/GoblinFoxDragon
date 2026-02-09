#ifndef CARD_SYSTEM_H
#define CARD_SYSTEM_H

#include "entity_behaviors.h"
#include "grid_tick.h"

// ============================================================================
// CARD CONSTANTS
// ============================================================================

#define MAX_HAND_SIZE 5
#define MAX_DECK_SIZE 30
#define CARD_COOLDOWN_FRAMES 180 // 3 seconds at 60fps

// ============================================================================
// CARD STRUCTURE
// ============================================================================

typedef enum {
    CARD_MILITIA = 1,
    CARD_SCOUT = 2,
    CARD_SWARMLING = 3,
    CARD_RAVAGER = 4,
    CARD_HEXBOUND = 5,
    CARD_BEHEMOTH = 6,
    CARD_SHADE = 7,
    CARD_PYROMANCER = 8,
    CARD_WARDEN = 9,
    CARD_TIDECALLER = 10,
    CARD_GOLEM = 11,
    CARD_VOIDREAVER = 12,
    CARD_ARCHON = 13,
    CARD_CHAOS = 14,
    CARD_WRAITH = 15,
    CARD_DRAGON = 16,
    // Structures
    CARD_OUTPOST = 100,
    CARD_MANAWELL = 101,
    CARD_CORRUPTION_SPIRE = 102,
    CARD_GROVE_HEART = 103,
    CARD_SIEGE_WORKSHOP = 104,
    CARD_SHADOW_SANCTUM = 105,
    CARD_INFERNO_TOWER = 106,
    CARD_NEXUS_CORE = 107
} CardType;

typedef struct {
    CardType type;
    int cost; // Influence points
    int cooldown; // Frames remaining
    int tech_level; // 0-3
    int is_structure; // 0 = unit, 1 = structure
    EntityType spawns_entity; // For unit cards
    StructureType spawns_structure; // For structure cards
    char name[32];
    float color_r, color_g, color_b; // Neon accent color
} Card;

typedef struct {
    Card hand[MAX_HAND_SIZE];
    Card deck[MAX_DECK_SIZE];
    int deck_count;
    int deck_draw_index; // Next card to draw
} CardDeck;

// ============================================================================
// CARD DEFINITIONS
// ============================================================================

static const Card CARD_TEMPLATES[] = {
    // type, cost, cooldown, tech_level, is_struct, ent, struct, name, r, g, b
    {CARD_MILITIA, 2, 0, 0, 0, ENT_MILITIA, STRUCT_NONE, "Militia", 1.0f, 0.0f, 0.8f},
    {CARD_SCOUT, 3, 0, 0, 0, ENT_SCOUT, STRUCT_NONE, "Scout", 0.0f, 1.0f, 1.0f},
    {CARD_SWARMLING, 1, 0, 0, 0, ENT_SWARMLING, STRUCT_NONE, "Swarmling", 1.0f, 1.0f, 0.0f},
    {CARD_RAVAGER, 4, 0, 0, 0, ENT_RAVAGER, STRUCT_NONE, "Ravager", 1.0f, 0.5f, 0.0f},
    {CARD_HEXBOUND, 5, 0, 0, 0, ENT_HEXBOUND, STRUCT_NONE, "Hexbound", 0.5f, 0.0f, 1.0f},
    {CARD_BEHEMOTH, 6, 0, 0, 0, ENT_BEHEMOTH, STRUCT_NONE, "Behemoth", 0.0f, 1.0f, 0.0f},
    {CARD_SHADE, 4, 0, 0, 0, ENT_SHADE, STRUCT_NONE, "Shade Stalker", 0.3f, 0.3f, 0.3f},
    {CARD_PYROMANCER, 5, 0, 0, 0, ENT_PYROMANCER, STRUCT_NONE, "Pyromancer", 1.0f, 0.2f, 0.0f},
    {CARD_WARDEN, 7, 0, 0, 0, ENT_WARDEN, STRUCT_NONE, "Warden", 0.0f, 0.5f, 1.0f},
    {CARD_TIDECALLER, 6, 0, 0, 0, ENT_TIDECALLER, STRUCT_NONE, "Tide Caller", 0.0f, 0.8f, 1.0f},
    {CARD_GOLEM, 8, 0, 0, 0, ENT_GOLEM, STRUCT_NONE, "Siege Golem", 0.5f, 0.5f, 0.5f},
    {CARD_VOIDREAVER, 7, 0, 0, 0, ENT_VOIDREAVER, STRUCT_NONE, "Void Reaver", 0.3f, 0.0f, 0.3f},
    {CARD_ARCHON, 10, 0, 0, 0, ENT_ARCHON, STRUCT_NONE, "Archon", 1.0f, 1.0f, 1.0f},
    {CARD_CHAOS, 6, 0, 0, 0, ENT_CHAOS, STRUCT_NONE, "Chaos Spawn", 1.0f, 0.0f, 1.0f},
    {CARD_WRAITH, 9, 0, 0, 0, ENT_WRAITH, STRUCT_NONE, "Wraith King", 0.2f, 1.0f, 0.2f},
    {CARD_DRAGON, 12, 0, 0, 0, ENT_DRAGON, STRUCT_NONE, "Dragon", 1.0f, 0.5f, 0.0f},
    // Structures
    {CARD_OUTPOST, 5, 0, 0, 1, ENT_NONE, STRUCT_OUTPOST, "Outpost", 0.0f, 1.0f, 0.0f},
    {CARD_MANAWELL, 6, 0, 0, 1, ENT_NONE, STRUCT_MANAWELL, "Mana Well", 0.0f, 0.5f, 1.0f},
    {CARD_CORRUPTION_SPIRE, 7, 0, 0, 1, ENT_NONE, STRUCT_CORRUPTION_SPIRE, "Corruption Spire", 0.5f, 0.0f, 0.5f},
    {CARD_GROVE_HEART, 8, 0, 0, 1, ENT_NONE, STRUCT_GROVE_HEART, "Grove Heart", 0.0f, 1.0f, 0.2f},
    {CARD_SIEGE_WORKSHOP, 9, 0, 0, 1, ENT_NONE, STRUCT_SIEGE_WORKSHOP, "Siege Workshop", 0.5f, 0.5f, 0.5f},
    {CARD_SHADOW_SANCTUM, 7, 0, 0, 1, ENT_NONE, STRUCT_SHADOW_SANCTUM, "Shadow Sanctum", 0.2f, 0.2f, 0.2f},
    {CARD_INFERNO_TOWER, 6, 0, 0, 1, ENT_NONE, STRUCT_INFERNO_TOWER, "Inferno Tower", 1.0f, 0.3f, 0.0f},
    {CARD_NEXUS_CORE, 15, 0, 0, 1, ENT_NONE, STRUCT_NEXUS_CORE, "Nexus Core", 1.0f, 1.0f, 0.0f}
};

// ============================================================================
// CARD SYSTEM STATE
// ============================================================================

typedef struct {
    int player_influence;
    int max_influence;
    CardDeck player_deck;
    CardDeck enemy_deck; // For multiplayer
} CardGameState;

// ============================================================================
// DECK MANAGEMENT
// ============================================================================

void init_starter_deck(CardDeck *deck) {
    deck->deck_count = 0;
    deck->deck_draw_index = 0;

    // Starter deck: 8 Militia, 6 Scout, 4 Swarmling, 3 Ravager, 3 Outpost
    for (int i = 0; i < 8; i++) {
        deck->deck[deck->deck_count++] = CARD_TEMPLATES[0]; // Militia
    }
    for (int i = 0; i < 6; i++) {
        deck->deck[deck->deck_count++] = CARD_TEMPLATES[1]; // Scout
    }
    for (int i = 0; i < 4; i++) {
        deck->deck[deck->deck_count++] = CARD_TEMPLATES[2]; // Swarmling
    }
    for (int i = 0; i < 3; i++) {
        deck->deck[deck->deck_count++] = CARD_TEMPLATES[3]; // Ravager
    }
    for (int i = 0; i < 3; i++) {
        deck->deck[deck->deck_count++] = CARD_TEMPLATES[16]; // Outpost
    }

    // Shuffle (Fisher-Yates)
    for (int i = deck->deck_count - 1; i > 0; i--) {
        int j = rand() % (i + 1);
        Card temp = deck->deck[i];
        deck->deck[i] = deck->deck[j];
        deck->deck[j] = temp;
    }

    // Draw initial hand
    for (int i = 0; i < MAX_HAND_SIZE; i++) {
        deck->hand[i] = deck->deck[deck->deck_draw_index++];
        if (deck->deck_draw_index >= deck->deck_count) {
            deck->deck_draw_index = 0; // Reshuffle
        }
    }
}

void draw_card(CardDeck *deck, int hand_slot) {
    if (hand_slot < 0 || hand_slot >= MAX_HAND_SIZE) return;

    deck->hand[hand_slot] = deck->deck[deck->deck_draw_index++];
    if (deck->deck_draw_index >= deck->deck_count) {
        deck->deck_draw_index = 0; // Loop deck
    }
}

// ============================================================================
// CARD PLAYING (SPAWN LOGIC)
// ============================================================================

int can_play_card(Card *card, CardGameState *card_state, GridState *grid, int grid_x, int grid_z) {
    // Check influence cost
    if (card->cost > card_state->player_influence) return 0;

    // Check cooldown
    if (card->cooldown > 0) return 0;

    // Check grid validity
    int idx = grid_idx(grid_x, grid_z);
    if (idx == -1) return 0;

    GridCell *cell = &grid->cells[idx];

    // Can't spawn in corrupted cells
    if (cell->state == CELL_CORRUPTED) return 0;

    // Can't spawn if cell is overcrowded
    if (cell->population > 200) return 0;

    // Structures can't overlap
    if (card->is_structure && cell->has_structure >= 0) return 0;

    return 1;
}

int play_card(Card *card,
              CardGameState *card_state,
              RTSGameState *rts_state,
              GridState *grid,
              int grid_x,
              int grid_z,
              int hand_slot) {
    if (!can_play_card(card, card_state, grid, grid_x, grid_z)) return 0;

    // Deduct cost
    card_state->player_influence -= card->cost;

    // Spawn entity or structure
    float world_x = grid_x * CELL_SIZE + CELL_SIZE / 2.0f;
    float world_z = grid_z * CELL_SIZE + CELL_SIZE / 2.0f;

    if (card->is_structure) {
        // Find empty structure slot
        for (int i = 0; i < MAX_STRUCTURES; i++) {
            if (!rts_state->structures[i].active) {
                Structure *s = &rts_state->structures[i];
                s->active = 1;
                s->id = i;
                s->type = card->spawns_structure;
                s->alignment = ALIGN_PLAYER;
                s->x = world_x;
                s->z = world_z;
                s->hp = 200; // Default, should be from table
                s->hp_max = 200;
                s->spawn_cooldown = 0;
                s->grid_x = grid_x;
                s->grid_z = grid_z;

                // Mark grid
                int idx = grid_idx(grid_x, grid_z);
                if (idx >= 0) grid->cells[idx].has_structure = i;

                break;
            }
        }
    } else {
        // Spawn entity (or multiple for Swarmling)
        int spawn_count = (card->type == CARD_SWARMLING) ? 4 : 1;

        for (int n = 0; n < spawn_count; n++) {
            for (int i = 0; i < MAX_ENTITIES; i++) {
                if (!rts_state->entities[i].active) {
                    Entity *e = &rts_state->entities[i];

                    // Slight offset for multiple spawns
                    float offset_x = (n % 2 == 0) ? -5.0f : 5.0f;
                    float offset_z = (n / 2 == 0) ? -5.0f : 5.0f;

                    init_entity(e,
                                card->spawns_entity,
                                ALIGN_PLAYER,
                                world_x + offset_x,
                                world_z + offset_z);
                    e->id = i;
                    break;
                }
            }
        }
    }

    // Apply cooldown
    card->cooldown = CARD_COOLDOWN_FRAMES;

    // Draw new card
    draw_card(&card_state->player_deck, hand_slot);

    return 1;
}

// ============================================================================
// CARD UPDATE (COOLDOWNS)
// ============================================================================

void update_cards(CardGameState *card_state) {
    for (int i = 0; i < MAX_HAND_SIZE; i++) {
        if (card_state->player_deck.hand[i].cooldown > 0) {
            card_state->player_deck.hand[i].cooldown--;
        }
    }
}

// ============================================================================
// INFLUENCE GENERATION
// ============================================================================

void update_influence(CardGameState *card_state, RTSGameState *rts_state) {
    // Base income: +1 per second (60 frames)
    static int income_timer = 0;
    income_timer++;
    if (income_timer >= 60) {
        card_state->player_influence += 1;
        income_timer = 0;
    }

    // Mana Wells give bonus
    for (int i = 0; i < MAX_STRUCTURES; i++) {
        Structure *s = &rts_state->structures[i];
        if (!s->active) continue;
        if (s->type == STRUCT_MANAWELL && s->alignment == ALIGN_PLAYER) {
            static int well_timer = 0;
            well_timer++;
            if (well_timer >= 180) { // Every 3 seconds
                card_state->player_influence += 1;
                well_timer = 0;
            }
        }
    }

    // Clamp
    if (card_state->player_influence > card_state->max_influence) {
        card_state->player_influence = card_state->max_influence;
    }
}

// ============================================================================
// TECH TREE INTEGRATION (UPGRADES CARDS)
// ============================================================================

void upgrade_card_in_deck(CardDeck *deck, CardType card_type) {
    for (int i = 0; i < deck->deck_count; i++) {
        if (deck->deck[i].type == card_type) {
            deck->deck[i].tech_level++;
            if (deck->deck[i].tech_level > 3) deck->deck[i].tech_level = 3;

            // Apply stat upgrades (example: Militia)
            if (card_type == CARD_MILITIA) {
                // Tech 1: +20 HP (handled in entity_behaviors via tech_level check)
                // Tech 2: Shield Bash ability
                // Tech 3: Captain upgrade
            }
        }
    }
}

// ============================================================================
// QUEST UNLOCKS (ADD CARDS TO DECK)
// ============================================================================

void unlock_card(CardDeck *deck, CardType card_type) {
    if (deck->deck_count >= MAX_DECK_SIZE) return;

    // Find template
    for (int i = 0; i < (int)(sizeof(CARD_TEMPLATES) / sizeof(Card)); i++) {
        if (CARD_TEMPLATES[i].type == card_type) {
            deck->deck[deck->deck_count++] = CARD_TEMPLATES[i];
            break;
        }
    }
}

// ============================================================================
// INITIALIZATION
// ============================================================================

void init_card_game_state(CardGameState *state) {
    state->player_influence = 10; // Starting influence
    state->max_influence = 30;
    init_starter_deck(&state->player_deck);
}

#endif // CARD_SYSTEM_H
