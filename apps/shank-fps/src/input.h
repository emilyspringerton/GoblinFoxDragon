#ifndef INPUT_H
#define INPUT_H

#include "shank.h"

// Initialize Input (Hide cursor, capture mouse)
void input_init(GameContext *game);

// Process Frame Input
void input_update(GameContext *game, float delta_time);

#endif
