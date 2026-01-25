#include "input.h"
#include <math.h>
#include "../../../packages/protocol/protocol.h" // Needed for WPN constants

#define MOUSE_SENSITIVITY 0.1f
#define MOVE_SPEED 10.0f
#define STRAFE_SPEED 8.0f
#define FRICTION 0.9f

void input_init(GameContext *game) {
    SDL_SetRelativeMouseMode(SDL_TRUE);
}

void input_update(GameContext *game, float delta_time) {
    SDL_Event event;
    Player *p = &game->local_player;
    
    while (SDL_PollEvent(&event)) {
        if (event.type == SDL_QUIT) game->running = false;
        if (event.type == SDL_KEYDOWN) {
            if (event.key.keysym.sym == SDLK_ESCAPE) game->running = false;
            // FIXED: WPN_MELEE -> WPN_KNIFE
            if (event.key.keysym.sym == SDLK_1) p->current_weapon = WPN_KNIFE;
            if (event.key.keysym.sym == SDLK_2) p->current_weapon = WPN_MAGNUM;
            if (event.key.keysym.sym == SDLK_3) p->current_weapon = WPN_AR;
            if (event.key.keysym.sym == SDLK_4) p->current_weapon = WPN_SHOTGUN;
            if (event.key.keysym.sym == SDLK_5) p->current_weapon = WPN_SNIPER;
        }
        if (event.type == SDL_MOUSEMOTION) {
            p->yaw += event.motion.xrel * MOUSE_SENSITIVITY;
            p->pitch -= event.motion.yrel * MOUSE_SENSITIVITY;
            if (p->pitch > 89.0f) p->pitch = 89.0f;
            if (p->pitch < -89.0f) p->pitch = -89.0f;
        }
    }

    const Uint8 *state = SDL_GetKeyboardState(NULL);
    float rad = p->yaw * (M_PI / 180.0f);
    float cos_yaw = cosf(rad);
    float sin_yaw = sinf(rad);

    float fwd_speed = 0.0f;
    float side_speed = 0.0f;

    if (state[SDL_SCANCODE_W]) fwd_speed += MOVE_SPEED;
    if (state[SDL_SCANCODE_S]) fwd_speed -= MOVE_SPEED;
    if (state[SDL_SCANCODE_D]) side_speed += STRAFE_SPEED;
    if (state[SDL_SCANCODE_A]) side_speed -= STRAFE_SPEED;

    p->vel.x += (sin_yaw * fwd_speed + cos_yaw * side_speed) * delta_time;
    p->vel.z -= (cos_yaw * fwd_speed - sin_yaw * side_speed) * delta_time;
    p->pos.x += p->vel.x * delta_time;
    p->pos.z += p->vel.z * delta_time;
    p->vel.x *= FRICTION;
    p->vel.z *= FRICTION;
}
