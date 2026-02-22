#include "town_map.h"

static const TownBuilding g_buildings[] = {
    {BLD_AUCTION_HOUSE, "Auction House", 50, 52, 34, 8, 8, 0},
    {BLD_TOWN_HALL, "Town Hall", 66, 61, 10, 8, 10, 0},
    {BLD_GUILD_HOUSE, "Guild House", 36, 58, 10, 8, 8, 0},
    {BLD_GOLD_GUILD, "Gold Guild", 28, 56, 10, 8, 8, 0},
    {BLD_POST_OFFICE, "Post Office", 22, 44, 12, 8, 7, 0},
    {BLD_BLACKSMITH, "Blacksmith", 28, 68, 8, 8, 8, 0},
    {BLD_WEAPONS_GUILD, "Weapons Guild", 33, 82, 14, 10, 8, 0},
    {BLD_POTIONS, "Potions", 58, 66, 8, 8, 7, 0},
    {BLD_ALCHEMY_SHOP, "Alchemy Shop", 72, 66, 10, 8, 7, 0},
    {BLD_SHADY_DEALER, "Shady Dealer", 82, 60, 10, 8, 8, 0},
    {BLD_FISH_SHOP, "Fish Shop", 52, 86, 10, 8, 7, 0},
    {BLD_ARMOR_SHOP, "Armor Shop", 72, 74, 10, 8, 8, 0},
    {BLD_MINECO_OPS, "Mineco Ops", 45, 26, 12, 8, 8, 0},
    {BLD_MINING_SUPPLIES, "Mining Supplies", 60, 26, 12, 8, 8, 0},
    {BLD_ARCHERY_GUILD, "Archery Guild", 84, 40, 12, 10, 8, 0},
    {BLD_POLICE, "Police", 66, 34, 8, 6, 7, 0},
    {BLD_GLOVE_SHOP, "Glove Shop", 80, 22, 8, 8, 7, 0},
};

static const CrisisSocket g_sockets[] = {
    {SOCK_ANCHOR_AUCTION, "Anchor: Auction", 52, 58, 3.5f, SOCK_ROLE_BUILDER, 1},
    {SOCK_RITUAL_TOWN_HALL, "Ritual: Town Hall", 66, 61, 3.5f, SOCK_ROLE_RITUALIST, 1},
    {SOCK_INTERCEPT_DOCK_ROUTE, "Intercept: Docks", 84, 86, 4.0f, SOCK_ROLE_STRIKE | SOCK_ROLE_SCOUT, 1},
    {SOCK_INTERCEPT_MINES_ROUTE, "Intercept: Mines", 88, 20, 4.0f, SOCK_ROLE_STRIKE | SOCK_ROLE_SCOUT, 1},
    {SOCK_HEAD_A_DOCKS, "Head A: Docks Edge", 90, 82, 4.5f, SOCK_ROLE_STRIKE, 0},
    {SOCK_HEAD_B_MINES, "Head B: Mines Edge", 92, 18, 4.5f, SOCK_ROLE_STRIKE, 0},
    {SOCK_SECRET_GATE_PRESSURE, "Secret Gate Pressure", 12, 20, 3.0f, SOCK_ROLE_SCOUT, 1}
};

static const TownRoutePoint g_routes[] = {
    {"Auction", 50, 52},
    {"Docks", 90, 86},
    {"Mines", 90, 20},
    {"Secret Gate", 12, 20},
    {"Town Hall", 66, 61}
};

const TownBuilding *town_map_buildings(size_t *count) { if (count) *count = sizeof(g_buildings) / sizeof(g_buildings[0]); return g_buildings; }
const CrisisSocket *town_map_sockets(size_t *count) { if (count) *count = sizeof(g_sockets) / sizeof(g_sockets[0]); return g_sockets; }
const TownRoutePoint *town_map_route_points(size_t *count) { if (count) *count = sizeof(g_routes) / sizeof(g_routes[0]); return g_routes; }
