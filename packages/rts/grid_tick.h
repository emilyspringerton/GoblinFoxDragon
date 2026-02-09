#ifndef GRID_TICK_H
#define GRID_TICK_H

#include <string.h>
#include "entity_behaviors.h"

// ============================================================================
// GRID CONSTANTS
// ============================================================================

#define GRID_WIDTH 40  // 40 cells wide
#define GRID_HEIGHT 40 // 40 cells tall
#define GRID_TOTAL (GRID_WIDTH * GRID_HEIGHT)

#define TICK_INTERVAL 120 // Frames between automata updates (2 sec at 60fps)

// ============================================================================
// CELL STRUCTURE
// ============================================================================

typedef enum {
    CELL_NEUTRAL = 0,
    CELL_PLAYER = 1,
    CELL_ENEMY = 2,
    CELL_CORRUPTED = 3
} CellState;

typedef struct {
    CellState state;
    int population; // 0-255
    int alignment_pressure; // -127 to +127 (negative = enemy, positive = player)
    int growth_rate; // -10 to +10 per tick
    float stability; // 0.0 to 1.0
    int conversion_progress; // 0-100, becomes converted at 100
    int corruption_level; // 0-100, becomes corrupted at 100
    int has_structure; // Structure ID or -1
    int has_town; // Town ID or -1
} GridCell;

typedef struct {
    GridCell cells[GRID_TOTAL];
    unsigned int last_tick_frame;
} GridState;

// ============================================================================
// GRID UTILITIES
// ============================================================================

static inline int grid_idx(int x, int z) {
    if (x < 0 || x >= GRID_WIDTH || z < 0 || z >= GRID_HEIGHT) return -1;
    return z * GRID_WIDTH + x;
}

static inline void grid_coords(int idx, int *x, int *z) {
    *z = idx / GRID_WIDTH;
    *x = idx % GRID_WIDTH;
}

// ============================================================================
// NEIGHBOR COUNTING (CONWAY LOGIC)
// ============================================================================

typedef struct {
    int player_count;
    int enemy_count;
    int neutral_count;
    int corrupted_count;
    int total_population;
} NeighborInfo;

NeighborInfo count_neighbors(GridState *grid, int x, int z) {
    NeighborInfo info = {0, 0, 0, 0, 0};

    // 8-directional Moore neighborhood
    int offsets[8][2] = {
        {-1, -1}, {0, -1}, {1, -1},
        {-1,  0},          {1,  0},
        {-1,  1}, {0,  1}, {1,  1}
    };

    for (int i = 0; i < 8; i++) {
        int nx = x + offsets[i][0];
        int nz = z + offsets[i][1];
        int idx = grid_idx(nx, nz);
        if (idx == -1) continue;

        GridCell *cell = &grid->cells[idx];
        info.total_population += cell->population;

        switch (cell->state) {
            case CELL_PLAYER: info.player_count++; break;
            case CELL_ENEMY: info.enemy_count++; break;
            case CELL_NEUTRAL: info.neutral_count++; break;
            case CELL_CORRUPTED: info.corrupted_count++; break;
        }
    }

    return info;
}

// ============================================================================
// CONWAY RULES (THE LIVING MAP)
// ============================================================================

void apply_conway_rules(GridState *grid, int x, int z) {
    int idx = grid_idx(x, z);
    if (idx == -1) return;

    GridCell *cell = &grid->cells[idx];
    NeighborInfo neighbors = count_neighbors(grid, x, z);

    // RULE 1: Conversion via neighbor pressure
    if (cell->state == CELL_NEUTRAL) {
        if (neighbors.player_count >= 3 && cell->alignment_pressure > 50) {
            cell->conversion_progress += 20;
            if (cell->conversion_progress >= 100) {
                cell->state = CELL_PLAYER;
                cell->conversion_progress = 0;
            }
        } else if (neighbors.enemy_count >= 3 && cell->alignment_pressure < -50) {
            cell->conversion_progress += 20;
            if (cell->conversion_progress >= 100) {
                cell->state = CELL_ENEMY;
                cell->conversion_progress = 0;
            }
        } else {
            cell->conversion_progress -= 5;
            if (cell->conversion_progress < 0) cell->conversion_progress = 0;
        }
    }

    // RULE 2: Population growth/decay
    cell->population += cell->growth_rate;

    // Natural growth based on neighbors
    if (cell->state != CELL_CORRUPTED) {
        int growth_factor = (neighbors.total_population / 8) / 10;
        cell->population += growth_factor;
    }

    // Clamp
    if (cell->population > 255) cell->population = 255;
    if (cell->population < 0) cell->population = 0;

    // RULE 3: Overpopulation collapse
    if (cell->population > 200) {
        cell->population = 150; // Collapse
        cell->stability -= 0.2f;
        // TODO: Spread to adjacent cells
    }

    // RULE 4: Underpopulation death
    if (cell->population < 20 && cell->state != CELL_NEUTRAL) {
        cell->stability -= 0.1f;
        if (cell->stability <= 0) {
            cell->state = CELL_NEUTRAL;
            cell->stability = 1.0f;
        }
    }

    // RULE 5: Corruption spread
    if (neighbors.corrupted_count >= 4) {
        cell->corruption_level += 15;
        if (cell->corruption_level >= 100) {
            cell->state = CELL_CORRUPTED;
            cell->population /= 2; // Corruption kills
        }
    } else if (cell->corruption_level > 0) {
        cell->corruption_level -= 2; // Decay slowly
    }

    // Stability recovery
    if (cell->stability < 1.0f) cell->stability += 0.05f;
    if (cell->stability > 1.0f) cell->stability = 1.0f;
}

// ============================================================================
// PRESSURE CALCULATION (FROM ENTITIES)
// ============================================================================

void calculate_alignment_pressure(GridState *grid, RTSGameState *rts_state) {
    // Reset pressure
    for (int i = 0; i < GRID_TOTAL; i++) {
        grid->cells[i].alignment_pressure = 0;
    }

    // Entities exert pressure on their cell
    for (int i = 0; i < MAX_ENTITIES; i++) {
        Entity *e = &rts_state->entities[i];
        if (!e->active) continue;

        int idx = grid_idx(e->grid_x, e->grid_z);
        if (idx == -1) continue;

        int pressure = 10; // Base pressure per unit
        if (e->alignment == ALIGN_PLAYER) {
            grid->cells[idx].alignment_pressure += pressure;
        } else if (e->alignment == ALIGN_ENEMY) {
            grid->cells[idx].alignment_pressure -= pressure;
        }
    }

    // Structures exert stronger pressure
    for (int i = 0; i < MAX_STRUCTURES; i++) {
        Structure *s = &rts_state->structures[i];
        if (!s->active) continue;

        int idx = grid_idx(s->grid_x, s->grid_z);
        if (idx == -1) continue;

        int pressure = 30; // Structures have 3x pressure
        if (s->alignment == ALIGN_PLAYER) {
            grid->cells[idx].alignment_pressure += pressure;
        } else if (s->alignment == ALIGN_ENEMY) {
            grid->cells[idx].alignment_pressure -= pressure;
        }

        // Also affect neighbors
        for (int dx = -1; dx <= 1; dx++) {
            for (int dz = -1; dz <= 1; dz++) {
                if (dx == 0 && dz == 0) continue;
                int nidx = grid_idx(s->grid_x + dx, s->grid_z + dz);
                if (nidx == -1) continue;
                if (s->alignment == ALIGN_PLAYER) {
                    grid->cells[nidx].alignment_pressure += 10;
                } else if (s->alignment == ALIGN_ENEMY) {
                    grid->cells[nidx].alignment_pressure -= 10;
                }
            }
        }
    }

    // Clamp pressure
    for (int i = 0; i < GRID_TOTAL; i++) {
        if (grid->cells[i].alignment_pressure > 127) grid->cells[i].alignment_pressure = 127;
        if (grid->cells[i].alignment_pressure < -127) grid->cells[i].alignment_pressure = -127;
    }
}

// ============================================================================
// TOWN ECOLOGY (CONWAY FOR TOWNS)
// ============================================================================

void update_town_ecology(RTSGameState *rts_state, GridState *grid) {
    for (int i = 0; i < MAX_TOWNS; i++) {
        Town *t = &rts_state->towns[i];
        if (!t->active) continue;

        // Count neighboring towns
        int friend_towns = 0;
        int enemy_towns = 0;

        for (int j = 0; j < MAX_TOWNS; j++) {
            if (i == j || !rts_state->towns[j].active) continue;
            float d = dist(t->x, t->z, rts_state->towns[j].x, rts_state->towns[j].z);
            if (d < 3 * CELL_SIZE) {
                if (rts_state->towns[j].alignment == t->alignment) {
                    friend_towns++;
                } else if (rts_state->towns[j].alignment != ALIGN_NEUTRAL) {
                    enemy_towns++;
                }
            }
        }

        // Conway-inspired rules for towns
        if (friend_towns == 0 && enemy_towns >= 2) {
            // Isolation death
            t->morale -= 20;
        }

        if (friend_towns >= 3) {
            // Overpopulation (one town might die)
            // For simplicity, lowest morale dies
            // TODO: Implement properly
        }

        // Morale → alignment flip
        if (t->morale <= 0 && t->alignment != ALIGN_NEUTRAL) {
            t->alignment = ALIGN_NEUTRAL;
            t->morale = 50;
        }

        // Morale recovery
        if (t->morale < 100 && friend_towns > enemy_towns) {
            t->morale += 5;
        }

        // Update grid cell
        int idx = grid_idx(t->grid_x, t->grid_z);
        if (idx != -1) {
            grid->cells[idx].has_town = i;
            grid->cells[idx].population = t->population;
        }
    }
}

// ============================================================================
// MAIN GRID TICK
// ============================================================================

void tick_grid(GridState *grid, RTSGameState *rts_state, unsigned int current_frame) {
    if (current_frame - grid->last_tick_frame < TICK_INTERVAL) return;
    grid->last_tick_frame = current_frame;

    // Step 1: Calculate pressure from entities/structures
    calculate_alignment_pressure(grid, rts_state);

    // Step 2: Apply Conway rules to all cells
    for (int z = 0; z < GRID_HEIGHT; z++) {
        for (int x = 0; x < GRID_WIDTH; x++) {
            apply_conway_rules(grid, x, z);
        }
    }

    // Step 3: Update town ecology
    update_town_ecology(rts_state, grid);
}

// ============================================================================
// INITIALIZATION
// ============================================================================

void init_grid(GridState *grid) {
    memset(grid, 0, sizeof(GridState));

    // All cells start neutral with some population
    for (int i = 0; i < GRID_TOTAL; i++) {
        grid->cells[i].state = CELL_NEUTRAL;
        grid->cells[i].population = 50;
        grid->cells[i].stability = 1.0f;
        grid->cells[i].growth_rate = 1;
        grid->cells[i].has_structure = -1;
        grid->cells[i].has_town = -1;
    }
}

#endif // GRID_TICK_H
