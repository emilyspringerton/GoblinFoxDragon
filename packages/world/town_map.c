#include "town_map.h"

#define TOWN_WORLD_SCALE 3.0f

typedef struct {
    TownBuildingId id;
    const char *label;
    float x, z;
    float w, d;
    float h;
    unsigned int tags;
} TownBuildingSeed;

static const TownBuildingSeed g_core_buildings[] = {
    {BLD_AUCTION_HOUSE, "Auction House", 60, 58, 28, 22, 30, TOWN_BLD_TAG_ENTERABLE | TOWN_BLD_TAG_DISTRICT_CIVIC},
    {BLD_TOWN_HALL, "Town Hall", 60, 80, 22, 18, 24, TOWN_BLD_TAG_ENTERABLE | TOWN_BLD_TAG_DISTRICT_CIVIC},
    {BLD_GUILD_HOUSE, "Guild House", 40, 44, 24, 18, 18, TOWN_BLD_TAG_ENTERABLE | TOWN_BLD_TAG_DISTRICT_CIVIC},
    {BLD_GOLD_GUILD, "Gold Guild", 84, 92, 22, 18, 18, TOWN_BLD_TAG_DISTRICT_MARKET},
    {BLD_POST_OFFICE, "Post Office", 12, 106, 24, 20, 14, TOWN_BLD_TAG_DISTRICT_BRUSH},
    {BLD_BLACKSMITH, "Blacksmith", 28, 128, 24, 18, 14, TOWN_BLD_TAG_ENTERABLE | TOWN_BLD_TAG_DISTRICT_BRUSH},
    {BLD_WEAPONS_GUILD, "Weapons Guild", 12, 156, 34, 22, 18, TOWN_BLD_TAG_DISTRICT_BRUSH},
    {BLD_POTIONS, "Potions Shop", 94, 144, 34, 24, 20, TOWN_BLD_TAG_DISTRICT_MARKET},
    {BLD_ALCHEMY_SHOP, "Alchemy Shop", 112, 130, 24, 18, 16, TOWN_BLD_TAG_DISTRICT_MARKET},
    {BLD_SHADY_DEALER, "Back Alley Market", 112, 96, 30, 28, 20, TOWN_BLD_TAG_ENTERABLE | TOWN_BLD_TAG_DISTRICT_MARKET},
    {BLD_FISH_SHOP, "Fishmonger", 2, 60, 34, 20, 16, TOWN_BLD_TAG_DISTRICT_WAREHOUSE},
    {BLD_ARMOR_SHOP, "Armor Shop", 2, 86, 32, 20, 16, TOWN_BLD_TAG_DISTRICT_WAREHOUSE},
    {BLD_MINECO_OPS, "Mining Co-op", 22, 88, 20, 14, 12, TOWN_BLD_TAG_DISTRICT_BRUSH},
    {BLD_MINING_SUPPLIES, "Mining Supplies", 28, 46, 20, 14, 12, TOWN_BLD_TAG_DISTRICT_WAREHOUSE},
    {BLD_ARCHERY_GUILD, "Archery Guild", 132, 34, 30, 20, 22, TOWN_BLD_TAG_DISTRICT_FINANCIAL},
    {BLD_POLICE, "Town Watch", 150, 26, 28, 24, 24, TOWN_BLD_TAG_ENTERABLE | TOWN_BLD_TAG_DISTRICT_FINANCIAL},
    {BLD_GLOVE_SHOP, "Market Exchange", 122, 14, 24, 18, 18, TOWN_BLD_TAG_DISTRICT_FINANCIAL},
};

#include "generated/town_expansion_blocks.inl"

static const CrisisSocket g_sockets[] = {
    {SOCK_ANCHOR_AUCTION, "Anchor: AUCTION SQUARE", 60, 58, 3.5f, SOCK_ROLE_BUILDER, 1},
    {SOCK_RITUAL_TOWN_HALL, "Anchor: TOWN HALL", 60, 86, 3.5f, SOCK_ROLE_RITUALIST, 1},
    {SOCK_INTERCEPT_DOCK_ROUTE, "Anchor: STADIUM ROAD", 96, 138, 4.0f, SOCK_ROLE_STRIKE | SOCK_ROLE_SCOUT, 1},
    {SOCK_INTERCEPT_MINES_ROUTE, "Anchor: MARKET WAY", 120, 24, 4.0f, SOCK_ROLE_STRIKE | SOCK_ROLE_SCOUT, 1},
    {SOCK_HEAD_A_DOCKS, "Anchor: WATERFRONT GATE", 8, 62, 4.5f, SOCK_ROLE_STRIKE, 0},
    {SOCK_HEAD_B_MINES, "Anchor: BRUSH GATE", 16, 118, 4.5f, SOCK_ROLE_STRIKE, 0},
    {SOCK_SECRET_GATE_PRESSURE, "Anchor: RIVER GATE", 146, 18, 3.0f, SOCK_ROLE_SCOUT, 1}
};

static const TownRoutePoint g_routes[] = {
    {"Auction Square", 60, 58},
    {"Town Hall Way", 60, 86},
    {"Stadium Road", 96, 138},
    {"Brush Blocks", 20, 120},
    {"Waterfront Gate", 8, 62},
    {"Market Way", 120, 24}
};

#define CORE_BUILDING_COUNT (sizeof(g_core_buildings) / sizeof(g_core_buildings[0]))
#define EXPANSION_BUILDING_COUNT (sizeof(g_expansion_seed_buildings) / sizeof(g_expansion_seed_buildings[0]))
#define TOTAL_BUILDING_COUNT (CORE_BUILDING_COUNT + EXPANSION_BUILDING_COUNT)

static TownBuilding g_buildings_scaled[TOTAL_BUILDING_COUNT];
static CrisisSocket g_sockets_scaled[sizeof(g_sockets) / sizeof(g_sockets[0])];
static TownRoutePoint g_routes_scaled[sizeof(g_routes) / sizeof(g_routes[0])];
static int g_scaled_init = 0;

static void town_map_scale_seed(size_t out_idx, const TownBuildingSeed *seed) {
    g_buildings_scaled[out_idx].id = seed->id;
    g_buildings_scaled[out_idx].label = seed->label;
    g_buildings_scaled[out_idx].x = seed->x * TOWN_WORLD_SCALE;
    g_buildings_scaled[out_idx].z = seed->z * TOWN_WORLD_SCALE;
    g_buildings_scaled[out_idx].w = seed->w * TOWN_WORLD_SCALE;
    g_buildings_scaled[out_idx].d = seed->d * TOWN_WORLD_SCALE;
    g_buildings_scaled[out_idx].h = seed->h * TOWN_WORLD_SCALE;
    g_buildings_scaled[out_idx].tags = seed->tags;
}

static void town_map_init_scaled() {
    if (g_scaled_init) return;
    g_scaled_init = 1;

    for (size_t i = 0; i < CORE_BUILDING_COUNT; i++) {
        town_map_scale_seed(i, &g_core_buildings[i]);
    }
    for (size_t i = 0; i < EXPANSION_BUILDING_COUNT; i++) {
        town_map_scale_seed(CORE_BUILDING_COUNT + i, &g_expansion_seed_buildings[i]);
    }

    for (size_t i = 0; i < sizeof(g_sockets_scaled) / sizeof(g_sockets_scaled[0]); i++) {
        g_sockets_scaled[i] = g_sockets[i];
        g_sockets_scaled[i].x *= TOWN_WORLD_SCALE;
        g_sockets_scaled[i].z *= TOWN_WORLD_SCALE;
        g_sockets_scaled[i].radius *= TOWN_WORLD_SCALE;
    }

    for (size_t i = 0; i < sizeof(g_routes_scaled) / sizeof(g_routes_scaled[0]); i++) {
        g_routes_scaled[i] = g_routes[i];
        g_routes_scaled[i].x *= TOWN_WORLD_SCALE;
        g_routes_scaled[i].z *= TOWN_WORLD_SCALE;
    }
}

const TownBuilding *town_map_buildings(size_t *count) {
    town_map_init_scaled();
    if (count) *count = TOTAL_BUILDING_COUNT;
    return g_buildings_scaled;
}

const CrisisSocket *town_map_sockets(size_t *count) {
    town_map_init_scaled();
    if (count) *count = sizeof(g_sockets_scaled) / sizeof(g_sockets_scaled[0]);
    return g_sockets_scaled;
}

const TownRoutePoint *town_map_route_points(size_t *count) {
    town_map_init_scaled();
    if (count) *count = sizeof(g_routes_scaled) / sizeof(g_routes_scaled[0]);
    return g_routes_scaled;
}
