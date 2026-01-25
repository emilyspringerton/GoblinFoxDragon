#ifndef MAP_H
#define MAP_H
#include "protocol.h"
typedef struct { int id; float x, y, z; float sx, sy, sz; float r, g, b; float friction; } Wall;
typedef struct { Wall walls[100]; int wall_count; } GameMap;
void map_init(GameMap *map);
int map_ray_cast(GameMap *map, Vec3 origin, Vec3 dir, float max_dist);
void map_resolve_collision(GameMap *map, Vec3 *pos, Vec3 *vel);
#endif
