#include <math.h>
#include "map.h" 

void map_init(GameMap *map) {
    map->wall_count = 0;
    
    // 1. The Neon Grid Floor (Huge)
    // ID 1 = Floor
    map->walls[map->wall_count++] = (Wall){1, 0, -1.0f, 0, 200.0f, 2.0f, 200.0f, 0.0f, 0.1f, 0.2f, 0.95f};

    // 2. Parkour Cubes (Random heights to jump on)
    srand(42); 
    for(int i=0; i<30; i++) {
        float x = (rand() % 60) - 30.0f;
        float z = (rand() % 60) - 30.0f;
        // Make them accessible steps
        float h = 1.0f + (rand() % 3); 
        
        map->walls[map->wall_count++] = (Wall){
            10 + i, 
            x, h/2.0f, z, 
            2.0f, h, 2.0f, 
            0.8f, 0.8f, 0.8f, 0.98f // High friction/slip
        };
        
    }

    srand(41); 
    for(int i=0; i<30; i++) {
        float x = (rand() % 60) - 30.0f;
        float z = (rand() % 60) - 30.0f;
        // Make them accessible steps
        float h = 1.0f + (rand() % 3); 
        
        map->walls[map->wall_count++] = (Wall){
            10 + i, 
            x, h/2.0f, z, 
            2.0f, h, 2.0f, 
            0.8f, 0.8f, 0.8f, 0.98f // High friction/slip
        };
    
    // 3. The "Shank" Platform (Center)
    map->walls[map->wall_count++] = (Wall){99, 0.0f, 0.5f, 0.0f, 4.0f, 1.0f, 4.0f, 1.0f, 0.8f, 0.0f, 0.96f};
}

int map_ray_cast(GameMap *map, Vec3 origin, Vec3 dir, float max_dist) {
    // Simple Ray-AABB intersection for shooting
    for(int i=0; i<map->wall_count; i++) {
        Wall *w = &map->walls[i];
        float min_x = w->x - w->sx/2; float max_x = w->x + w->sx/2;
        float min_y = w->y - w->sy/2; float max_y = w->y + w->sy/2;
        float min_z = w->z - w->sz/2; float max_z = w->z + w->sz/2;

        float tmin = (min_x - origin.x) / dir.x; 
        float tmax = (max_x - origin.x) / dir.x; 
        if (tmin > tmax) { float t=tmin; tmin=tmax; tmax=t; }
        
        float tymin = (min_y - origin.y) / dir.y; 
        float tymax = (max_y - origin.y) / dir.y; 
        if (tymin > tymax) { float t=tymin; tymin=tymax; tymax=t; }
        
        if ((tmin > tymax) || (tymin > tmax)) continue;
        if (tymin > tmin) tmin = tymin;
        if (tymax < tmax) tmax = tymax;
        
        float tzmin = (min_z - origin.z) / dir.z; 
        float tzmax = (max_z - origin.z) / dir.z; 
        if (tzmin > tzmax) { float t=tzmin; tzmin=tzmax; tzmax=t; }
        
        if ((tmin > tzmax) || (tzmin > tmax)) continue;
        if (tzmin > tmin) tmin = tzmin;
        if (tzmax < tmax) tmax = tzmax;
        
        if (tmin > 0 && tmin < max_dist) return 1; // Hit
    }
    return 0;
}

// THE FIX: Platforming Physics
void map_resolve_collision(GameMap *map, Vec3 *pos, Vec3 *vel) {
    float player_w = 0.6f;
    float player_h = 1.8f; 
    int on_ground = 0;

    for(int i=0; i<map->wall_count; i++) {
        Wall *w = &map->walls[i];
        
        float half_sx = w->sx/2 + player_w/2;
        float half_sy = w->sy/2 + player_h/2; // Height check is tricky, simplified
        float half_sz = w->sz/2 + player_w/2;
        
        float dx = pos->x - w->x;
        float dy = (pos->y + player_h/2) - w->y; // Center of player
        float dz = pos->z - w->z;
        
        float abs_dx = fabs(dx);
        float abs_dy = fabs(dy);
        float abs_dz = fabs(dz);
        
        if (abs_dx < half_sx && abs_dz < half_sz) {
            // Potential collision. Check vertical bounds carefully.
            // Player Bottom: pos->y
            // Wall Top: w->y + w->sy/2
            
            float wall_top = w->y + w->sy/2;
            float wall_bot = w->y - w->sy/2;
            
            // LANDING LOGIC
            // If we are falling, and our feet were previously above the wall...
            // Or just simplistic: if feet are near top and velocity is down
            if (vel->y <= 0 && pos->y >= (wall_top - 0.1f) && pos->y < (wall_top + 0.5f)) {
                pos->y = wall_top;
                vel->y = 0;
                on_ground = 1;
            }
            // CEILING HIT
            else if (vel->y > 0 && (pos->y + player_h) > wall_bot && pos->y < wall_bot) {
                pos->y = wall_bot - player_h;
                vel->y = 0;
            }
            // SIDE HIT (Push out closest axis)
            else if (pos->y < wall_top - 0.1f) {
                float pen_x = half_sx - abs_dx;
                float pen_z = half_sz - abs_dz;
                
                if (pen_x < pen_z) {
                    if (dx > 0) pos->x = w->x + half_sx;
                    else pos->x = w->x - half_sx;
                    vel->x = 0;
                } else {
                    if (dz > 0) pos->z = w->z + half_sz;
                    else pos->z = w->z - half_sz;
                    vel->z = 0;
                }
            }
        }
    }
    
    // Floor Safety
    if (pos->y < -20.0f) { pos->y = 10.0f; pos->x = 0; pos->z = 0; vel->y = 0; }
}

