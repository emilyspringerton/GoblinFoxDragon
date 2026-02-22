#ifndef TOWN_MAP_H
#define TOWN_MAP_H
#include <stddef.h>
typedef enum { BLD_AUCTION_HOUSE = 0, BLD_TOWN_HALL, BLD_GUILD_HOUSE, BLD_GOLD_GUILD, BLD_POST_OFFICE, BLD_BLACKSMITH, BLD_WEAPONS_GUILD, BLD_POTIONS, BLD_ALCHEMY_SHOP, BLD_SHADY_DEALER, BLD_FISH_SHOP, BLD_ARMOR_SHOP, BLD_MINECO_OPS, BLD_MINING_SUPPLIES, BLD_ARCHERY_GUILD, BLD_POLICE, BLD_GLOVE_SHOP, BLD_MISC_COUNT } TownBuildingId;
typedef struct { TownBuildingId id; const char *label; float x, z; float w, d; float h; unsigned int tags; } TownBuilding;
typedef enum { SOCK_ANCHOR_AUCTION = 0, SOCK_RITUAL_TOWN_HALL, SOCK_INTERCEPT_DOCK_ROUTE, SOCK_INTERCEPT_MINES_ROUTE, SOCK_HEAD_A_DOCKS, SOCK_HEAD_B_MINES, SOCK_SECRET_GATE_PRESSURE, SOCK_COUNT } CrisisSocketId;
typedef enum { SOCK_ROLE_STRIKE = 1 << 0, SOCK_ROLE_BUILDER = 1 << 1, SOCK_ROLE_RITUALIST = 1 << 2, SOCK_ROLE_SCOUT = 1 << 3 } SocketRoleFlags;
typedef struct { CrisisSocketId id; const char *label; float x, z; float radius; unsigned int role_flags; int counts_for_final_gate; } CrisisSocket;
typedef struct { const char *label; float x, z; } TownRoutePoint;
const TownBuilding *town_map_buildings(size_t *count);
const CrisisSocket *town_map_sockets(size_t *count);
const TownRoutePoint *town_map_route_points(size_t *count);
#endif
