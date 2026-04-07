#include "town_map.h"

#define TOWN_WORLD_SCALE 3.0f

static const TownBuilding g_buildings[] = {
    {BLD_AUCTION_HOUSE, "Campus Tower", 60, 52, 20, 16, 24, 0},
    {BLD_TOWN_HALL, "Civic Hall", 60, 60, 12, 10, 16, 0},
    {BLD_GUILD_HOUSE, "Grand Ave South Lobby", 46, 46, 18, 12, 12, 0},
    {BLD_GOLD_GUILD, "Grand Ave North Lobby", 62, 82, 16, 12, 12, 0},
    {BLD_POST_OFFICE, "Brush Row North", 14, 96, 16, 14, 10, 0},
    {BLD_BLACKSMITH, "Brush Row Court", 24, 112, 16, 12, 10, 0},
    {BLD_WEAPONS_GUILD, "Brush Hall", 16, 136, 24, 16, 14, 0},
    {BLD_POTIONS, "Stadium Concourse", 82, 132, 26, 18, 16, 0},
    {BLD_ALCHEMY_SHOP, "Event Plaza Hall", 96, 124, 18, 14, 12, 0},
    {BLD_SHADY_DEALER, "Arena Garage", 98, 92, 24, 24, 16, 0},
    {BLD_FISH_SHOP, "Warehouse Bay A", 8, 54, 28, 16, 12, 0},
    {BLD_ARMOR_SHOP, "Warehouse Bay B", 8, 74, 26, 16, 12, 0},
    {BLD_MINECO_OPS, "Utility Annex", 22, 78, 14, 10, 8, 0},
    {BLD_MINING_SUPPLIES, "Utility Garage", 22, 50, 14, 10, 8, 0},
    {BLD_ARCHERY_GUILD, "Financial Podium", 118, 42, 22, 16, 18, 0},
    {BLD_POLICE, "Riverward Core", 134, 34, 20, 18, 20, 0},
    {BLD_GLOVE_SHOP, "Congress Exchange", 110, 22, 18, 14, 14, 0},
};

static const CrisisSocket g_sockets[] = {
    {SOCK_ANCHOR_AUCTION, "Anchor: CAMPUS PLAZA", 60, 58, 3.5f, SOCK_ROLE_BUILDER, 1},
    {SOCK_RITUAL_TOWN_HALL, "Anchor: GRAND AVENUE", 60, 86, 3.5f, SOCK_ROLE_RITUALIST, 1},
    {SOCK_INTERCEPT_DOCK_ROUTE, "Anchor: STADIUM DISTRICT", 96, 132, 4.0f, SOCK_ROLE_STRIKE | SOCK_ROLE_SCOUT, 1},
    {SOCK_INTERCEPT_MINES_ROUTE, "Anchor: FINANCIAL EDGE", 120, 24, 4.0f, SOCK_ROLE_STRIKE | SOCK_ROLE_SCOUT, 1},
    {SOCK_HEAD_A_DOCKS, "Anchor: WAREHOUSE EDGE", 8, 62, 4.5f, SOCK_ROLE_STRIKE, 0},
    {SOCK_HEAD_B_MINES, "Anchor: BRUSH BLOCKS", 16, 112, 4.5f, SOCK_ROLE_STRIKE, 0},
    {SOCK_SECRET_GATE_PRESSURE, "Anchor: RIVER GATE", 142, 18, 3.0f, SOCK_ROLE_SCOUT, 1}
};

static const TownRoutePoint g_routes[] = {
    {"Campus Plaza", 60, 58},
    {"Grand Avenue", 60, 86},
    {"Stadium District", 96, 132},
    {"Brush Blocks", 20, 114},
    {"Warehouse Edge", 8, 62},
    {"Financial Edge", 120, 24}
};

static TownBuilding g_buildings_scaled[sizeof(g_buildings) / sizeof(g_buildings[0])];
static CrisisSocket g_sockets_scaled[sizeof(g_sockets) / sizeof(g_sockets[0])];
static TownRoutePoint g_routes_scaled[sizeof(g_routes) / sizeof(g_routes[0])];
static int g_scaled_init = 0;

static void town_map_init_scaled() {
    if (g_scaled_init) return;
    g_scaled_init = 1;

    for (size_t i = 0; i < sizeof(g_buildings_scaled) / sizeof(g_buildings_scaled[0]); i++) {
        g_buildings_scaled[i] = g_buildings[i];
        g_buildings_scaled[i].x *= TOWN_WORLD_SCALE;
        g_buildings_scaled[i].z *= TOWN_WORLD_SCALE;
        g_buildings_scaled[i].w *= TOWN_WORLD_SCALE;
        g_buildings_scaled[i].d *= TOWN_WORLD_SCALE;
        g_buildings_scaled[i].h *= TOWN_WORLD_SCALE;
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
    if (count) *count = sizeof(g_buildings_scaled) / sizeof(g_buildings_scaled[0]);
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
