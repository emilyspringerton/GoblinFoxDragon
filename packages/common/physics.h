#ifndef PHYSICS_H
#define PHYSICS_H
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include "protocol.h"

// --- TUNING ---
#define GRAVITY_FLOAT 0.025f 
#define GRAVITY_DROP 0.075f  
#define JUMP_FORCE 0.95f     
#define MAX_SPEED 0.95f      
#define FRICTION 0.15f      
#define ACCEL 0.6f          
#define STOP_SPEED 0.1f     
#define SLIDE_FRICTION 0.01f 
#define CROUCH_SPEED 0.35f  

// --- BUGGY TUNING ---
#define BUGGY_MAX_SPEED 2.5f    
#define BUGGY_ACCEL 0.08f       
#define BUGGY_FRICTION 0.03f    
#define BUGGY_GRAVITY 0.15f     

#define BIKE_MAX_SPEED 3.8f
#define BIKE_FRICTION 0.02f
#define BIKE_GRAVITY 0.12f

// --- VEHICLE HANDLING ---
#define BUGGY_LATERAL_GRIP 10.0f
#define BUGGY_DRIFT_LATERAL_GRIP 2.0f
#define BUGGY_ALIGN_RATE 6.0f

#define BIKE_LATERAL_GRIP 14.0f
#define BIKE_DRIFT_LATERAL_GRIP 4.0f
#define BIKE_ALIGN_RATE 8.0f

#define VEH_MIN_GRIP_SPEED 2.5f

#define BIKE_GEARS 5
static const float BIKE_GEAR_MAX[BIKE_GEARS + 1] = {0.0f, 1.2f, 2.0f, 2.7f, 3.3f, 3.8f};
static const float BIKE_GEAR_ACCEL[BIKE_GEARS + 1] = {0.0f, 0.14f, 0.11f, 0.095f, 0.08f, 0.07f};

#define EYE_HEIGHT 2.59f    
#define PLAYER_WIDTH 0.97f  
#define PLAYER_HEIGHT 6.47f 
#define HEAD_SIZE 1.94f     
#define HEAD_OFFSET 2.42f   
#define MELEE_RANGE_SQ 250.0f 

void evolve_bot(PlayerState *loser, PlayerState *winner);
PlayerState* get_best_bot();

typedef struct { float x, y, z, w, h, d; } Box;
typedef struct { float x, y; } Vec2;

// --- SCENES ---
static const Box map_geo_stadium[] = {
    {0.00, -2.00, 0.00, 800.00, 4.00, 800.00},
    {400.00, 100.00, 0.00, 10.00, 200.00, 800.00},
    {-400.00, 100.00, 0.00, 10.00, 200.00, 800.00},
    {0.00, 100.00, 400.00, 800.00, 200.00, 10.00},
    {0.00, 100.00, -400.00, 800.00, 200.00, 10.00},
    {213.87, 8.96, 200.27, 15.00, 10.00, 15.00},
    {308.74, 7.12, -328.21, 15.00, 10.00, 15.00},
    {-238.16, 2.91, -230.20, 15.00, 10.00, 15.00},
    {92.42, 4.21, 263.73, 15.00, 10.00, 15.00},
    {-4.91, 9.89, 348.78, 15.00, 10.00, 15.00},
    {-150.07, 8.09, -176.99, 15.00, 10.00, 15.00},
    {-306.92, 3.73, 222.71, 15.00, 10.00, 15.00},
    {120.95, 4.60, 18.02, 15.00, 10.00, 15.00},
    {285.46, 9.59, -176.44, 15.00, 10.00, 15.00},
    {52.00, 9.55, 266.13, 15.00, 10.00, 15.00},
    {288.99, 9.24, -144.13, 15.00, 10.00, 15.00},
    {339.68, 3.44, -104.49, 15.00, 10.00, 15.00},
    {-129.92, 2.70, 329.81, 15.00, 10.00, 15.00},
    {-166.15, 6.06, 136.80, 15.00, 10.00, 15.00},
    {-105.63, 3.99, 203.16, 15.00, 10.00, 15.00},
    {-307.50, 2.08, 73.06, 15.00, 10.00, 15.00},
    {147.40, 5.61, -135.57, 15.00, 10.00, 15.00},
    {162.00, 9.95, 344.85, 15.00, 10.00, 15.00},
    {-114.92, 3.17, 146.63, 15.00, 10.00, 15.00},
    {336.06, 5.49, 59.15, 15.00, 10.00, 15.00},
    {-168.92, 3.70, 33.12, 15.00, 10.00, 15.00},
    {-338.01, 8.58, -117.71, 15.00, 10.00, 15.00},
    {-127.03, 5.75, 29.02, 15.00, 10.00, 15.00},
    {128.29, 7.18, -329.66, 15.00, 10.00, 15.00},
    {-18.08, 8.97, 172.50, 15.00, 10.00, 15.00},
    {225.41, 5.36, 170.34, 15.00, 10.00, 15.00},
    {337.22, 4.43, -14.90, 15.00, 10.00, 15.00},
    {-321.12, 8.33, 22.02, 15.00, 10.00, 15.00}
};

static const Box map_geo_garage[] = {
    {0.00, -2.00, 0.00, 160.00, 4.00, 160.00},
    {0.00, -8.00, 0.00, 170.00, 2.00, 170.00},
    {0.00, 18.00, 0.00, 160.00, 4.00, 160.00},
    {60.00, 9.00, 0.00, 4.00, 18.00, 160.00},
    {-60.00, 9.00, 0.00, 4.00, 18.00, 160.00},
    {0.00, 9.00, 60.00, 160.00, 18.00, 4.00},
    {0.00, 9.00, -60.00, 160.00, 18.00, 4.00},
    {64.00, 9.00, 0.00, 4.00, 22.00, 170.00},
    {-64.00, 9.00, 0.00, 4.00, 22.00, 170.00},
    {0.00, 9.00, 64.00, 170.00, 22.00, 4.00},
    {0.00, 9.00, -64.00, 170.00, 22.00, 4.00},
    {0.00, 21.00, 64.50, 174.00, 2.00, 6.00},
    {0.00, 21.00, -64.50, 174.00, 2.00, 6.00},
    {64.50, 21.00, 0.00, 6.00, 2.00, 174.00},
    {-64.50, 21.00, 0.00, 6.00, 2.00, 174.00},
    {0.00, 9.00, 52.00, 14.00, 12.00, 2.00},
    {-10.00, 5.00, -20.00, 12.00, 4.00, 12.00},
    {10.00, 5.00, -20.00, 12.00, 4.00, 12.00}
};


#define CITY_MAX_BOXES 8192
static Box map_geo_voxworld[CITY_MAX_BOXES];
static int map_geo_voxworld_count = 0;
static int map_geo_voxworld_init = 0;

static const Box *map_geo = map_geo_stadium;
static int map_count = 0;

#define GARAGE_KILL_Y -30.0f
#define GARAGE_BOUNDS_X 70.0f
#define GARAGE_BOUNDS_Z 70.0f

#define STADIUM_KILL_Y -80.0f
#define STADIUM_BOUNDS_X 420.0f
#define STADIUM_BOUNDS_Z 420.0f

#define VOXWORLD_KILL_Y -120.0f
#define VOXWORLD_BOUNDS_X 1180.0f
#define VOXWORLD_BOUNDS_Z 1180.0f
#define VOXWORLD_SEED 1337
#define VOXWORLD_TRACE_STEP 4.0f

#define CITY_KILL_Y -120.0f
#define CITY_SOFT_X 2400.0f
#define CITY_SOFT_Z 2400.0f
#define CITY_HARD_X 3200.0f
#define CITY_HARD_Z 3200.0f
#define CITY_EDGE_PUSH 0.035f
#define CITY_EDGE_FRICTION 0.975f
#define CITY_GRID_RADIUS 12
#define CITY_BLOCK_SIZE 230.0f
#define CITY_ROAD_SIZE 54.0f
#define CITY_DISTRICTS 5

#define GARAGE_PORTAL_X 0.0f
#define GARAGE_PORTAL_Y 6.0f
#define GARAGE_PORTAL_Z 56.0f
#define GARAGE_PORTAL_RADIUS 6.0f
#define GARAGE_CITY_PORTAL_X -56.0f
#define GARAGE_CITY_PORTAL_Y 6.0f
#define GARAGE_CITY_PORTAL_Z 0.0f
#define GARAGE_CITY_PORTAL_RADIUS 6.0f
#define STADIUM_PORTAL_X 0.0f
#define STADIUM_PORTAL_Y 2.0f
#define STADIUM_PORTAL_Z 0.0f
#define STADIUM_PORTAL_RADIUS 16.0f
#define STADIUM_EDGE_PORTAL_X 406.0f
#define STADIUM_EDGE_PORTAL_Y 2.0f
#define STADIUM_EDGE_PORTAL_Z 0.0f
#define STADIUM_EDGE_PORTAL_RADIUS 14.0f
#define STADIUM_EDGE_TELEPORT_X -360.0f
#define STADIUM_EDGE_TELEPORT_Y 2.0f
#define STADIUM_EDGE_TELEPORT_Z 0.0f
#define VOXWORLD_PORTAL_X -360.0f
#define VOXWORLD_PORTAL_Y 2.0f
#define VOXWORLD_PORTAL_Z 0.0f
#define VOXWORLD_PORTAL_RADIUS 16.0f
#define CITY_PORTAL_X 0.0f
#define CITY_PORTAL_Y 2.0f
#define CITY_PORTAL_Z 0.0f
#define CITY_PORTAL_RADIUS 16.0f
#define PORTAL_ID_GARAGE_EXIT 0
#define PORTAL_ID_STADIUM_TO_VOXWORLD 1
#define PORTAL_ID_VOXWORLD_TO_STADIUM 2
#define PORTAL_ID_CITY_GATE 3

typedef struct {
    int portal_id;
    float x;
    float y;
    float z;
    float radius;
} PortalDef;

typedef struct {
    float x;
    float y;
    float z;
    const char *label;
} VehiclePad;

static const VehiclePad garage_vehicle_pads[] = {
    {-30.0f, 0.0f, -30.0f, "FOXBODY '93"},
    {0.0f, 0.0f, -30.0f, "LANDSHIP"},
    {30.0f, 0.0f, -30.0f, "RESERVED"}
};

float phys_rand_f() { return ((float)(rand()%1000)/500.0f) - 1.0f; }

static inline void init_voxworld_city_geo() {
    if (map_geo_voxworld_init) return;
    map_geo_voxworld_init = 1;
    map_geo_voxworld_count = 0;

    const float block = CITY_BLOCK_SIZE;
    const float road = CITY_ROAD_SIZE;
    const float pitch = block + road;
    const float world_extent = (CITY_GRID_RADIUS + 2) * pitch;

    map_geo_voxworld[map_geo_voxworld_count++] = (Box){0.0f, -2.0f, 0.0f, world_extent * 2.0f, 4.0f, world_extent * 2.0f};

    for (int gx = -CITY_GRID_RADIUS; gx <= CITY_GRID_RADIUS; gx++) {
        for (int gz = -CITY_GRID_RADIUS; gz <= CITY_GRID_RADIUS; gz++) {
            float cx = gx * pitch;
            float cz = gz * pitch;
            if ((abs(gx) <= 1 && gz == 0) || (abs(gz) <= 1 && gx == 0)) continue;

            int district = abs((gx * 131 + gz * 197 + VOXWORLD_SEED * 17) ^ (gx * 53 - gz * 61)) % CITY_DISTRICTS;
            float district_h_min = 20.0f;
            float district_h_var = 50.0f;
            float district_density = 1.0f;
            float alley_bias = 0.0f;
            float landmark_prob = 0.03f;

            if (district == 0) { district_h_min = 42.0f; district_h_var = 95.0f; district_density = 1.2f; landmark_prob = 0.11f; }
            else if (district == 1) { district_h_min = 18.0f; district_h_var = 42.0f; district_density = 1.0f; alley_bias = 0.07f; landmark_prob = 0.04f; }
            else if (district == 2) { district_h_min = 12.0f; district_h_var = 28.0f; district_density = 0.75f; alley_bias = 0.18f; landmark_prob = 0.06f; }
            else if (district == 3) { district_h_min = 14.0f; district_h_var = 36.0f; district_density = 0.65f; alley_bias = 0.26f; landmark_prob = 0.03f; }
            else { district_h_min = 36.0f; district_h_var = 70.0f; district_density = 0.9f; alley_bias = 0.09f; landmark_prob = 0.12f; }

            float n1 = sinf((float)(gx * 17 + gz * 31 + VOXWORLD_SEED) * 0.13f);
            float n2 = cosf((float)(gx * 11 - gz * 23 + VOXWORLD_SEED) * 0.19f);
            float n3 = sinf((float)(gx * 37 + gz * 13 + VOXWORLD_SEED) * 0.29f);
            float h = district_h_min + (n1 + 1.0f) * (district_h_var * 0.6f) + (n2 + 1.0f) * (district_h_var * 0.4f);
            float w = 34.0f + (fabsf(n1) * 54.0f);
            float d = 34.0f + (fabsf(n2) * 54.0f);

            if (n3 > (0.58f - alley_bias)) continue;
            if (district_density < 1.0f && n1 > district_density) continue;

            if (district == 2 && fabsf(n2) > 0.82f) continue; // market plazas

            if (map_geo_voxworld_count + 1 < CITY_MAX_BOXES) {
                map_geo_voxworld[map_geo_voxworld_count++] = (Box){cx, h * 0.5f, cz, w, h, d};
            }
            if (map_geo_voxworld_count + 1 < CITY_MAX_BOXES && (gx + gz) % 3 == 0) {
                map_geo_voxworld[map_geo_voxworld_count++] = (Box){cx + 0.35f * block, (h * 0.35f), cz - 0.3f * block, w * 0.55f, h * 0.7f, d * 0.55f};
            }
            if (map_geo_voxworld_count + 1 < CITY_MAX_BOXES && district == 0 && n3 < -0.72f) {
                map_geo_voxworld[map_geo_voxworld_count++] = (Box){cx - 0.26f * block, h + 26.0f, cz + 0.22f * block, 24.0f, h * 0.9f, 24.0f};
            }
            if (map_geo_voxworld_count + 1 < CITY_MAX_BOXES && district == 1 && ((gx + gz) % 4 == 0)) { // ramps
                map_geo_voxworld[map_geo_voxworld_count++] = (Box){cx + 0.44f * block, 4.0f, cz, 32.0f, 8.0f, 72.0f};
            }
            if (map_geo_voxworld_count + 1 < CITY_MAX_BOXES && district == 3 && ((gx * gz) % 5 == 0)) { // tunnels/ruins
                map_geo_voxworld[map_geo_voxworld_count++] = (Box){cx, 5.0f, cz - 0.45f * block, 120.0f, 10.0f, 20.0f};
            }
            if (map_geo_voxworld_count + 1 < CITY_MAX_BOXES && fabsf(n2) < landmark_prob) {
                float tower_h = h + 80.0f + fabsf(n1) * 120.0f;
                map_geo_voxworld[map_geo_voxworld_count++] = (Box){cx, tower_h * 0.5f, cz, 22.0f, tower_h, 22.0f};
            }
        }
    }

    if (map_geo_voxworld_count > CITY_MAX_BOXES - 256) {
        printf("[city] warning: box usage high %d/%d\n", map_geo_voxworld_count, CITY_MAX_BOXES);
    }
}

static int phys_scene_id = SCENE_STADIUM;

static inline void phys_set_scene(int scene_id) {
    phys_scene_id = scene_id;
    if (scene_id == SCENE_GARAGE_OSAKA) {
        map_geo = map_geo_garage;
        map_count = (int)(sizeof(map_geo_garage) / sizeof(Box));
    } else if (scene_id == SCENE_VOXWORLD || scene_id == SCENE_CITY) {
        init_voxworld_city_geo();
        map_geo = map_geo_voxworld;
        map_count = map_geo_voxworld_count;
    } else {
        map_geo = map_geo_stadium;
        map_count = (int)(sizeof(map_geo_stadium) / sizeof(Box));
    }
}

static inline void scene_spawn_point(int scene_id, int slot, float *out_x, float *out_y, float *out_z) {
    if (scene_id == SCENE_GARAGE_OSAKA) {
        float offsets[] = {-20.0f, 0.0f, 20.0f, -10.0f, 10.0f};
        int idx = slot % 5;
        *out_x = offsets[idx];
        *out_y = 2.0f;
        *out_z = 20.0f;
        return;
    }
    if (scene_id == SCENE_VOXWORLD) {
        float offsets[] = {-120.0f, -60.0f, 0.0f, 60.0f, 120.0f};
        int idx = slot % 5;
        *out_x = offsets[idx];
        *out_y = 6.0f;
        *out_z = 0.0f;
        return;
    }
    if (scene_id == SCENE_CITY) {
        init_voxworld_city_geo();
        const float pitch = CITY_BLOCK_SIZE + CITY_ROAD_SIZE;
        static const int anchors[][2] = {
            {0, 0}, {2, -1}, {-3, 2}, {4, 3}, {-4, -2}, {1, 4}, {-2, -4}, {6, 0}, {0, -6}
        };
        int idx = abs(slot) % (int)(sizeof(anchors) / sizeof(anchors[0]));
        *out_x = anchors[idx][0] * pitch;
        *out_y = 6.0f;
        *out_z = anchors[idx][1] * pitch;

        for (int attempt = 0; attempt < 8; attempt++) {
            int collision = 0;
            for (int i = 1; i < map_geo_voxworld_count; i++) {
                Box b = map_geo_voxworld[i];
                if (fabsf(*out_x - b.x) < (b.w * 0.5f + 6.0f) && fabsf(*out_z - b.z) < (b.d * 0.5f + 6.0f) && 6.0f < b.y + b.h * 0.5f) {
                    collision = 1;
                    break;
                }
            }
            if (!collision) break;
            *out_x += 0.5f * pitch;
            *out_z -= 0.35f * pitch;
        }
        return;
    }
    if (slot % 2 == 0) {
        *out_x = 0.0f; *out_z = 0.0f; *out_y = 80.0f;
    } else {
        float ang = phys_rand_f() * 6.28f;
        *out_x = sinf(ang) * 500.0f;
        *out_z = cosf(ang) * 500.0f;
        *out_y = 20.0f;
    }
}

static inline void scene_force_spawn(PlayerState *p) {
    float sx = 0.0f, sy = 0.0f, sz = 0.0f;
    phys_set_scene(p->scene_id);
    scene_spawn_point(p->scene_id, p->id, &sx, &sy, &sz);
    p->x = sx; p->y = sy; p->z = sz;
    p->vx = 0.0f; p->vy = 0.0f; p->vz = 0.0f;
}

static inline void scene_safety_check(PlayerState *p) {
    if (!isfinite(p->x) || !isfinite(p->y) || !isfinite(p->z)) {
        scene_force_spawn(p);
        return;
    }
    if (p->scene_id == SCENE_GARAGE_OSAKA) {
        if (p->y < GARAGE_KILL_Y ||
            p->x < -GARAGE_BOUNDS_X || p->x > GARAGE_BOUNDS_X ||
            p->z < -GARAGE_BOUNDS_Z || p->z > GARAGE_BOUNDS_Z) {
            scene_force_spawn(p);
        }
        return;
    }
    if (p->scene_id == SCENE_STADIUM) {
        if (p->y < STADIUM_KILL_Y ||
            p->x < -STADIUM_BOUNDS_X || p->x > STADIUM_BOUNDS_X ||
            p->z < -STADIUM_BOUNDS_Z || p->z > STADIUM_BOUNDS_Z) {
            scene_force_spawn(p);
        }
        return;
    }
    if (p->scene_id == SCENE_VOXWORLD) {
        if (p->y < VOXWORLD_KILL_Y ||
            p->x < -VOXWORLD_BOUNDS_X || p->x > VOXWORLD_BOUNDS_X ||
            p->z < -VOXWORLD_BOUNDS_Z || p->z > VOXWORLD_BOUNDS_Z) {
            scene_force_spawn(p);
        }
        return;
    }
    if (p->scene_id == SCENE_CITY) {
        if (p->y < CITY_KILL_Y) {
            scene_force_spawn(p);
            return;
        }

        float out_x = fabsf(p->x) - CITY_SOFT_X;
        float out_z = fabsf(p->z) - CITY_SOFT_Z;
        if (out_x > 0.0f) {
            float dir = (p->x > 0.0f) ? -1.0f : 1.0f;
            p->vx += dir * CITY_EDGE_PUSH * (1.0f + out_x / 320.0f);
            p->vx *= CITY_EDGE_FRICTION;
            p->vz *= 0.992f;
        }
        if (out_z > 0.0f) {
            float dir = (p->z > 0.0f) ? -1.0f : 1.0f;
            p->vz += dir * CITY_EDGE_PUSH * (1.0f + out_z / 320.0f);
            p->vz *= CITY_EDGE_FRICTION;
            p->vx *= 0.992f;
        }

    }
}

static inline int scene_portal_active(int scene_id) {
    return scene_id == SCENE_GARAGE_OSAKA || scene_id == SCENE_STADIUM || scene_id == SCENE_CITY || scene_id == SCENE_VOXWORLD;
}

static inline int scene_portals(int scene_id, PortalDef *out, int max) {
    if (!out || max <= 0) return 0;
    int count = 0;
    if (scene_id == SCENE_GARAGE_OSAKA) {
        if (count < max) {
            out[count++] = (PortalDef){PORTAL_ID_GARAGE_EXIT, GARAGE_PORTAL_X, GARAGE_PORTAL_Y, GARAGE_PORTAL_Z, GARAGE_PORTAL_RADIUS};
        }
        if (count < max) {
            out[count++] = (PortalDef){PORTAL_ID_CITY_GATE, GARAGE_CITY_PORTAL_X, GARAGE_CITY_PORTAL_Y, GARAGE_CITY_PORTAL_Z, GARAGE_CITY_PORTAL_RADIUS};
        }
        return count;
    }
    if (scene_id == SCENE_STADIUM) {
        out[count++] = (PortalDef){PORTAL_ID_GARAGE_EXIT, STADIUM_PORTAL_X, STADIUM_PORTAL_Y, STADIUM_PORTAL_Z, STADIUM_PORTAL_RADIUS};
        return count;
    }
    if (scene_id == SCENE_CITY) {
        out[count++] = (PortalDef){PORTAL_ID_CITY_GATE, CITY_PORTAL_X, CITY_PORTAL_Y, CITY_PORTAL_Z, CITY_PORTAL_RADIUS};
        return count;
    }
    if (scene_id == SCENE_VOXWORLD) {
        out[count++] = (PortalDef){PORTAL_ID_VOXWORLD_TO_STADIUM, VOXWORLD_PORTAL_X, VOXWORLD_PORTAL_Y, VOXWORLD_PORTAL_Z, VOXWORLD_PORTAL_RADIUS};
        return count;
    }
    return 0;
}

static inline int portal_resolve_destination(int current_scene, int portal_id, int slot,
                                             int *out_scene, float *out_x, float *out_y, float *out_z) {
    if (!out_scene || !out_x || !out_y || !out_z) return 0;
    if (current_scene == SCENE_GARAGE_OSAKA && portal_id == PORTAL_ID_GARAGE_EXIT) {
        *out_scene = SCENE_STADIUM;
        scene_spawn_point(*out_scene, slot, out_x, out_y, out_z);
        return 1;
    }
    if (current_scene == SCENE_STADIUM && portal_id == PORTAL_ID_GARAGE_EXIT) {
        *out_scene = SCENE_GARAGE_OSAKA;
        scene_spawn_point(*out_scene, slot, out_x, out_y, out_z);
        return 1;
    }
    if (current_scene == SCENE_GARAGE_OSAKA && portal_id == PORTAL_ID_CITY_GATE) {
        *out_scene = SCENE_CITY;
        scene_spawn_point(*out_scene, slot, out_x, out_y, out_z);
        return 1;
    }
    if (current_scene == SCENE_CITY && portal_id == PORTAL_ID_CITY_GATE) {
        *out_scene = SCENE_GARAGE_OSAKA;
        scene_spawn_point(*out_scene, slot, out_x, out_y, out_z);
        return 1;
    }
    if (current_scene == SCENE_STADIUM && portal_id == PORTAL_ID_STADIUM_TO_VOXWORLD) {
        *out_scene = SCENE_VOXWORLD;
        *out_x = STADIUM_EDGE_TELEPORT_X;
        *out_y = STADIUM_EDGE_TELEPORT_Y;
        *out_z = STADIUM_EDGE_TELEPORT_Z;
        return 1;
    }
    if (current_scene == SCENE_VOXWORLD && portal_id == PORTAL_ID_VOXWORLD_TO_STADIUM) {
        *out_scene = SCENE_STADIUM;
        *out_x = STADIUM_EDGE_PORTAL_X - 20.0f;
        *out_y = STADIUM_EDGE_PORTAL_Y;
        *out_z = STADIUM_EDGE_PORTAL_Z;
        return 1;
    }
    return 0;
}

static inline void scene_portal_info(int scene_id, float *out_x, float *out_y, float *out_z, float *out_radius) {
    PortalDef portals[4];
    int count = scene_portals(scene_id, portals, 4);
    if (count > 0) {
        *out_x = portals[0].x;
        *out_y = portals[0].y;
        *out_z = portals[0].z;
        *out_radius = portals[0].radius;
        return;
    }
    *out_x = 0.0f; *out_y = 0.0f; *out_z = 0.0f; *out_radius = 0.0f;
}

static inline const VehiclePad *scene_vehicle_pads(int scene_id, int *out_count) {
    if (scene_id == SCENE_GARAGE_OSAKA) {
        if (out_count) *out_count = (int)(sizeof(garage_vehicle_pads) / sizeof(VehiclePad));
        return garage_vehicle_pads;
    }
    if (out_count) *out_count = 0;
    return NULL;
}

static inline int scene_portal_triggered(PlayerState *p, int *out_portal_id) {
    if (!scene_portal_active(p->scene_id)) return 0;

    PortalDef portals[4];
    int portal_count = scene_portals(p->scene_id, portals, 4);
    for (int i = 0; i < portal_count; i++) {
        float dx = p->x - portals[i].x;
        float dz = p->z - portals[i].z;
        float dist_sq = dx * dx + dz * dz;
        if (dist_sq <= (portals[i].radius * portals[i].radius)) {
            if (out_portal_id) *out_portal_id = portals[i].portal_id;
            return 1;
        }
    }
    return 0;
}

static inline int scene_near_vehicle_pad(int scene_id, float x, float z, float max_dist, int *out_idx) {
    int count = 0;
    const VehiclePad *pads = scene_vehicle_pads(scene_id, &count);
    if (!pads || count == 0) return 0;
    float best_dist_sq = max_dist * max_dist;
    int best_idx = -1;
    for (int i = 0; i < count; i++) {
        float dx = x - pads[i].x;
        float dz = z - pads[i].z;
        float dist_sq = dx * dx + dz * dz;
        if (dist_sq <= best_dist_sq) {
            best_dist_sq = dist_sq;
            best_idx = i;
        }
    }
    if (best_idx >= 0) {
        if (out_idx) *out_idx = best_idx;
        return 1;
    }
    return 0;
}

static inline void apply_friction_2d(Vec2 *vel, float friction, float dt) {
    float speed = sqrtf(vel->x * vel->x + vel->y * vel->y);
    if (speed <= 0.0001f) return;
    float drop = speed * friction * dt;
    float newspeed = speed - drop;
    if (newspeed < 0.0f) newspeed = 0.0f;
    float ratio = newspeed / speed;
    vel->x *= ratio;
    vel->y *= ratio;
}

static inline float vox_hash_noise(float x, float z) {
    float n = sinf(x * 12.9898f + z * 78.233f + (float)VOXWORLD_SEED * 0.001f) * 43758.5453f;
    return n - floorf(n);
}

static inline float phys_vox_height_at(float x, float z) {
    if (x < -VOXWORLD_BOUNDS_X || x > VOXWORLD_BOUNDS_X ||
        z < -VOXWORLD_BOUNDS_Z || z > VOXWORLD_BOUNDS_Z) {
        return -1000.0f;
    }
    float base = 5.0f;
    float waves = sinf((x + (float)VOXWORLD_SEED) * 0.01f) * 3.0f +
                  cosf((z - (float)VOXWORLD_SEED) * 0.012f) * 2.5f;
    float detail = (vox_hash_noise(x * 0.2f, z * 0.2f) - 0.5f) * 2.0f;
    float h = base + waves + detail;
    if (h < 0.0f) h = 0.0f;
    return h;
}

static inline void phys_vox_normal_at(float x, float z, float *nx, float *ny, float *nz) {
    float hL = phys_vox_height_at(x - 1.0f, z);
    float hR = phys_vox_height_at(x + 1.0f, z);
    float hD = phys_vox_height_at(x, z - 1.0f);
    float hU = phys_vox_height_at(x, z + 1.0f);
    float gx = hR - hL;
    float gz = hU - hD;
    float len = sqrtf(gx * gx + 4.0f + gz * gz);
    if (len < 0.0001f) {
        *nx = 0.0f; *ny = 1.0f; *nz = 0.0f;
        return;
    }
    *nx = -gx / len;
    *ny = 2.0f / len;
    *nz = -gz / len;
}

static inline float norm_yaw_deg(float yaw) {
    while (yaw >= 360.0f) yaw -= 360.0f;
    while (yaw < 0.0f) yaw += 360.0f;
    return yaw;
}

static inline float clamp_pitch_deg(float pitch) {
    if (pitch > 89.0f) pitch = 89.0f;
    if (pitch < -89.0f) pitch = -89.0f;
    return pitch;
}

static inline float angle_diff(float a, float b) {
    float d = a - b;
    while (d < -180) d += 360;
    while (d > 180) d -= 360;
    return d;
}

void reflect_vector(float *vx, float *vy, float *vz, float nx, float ny, float nz) {
    float dot = (*vx * nx) + (*vy * ny) + (*vz * nz);
    *vx = *vx - 2.0f * dot * nx;
    *vy = *vy - 2.0f * dot * ny;
    *vz = *vz - 2.0f * dot * nz;
}

static inline int trace_map_boxes(float x1, float y1, float z1, float x2, float y2, float z2,
              float *out_x, float *out_y, float *out_z, float *nx, float *ny, float *nz) {
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        if (x2 > b.x - b.w/2 && x2 < b.x + b.w/2 &&
            z2 > b.z - b.d/2 && z2 < b.z + b.d/2 &&
            y2 > b.y - b.h/2 && y2 < b.y + b.h/2) {
            float dx = x1 - b.x; float dz = z1 - b.z;
            float w = b.w; float d = b.d;
            if (fabs(dx)/w > fabs(dz)/d) {
                *nx = (dx > 0) ? 1.0f : -1.0f; *ny = 0.0f; *nz = 0.0f;
                *out_x = (dx > 0) ? b.x + b.w/2 + 0.1f : b.x - b.w/2 - 0.1f;
                *out_y = y2; *out_z = z2;
            } else {
                *nx = 0.0f; *ny = 0.0f; *nz = (dz > 0) ? 1.0f : -1.0f;
                *out_x = x2; *out_y = y2;
                *out_z = (dz > 0) ? b.z + b.d/2 + 0.1f : b.z - b.d/2 - 0.1f;
            }
            return 1;
        }
    }
    if (y2 < 0.0f) {
        *nx = 0.0f; *ny = 1.0f; *nz = 0.0f;
        *out_x = x2; *out_y = 0.1f; *out_z = z2;
        return 1;
    }
    return 0;
}

static inline int trace_map_vox(float x1, float y1, float z1, float x2, float y2, float z2,
              float *out_x, float *out_y, float *out_z, float *nx, float *ny, float *nz) {
    float dx = x2 - x1, dy = y2 - y1, dz = z2 - z1;
    float dist = sqrtf(dx * dx + dy * dy + dz * dz);
    int steps = (int)(dist / VOXWORLD_TRACE_STEP) + 2;
    if (steps < 2) steps = 2;
    for (int i = 1; i <= steps; i++) {
        float t = (float)i / (float)steps;
        float sx = x1 + dx * t;
        float sy = y1 + dy * t;
        float sz = z1 + dz * t;
        float h = phys_vox_height_at(sx, sz);
        if (sy <= h) {
            *out_x = sx;
            *out_y = h + 0.1f;
            *out_z = sz;
            phys_vox_normal_at(sx, sz, nx, ny, nz);
            return 1;
        }
    }
    return 0;
}

int trace_map(float x1, float y1, float z1, float x2, float y2, float z2,
              float *out_x, float *out_y, float *out_z, float *nx, float *ny, float *nz) {
    if (phys_scene_id == SCENE_VOXWORLD) {
        return trace_map_vox(x1, y1, z1, x2, y2, z2, out_x, out_y, out_z, nx, ny, nz);
    }
    return trace_map_boxes(x1, y1, z1, x2, y2, z2, out_x, out_y, out_z, nx, ny, nz);
}

int check_hit_location(float ox, float oy, float oz, float dx, float dy, float dz, PlayerState *target) {
    if (!target->active) return 0;
    float tx = target->x; float tz = target->z;
    float h_size = target->in_vehicle ? 4.0f : HEAD_SIZE;
    float h_off = target->in_vehicle ? 2.0f : HEAD_OFFSET;
    float head_y = target->y + h_off;
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

static void vehicle_stabilize(PlayerState *p, float dt) {
    if (!p || !p->in_vehicle || dt <= 0.0f) return;

    float speed = sqrtf(p->vx * p->vx + p->vz * p->vz);
    if (speed < 0.001f) return;

    float yaw_rad = p->yaw * 0.0174532925f;
    float fwd_x = sinf(yaw_rad);
    float fwd_z = cosf(yaw_rad);
    float right_x = cosf(yaw_rad);
    float right_z = -sinf(yaw_rad);

    float v_fwd = p->vx * fwd_x + p->vz * fwd_z;
    float v_lat = p->vx * right_x + p->vz * right_z;

    int drifting = p->crouching != 0;
    float lateral_grip = BUGGY_LATERAL_GRIP;
    float align_rate = BUGGY_ALIGN_RATE;
    if (p->vehicle_type == VEH_BIKE) {
        lateral_grip = BIKE_LATERAL_GRIP;
        align_rate = BIKE_ALIGN_RATE;
    }
    if (drifting) {
        lateral_grip = (p->vehicle_type == VEH_BIKE) ? BIKE_DRIFT_LATERAL_GRIP : BUGGY_DRIFT_LATERAL_GRIP;
        align_rate = 0.0f;
    }

    float lat_damp = expf(-lateral_grip * dt);
    v_lat *= lat_damp;

    p->vx = fwd_x * v_fwd + right_x * v_lat;
    p->vz = fwd_z * v_fwd + right_z * v_lat;

    speed = sqrtf(p->vx * p->vx + p->vz * p->vz);
    if (drifting || speed < VEH_MIN_GRIP_SPEED) return;

    float inv_speed = 1.0f / speed;
    float vel_x = p->vx * inv_speed;
    float vel_z = p->vz * inv_speed;

    float blend = 1.0f - expf(-align_rate * dt);
    float dir_x = vel_x + (fwd_x - vel_x) * blend;
    float dir_z = vel_z + (fwd_z - vel_z) * blend;
    float dir_len = sqrtf(dir_x * dir_x + dir_z * dir_z);
    if (dir_len < 0.001f) return;

    dir_x /= dir_len;
    dir_z /= dir_len;
    p->vx = dir_x * speed;
    p->vz = dir_z * speed;
}

void apply_friction(PlayerState *p, float dt) {
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (speed < 0.001f) { p->vx = 0; p->vz = 0; return; }

    float drop = 0;
    if (p->in_vehicle) {
        float vehicle_friction = (p->vehicle_type == VEH_BIKE) ? BIKE_FRICTION : BUGGY_FRICTION;
        drop = speed * vehicle_friction;
    }
    else if (p->on_ground) {
        if (p->crouching) {
            if (speed > 0.75f) drop = speed * SLIDE_FRICTION;
            else drop = speed * (FRICTION * 3.0f);
        } else {
            float control = (speed < STOP_SPEED) ? STOP_SPEED : speed;
            drop = control * FRICTION;
        }
    }
    float newspeed = speed - drop;
    if (newspeed < 0) newspeed = 0;
    newspeed /= speed;
    p->vx *= newspeed; p->vz *= newspeed;

    if (p->in_vehicle) vehicle_stabilize(p, dt);
}


void accelerate(PlayerState *p, float wish_x, float wish_z, float wish_speed, float accel) {
    if (p->in_vehicle) {
        float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
        float add_speed = wish_speed - current_speed;
        if (add_speed <= 0) return;
        float vehicle_max_speed = (p->vehicle_type == VEH_BIKE) ? BIKE_MAX_SPEED : BUGGY_MAX_SPEED;
        float acc_speed = accel * vehicle_max_speed;
        if (acc_speed > add_speed) acc_speed = add_speed;
        p->vx += acc_speed * wish_x; p->vz += acc_speed * wish_z;
        return;
    }
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    if (p->crouching && speed > 0.75f && p->on_ground) return;
    if (p->crouching && p->on_ground && speed < 0.75f && wish_speed > CROUCH_SPEED) wish_speed = CROUCH_SPEED;
    float current_speed = (p->vx * wish_x) + (p->vz * wish_z);
    float add_speed = wish_speed - current_speed;
    if (add_speed <= 0) return;
    float acc_speed = accel * MAX_SPEED; 
    if (acc_speed > add_speed) acc_speed = add_speed;
    p->vx += acc_speed * wish_x; p->vz += acc_speed * wish_z;
}

void resolve_collision(PlayerState *p) {
    float pw = p->in_vehicle ? 3.0f : PLAYER_WIDTH;
    float ph = p->in_vehicle ? 3.0f : (p->crouching ? (PLAYER_HEIGHT / 2.0f) : PLAYER_HEIGHT);
    p->on_ground = 0;
    if (phys_scene_id == SCENE_VOXWORLD) {
        float floor_y = phys_vox_height_at(p->x, p->z);
        if (p->y < floor_y) { p->y = floor_y; p->vy = 0; p->on_ground = 1; }
        return;
    }
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
                        p->vx = 0; p->x = (dx > 0) ? b.x + b.w/2 + pw : b.x - b.w/2 - pw;
                    } else { 
                        p->vz = 0; p->z = (dz > 0) ? b.z + b.d/2 + pw : b.z - b.d/2 - pw;
                    }
                }
            }
        }
    }
}

void phys_respawn(PlayerState *p, unsigned int now) {
    p->active = 1; p->state = STATE_ALIVE;
    p->health = 100; p->shield = 100; p->respawn_time = 0; p->in_vehicle = 0;
    p->vehicle_type = VEH_NONE;
    p->bike_gear = 0;
    p->in_bike = 0;
    p->use_was_down = 0;
    p->bike_was_down = 0;
    if (p->scene_id != SCENE_GARAGE_OSAKA && p->scene_id != SCENE_STADIUM && p->scene_id != SCENE_VOXWORLD && p->scene_id != SCENE_CITY) {
        p->scene_id = SCENE_GARAGE_OSAKA;
    }
    scene_spawn_point(p->scene_id, p->id, &p->x, &p->y, &p->z);
    p->current_weapon = WPN_MAGNUM;
    for(int i=0; i<MAX_WEAPONS; i++) p->ammo[i] = WPN_STATS[i].ammo_max;
    p->storm_charges = 0;
    p->ability_cooldown = 0;
    p->portal_cooldown_until_ms = 0;
    p->stunned_until_ms = 0;
    p->stun_immune_until_ms = 0;
    if (p->is_bot) {
        PlayerState *winner = get_best_bot();
        if (winner && winner != p) evolve_bot(p, winner);
    }
}

static inline void spawn_projectile(Projectile *projectiles, PlayerState *p, int damage, int bounces, float speed_mult) {
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if (!projectiles[i].active) {
            Projectile *proj = &projectiles[i];
            proj->active = 1;
            proj->owner_id = p->id;
            proj->damage = damage;
            proj->bounces_left = bounces;
            proj->scene_id = (unsigned char)p->scene_id;

            float r = -p->yaw * 0.0174533f; float rp = p->pitch * 0.0174533f;
            float speed = 4.0f * speed_mult;
            proj->vx = sinf(r) * cosf(rp) * speed;
            proj->vy = sinf(rp) * speed;
            proj->vz = -cosf(r) * cosf(rp) * speed;
            proj->x = p->x;
            proj->y = p->y + EYE_HEIGHT;
            proj->z = p->z;
            return;
        }
    }
}

void update_weapons(PlayerState *p, PlayerState *targets, Projectile *projectiles, int shoot, int reload, int ability_press) {
    if (p->in_vehicle) return; 
    if (p->reload_timer > 0) p->reload_timer--;
    if (p->attack_cooldown > 0) p->attack_cooldown--;
    if (p->is_shooting > 0) p->is_shooting--;
    if (p->ability_cooldown > 0) p->ability_cooldown--;

    if (ability_press && p->ability_cooldown == 0 && p->storm_charges == 0) {
        p->storm_charges = 5;
        p->ability_cooldown = 480;
    }

    int w = p->current_weapon;
    if (reload && p->reload_timer == 0 && w != WPN_KNIFE) {
        if (p->ammo[w] < WPN_STATS[w].ammo_max) {
            if (p->ammo[w] > 0) p->reload_timer = RELOAD_TIME_TACTICAL;
            else p->reload_timer = RELOAD_TIME_FULL; 
        }
    }
    if (p->reload_timer == 1) p->ammo[w] = WPN_STATS[w].ammo_max;
    if (shoot && p->attack_cooldown == 0 && p->reload_timer == 0) {
        if (p->storm_charges > 0 && w == WPN_SNIPER) {
            int storm_damage = (int)(WPN_STATS[w].dmg * 0.7f);
            spawn_projectile(projectiles, p, storm_damage, 1, 1.5f);
            p->storm_charges--;
            p->attack_cooldown = 8;
            p->recoil_anim = 0.5f;
            return;
        }
        if (w != WPN_KNIFE && p->ammo[w] <= 0) p->reload_timer = RELOAD_TIME_FULL;
        else {
            p->is_shooting = 5; p->recoil_anim = 1.0f;
            p->attack_cooldown = WPN_STATS[w].rof;
            if (w != WPN_KNIFE) p->ammo[w]--;
            
            float r = -p->yaw * 0.0174533f; float rp = p->pitch * 0.0174533f;
            float dx = sinf(r) * cosf(rp); float dy = sinf(rp); float dz = -cosf(r) * cosf(rp);
            if (WPN_STATS[w].spr > 0) {
                dx += phys_rand_f() * WPN_STATS[w].spr;
                dy += phys_rand_f() * WPN_STATS[w].spr;
                dz += phys_rand_f() * WPN_STATS[w].spr;
            }

            for(int i=0; i<MAX_CLIENTS; i++) {
                if (p == &targets[i]) continue;
                if (!targets[i].active || targets[i].state == STATE_DEAD) continue;
                if (targets[i].scene_id != p->scene_id) continue;
                if (w == WPN_KNIFE) {
                    float kx = p->x - targets[i].x;
                    float ky = p->y - targets[i].y; float kz = p->z - targets[i].z;
                    if ((kx*kx + ky*ky + kz*kz) > MELEE_RANGE_SQ + 22.0f ) continue;
                }
                int hit_type = check_hit_location(p->x, p->y + EYE_HEIGHT, p->z, dx, dy, dz, &targets[i]);
                if (hit_type > 0) {
                    printf("🔫 HIT! Dmg: %d on Target %d\n", WPN_STATS[w].dmg, i);
                    int damage = WPN_STATS[w].dmg;
                    p->accumulated_reward += 10.0f;
                    targets[i].shield_regen_timer = SHIELD_REGEN_DELAY;
                    if (hit_type == 2 && targets[i].shield <= 0) { damage *= 3; p->hit_feedback = 20;
                    } else { p->hit_feedback = 10; } 
                    
                    if (targets[i].shield > 0) {
                        if (targets[i].shield >= damage) { targets[i].shield -= damage; damage = 0; } 
                        else { damage -= targets[i].shield; targets[i].shield = 0; }
                    }
                    targets[i].health -= damage;
                    if(targets[i].health <= 0) {
                        p->kills++; targets[i].deaths++; 
                        p->accumulated_reward += 1000.0f;
                        p->hit_feedback = 30; // KILL CONFIRM (Triggers Double Ring)
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
