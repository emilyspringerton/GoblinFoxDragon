#ifndef BOT_AI_H
#define BOT_AI_H

#include "../common/protocol.h"
#include <math.h>

// This is the "Policy" function
// Inputs: State
// Outputs: Controls (Forward, Strafe, Yaw, Pitch, Shoot)
void bot_think(PlayerState *bot, ServerState *world) {
    // 1. Find Closest Target
    PlayerState *target = NULL;
    float best_dist = 9999.0f;

    for (int i=0; i<MAX_CLIENTS; i++) {
        if (world->players[i].active && &world->players[i] != bot && !world->players[i].is_bot) {
            float dx = world->players[i].x - bot->x;
            float dz = world->players[i].z - bot->z;
            float d = sqrtf(dx*dx + dz*dz);
            if (d < best_dist) {
                best_dist = d;
                target = &world->players[i];
            }
        }
    }

    if (!target) {
        bot->in_fwd = 0; bot->in_shoot = 0;
        return;
    }

    // 2. Aim at Target (Math)
    float dx = target->x - bot->x;
    float dy = (target->y + 2.0f) - (bot->y + 4.0f); // Eye to Body
    float dz = target->z - bot->z;
    float dist_h = sqrtf(dx*dx + dz*dz);

    // Calculate Yaw (Atan2 returns -PI to PI, convert to Degrees)
    // Note: Physics expects inverted Yaw for visuals, but pure math is standard
    // Visual Yaw = -Math Yaw
    float target_yaw = atan2f(dx, -dz) * (180.0f / M_PI); // -Z is North
    float target_pitch = atan2f(dy, dist_h) * (180.0f / M_PI);

    // Smooth Aim (Slew)
    float diff = target_yaw - bot->yaw;
    while(diff > 180) diff -= 360;
    while(diff < -180) diff += 360;
    bot->yaw += diff * 0.1f; // Turn speed
    bot->pitch += (target_pitch - bot->pitch) * 0.1f;

    // 3. Movement Logic (The "Brain")
    if (dist_h > 15.0f) bot->in_fwd = 1.0f;        // Too far? Run.
    else if (dist_h < 5.0f) bot->in_fwd = -1.0f;   // Too close? Back up.
    else bot->in_fwd = 0.0f;                       // Sweet spot.

    // Random Strafe
    if ((world->server_tick / 60) % 2 == 0) bot->in_strafe = 1.0f;
    else bot->in_strafe = -1.0f;

    // 4. Trigger Discipline
    // Shoot if looking roughly at target
    if (fabs(diff) < 10.0f) {
        bot->in_shoot = 1;
    } else {
        bot->in_shoot = 0;
    }
}
#endif
