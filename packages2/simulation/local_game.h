#ifndef LOCAL_GAME_H
#define LOCAL_GAME_H

#include "../common/protocol.h"

#include <math.h>
#include <string.h>

static ServerState local_state;

static inline void local_init_match(int bot_count, int mode) {
    memset(&local_state, 0, sizeof(local_state));
    local_state.game_mode = mode;

    local_state.players[0].active = 1;
    local_state.players[0].health = 100;
    local_state.players[0].shield = 100;
    local_state.players[0].current_weapon = 1;

    for (int i = 1; i < MAX_CLIENTS && i <= bot_count; i++) {
        local_state.players[i].active = 1;
        local_state.players[i].is_bot = 1;
        local_state.players[i].health = 100;
        local_state.players[i].shield = 100;
    }
}

static inline void local_update(float forward, float strafe, float yaw, float pitch,
                                int shoot, int weapon_request, int jump, int crouch,
                                int reload, const void *net_packets, int net_packet_count) {
    (void)shoot;
    (void)weapon_request;
    (void)jump;
    (void)crouch;
    (void)reload;
    (void)net_packets;
    (void)net_packet_count;

    PlayerState *player = &local_state.players[0];
    float speed = 0.25f;
    float forward_dx = cosf(yaw) * forward;
    float forward_dz = sinf(yaw) * forward;
    float strafe_dx = -sinf(yaw) * strafe;
    float strafe_dz = cosf(yaw) * strafe;

    player->x += (forward_dx + strafe_dx) * speed;
    player->z += (forward_dz + strafe_dz) * speed;
    player->yaw = yaw;
    player->pitch = pitch;
    player->in_fwd = forward;
    player->in_strafe = strafe;
}

#endif
