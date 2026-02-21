#ifndef SHANKPIT_NET_SIM_H
#define SHANKPIT_NET_SIM_H

#include "protocol.h"
#include "physics.h"
#include "shared_movement.h"

#define SHANKPIT_NET_FIXED_DT 0.016f

void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time);

static inline void shankpit_apply_usercmd_inputs(PlayerState *p, const UserCmd *cmd) {
    if (!p || !cmd) return;

    // Net movement contract:
    // 1) Raw command carries intent axes + control yaw/pitch.
    // 2) Axes are clamped/normalized once here before simulation.
    // 3) Client prediction/replay and server auth must both call this path.

    if (isfinite(cmd->yaw)) p->yaw = norm_yaw_deg(cmd->yaw);
    if (isfinite(cmd->pitch)) p->pitch = clamp_pitch_deg(cmd->pitch);

    p->in_fwd = cmd->fwd;
    p->in_strafe = cmd->str;

    float move_len = sqrtf(p->in_fwd * p->in_fwd + p->in_strafe * p->in_strafe);
    if (move_len > 1.0f) {
        p->in_fwd /= move_len;
        p->in_strafe /= move_len;
    }

    p->in_jump = (cmd->buttons & BTN_JUMP) != 0;
    p->in_shoot = (cmd->buttons & BTN_ATTACK) != 0;
    p->crouching = (cmd->buttons & BTN_CROUCH) != 0;
    p->in_reload = (cmd->buttons & BTN_RELOAD) != 0;
    p->in_use = (cmd->buttons & BTN_USE) != 0;
    p->in_ability = (cmd->buttons & BTN_ABILITY_1) != 0;

    if (cmd->weapon_idx >= 0 && cmd->weapon_idx < MAX_WEAPONS) {
        p->current_weapon = cmd->weapon_idx;
    }
}

static inline void shankpit_simulate_movement_tick(PlayerState *p, unsigned int now_ms) {
    if (!p) return;

    // Net movement contract:
    // - Intent -> world-space wish conversion is shared (shankpit_move_wish_from_intent).
    // - Simulation order and fixed dt (SHANKPIT_NET_FIXED_DT) must stay identical for
    //   server authority and client prediction/replay.
    // - Reconciliation should only correct transport drift, not hide sim mismatches.

    MoveIntent move_intent = {
        .forward = p->in_fwd,
        .strafe = p->in_vehicle ? 0.0f : p->in_strafe,
        .control_yaw_deg = p->yaw,
        .wants_jump = p->in_jump,
        .wants_sprint = 0
    };
    MoveWish move_wish = shankpit_move_wish_from_intent(move_intent);

    float max_spd = p->in_vehicle ? BUGGY_MAX_SPEED : MAX_SPEED;
    float acc = p->in_vehicle ? BUGGY_ACCEL : ACCEL;
    float wish_speed = move_wish.magnitude * max_spd;
    accelerate(p, move_wish.dir_x, move_wish.dir_z, wish_speed, acc);

    float g = p->in_vehicle ? BUGGY_GRAVITY : (p->in_jump ? GRAVITY_FLOAT : GRAVITY_DROP);
    p->vy -= g;
    if (p->in_jump && p->on_ground) {
        p->y += 0.1f;
        p->vy += JUMP_FORCE;
    }

    update_entity(p, SHANKPIT_NET_FIXED_DT, NULL, now_ms);
}

#endif
