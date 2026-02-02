#ifndef SERVER_STATE_H
#define SERVER_STATE_H

#ifdef _WIN32
#include <winsock2.h>
#else
#include <netinet/in.h>
#endif

#include <stdio.h>
#include <string.h>

#include "../../../packages/simulation/local_game.h"

static inline void server_disconnect(int i, unsigned int *client_last_seq) {
    printf("[SERVER] DISCONNECT slot=%d\n", i);
    memset(&local_state.players[i], 0, sizeof(PlayerState));
    memset(&local_state.clients[i], 0, sizeof(struct sockaddr_in));
    local_state.client_meta[i].active = 0;
    local_state.client_meta[i].last_heard_ms = 0;
    client_last_seq[i] = 0;
}

#endif
