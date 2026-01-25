#ifndef NET_H
#define NET_H

#include "shank.h"
#include "../../../packages/protocol/protocol.h"

void net_init(GameContext *game, const char *ip);
void net_send_input(GameContext *game);
void net_receive_state(GameContext *game);

#endif
