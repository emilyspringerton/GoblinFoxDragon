#ifndef SHANKPIT_NET_SIM_H
#define SHANKPIT_NET_SIM_H

#include "protocol.h"
#include "physics.h"
#include "shared_movement.h"

#define SHANKPIT_NET_FIXED_DT 0.016f

extern ServerState local_state;
void update_entity(PlayerState *p, float dt, void *server_context, unsigned int cmd_time);

static inline int shankpit_find_player_helicopter(int player_id) {
    for (int i = 0; i < MAX_HELICOPTERS; i++) {
        HelicopterState *h = &local_state.helicopters[i];
        if (!h->active) continue;
        if (h->occupant_player_id == player_id) return i;
    }
    return -1;
}

static inline void shankpit_helicopter_simulate(HelicopterState *h, PlayerState *pilot, float dt) {
    if (!h || !h->active) return;
    if (pilot) {
        h->input_forward = pilot->in_fwd;
        h->input_strafe = pilot->in_strafe;
        h->input_yaw = pilot->in_strafe;
        h->input_collective = (pilot->in_jump ? 1.0f : 0.0f) - (pilot->crouching ? 1.0f : 0.0f);
    } else {
        h->input_forward = 0.0f;
        h->input_strafe = 0.0f;
        h->input_yaw = 0.0f;
        h->input_collective = 0.0f;
    }

    h->yaw = norm_yaw_deg(h->yaw + h->input_yaw * g_heli_tuning.yaw_rate_deg);

    float yaw_r = -h->yaw * (SHANKPIT_PI / 180.0f);
    float fwd_x = sinf(yaw_r), fwd_z = -cosf(yaw_r);
    float right_x = cosf(yaw_r), right_z = sinf(yaw_r);

    h->vx += (fwd_x * h->input_forward * g_heli_tuning.forward_accel) +
             (right_x * h->input_strafe * g_heli_tuning.strafe_accel);
    h->vz += (fwd_z * h->input_forward * g_heli_tuning.forward_accel) +
             (right_z * h->input_strafe * g_heli_tuning.strafe_accel);

    h->vy += g_heli_tuning.hover_lift - GRAVITY_DROP;
    if (h->input_collective > 0.05f) h->vy += g_heli_tuning.ascend_accel * h->input_collective;
    if (h->input_collective < -0.05f) h->vy += g_heli_tuning.descend_accel * h->input_collective;
    if (fabsf(h->input_collective) < 0.05f) h->vy *= (1.0f - g_heli_tuning.hover_assist);

    h->vx *= (1.0f - g_heli_tuning.drag);
    h->vz *= (1.0f - g_heli_tuning.drag);
    h->vy *= (1.0f - g_heli_tuning.vertical_damping);

    float hspeed = sqrtf(h->vx * h->vx + h->vz * h->vz);
    if (hspeed > g_heli_tuning.max_hspeed && hspeed > 0.001f) {
        float scale = g_heli_tuning.max_hspeed / hspeed;
        h->vx *= scale;
        h->vz *= scale;
    }
    if (h->vy > g_heli_tuning.max_vspeed_up) h->vy = g_heli_tuning.max_vspeed_up;
    if (h->vy < -g_heli_tuning.max_vspeed_down) h->vy = -g_heli_tuning.max_vspeed_down;

    float next_x = h->x + h->vx;
    float next_y = h->y + h->vy;
    float next_z = h->z + h->vz;
    float hit_x = 0.0f, hit_y = 0.0f, hit_z = 0.0f, nx = 0.0f, ny = 0.0f, nz = 0.0f;
    phys_set_scene(h->scene_id);
    if (trace_map(h->x, h->y, h->z, next_x, next_y, next_z, &hit_x, &hit_y, &hit_z, &nx, &ny, &nz)) {
        h->x = hit_x; h->y = hit_y; h->z = hit_z;
        if (fabsf(nx) > 0.01f) h->vx *= -0.22f;
        if (fabsf(ny) > 0.01f) h->vy *= -0.18f;
        if (fabsf(nz) > 0.01f) h->vz *= -0.22f;
    } else {
        h->x = next_x; h->y = next_y; h->z = next_z;
    }

    float ground_y = 0.0f;
    for (int i = 0; i < map_count; i++) {
        Box b = map_geo[i];
        if (h->x >= b.x - b.w * 0.5f && h->x <= b.x + b.w * 0.5f &&
            h->z >= b.z - b.d * 0.5f && h->z <= b.z + b.d * 0.5f) {
            float top = b.y + b.h * 0.5f;
            if (top > ground_y) ground_y = top;
        }
    }
    float skid_y = ground_y + g_heli_tuning.collider_height * 0.9f;
    if (h->y <= skid_y) {
        if (h->vy < -0.65f) h->health -= 4;
        h->y = skid_y;
        h->vy = 0.0f;
        h->grounded = 1;
    } else {
        h->grounded = 0;
    }

    h->pitch_visual = shankpit_clampf(-h->input_forward * g_heli_tuning.pitch_visual_max,
                                      -g_heli_tuning.pitch_visual_max, g_heli_tuning.pitch_visual_max);
    h->roll_visual = shankpit_clampf(-h->input_strafe * g_heli_tuning.roll_visual_max,
                                     -g_heli_tuning.roll_visual_max, g_heli_tuning.roll_visual_max);

    float desired_rotor = g_heli_tuning.rotor_spin_idle +
        (pilot ? (g_heli_tuning.rotor_spin_max - g_heli_tuning.rotor_spin_idle) : 4.0f);
    h->rotor_speed += (desired_rotor - h->rotor_speed) * 0.06f;
    h->rotor_angle = norm_yaw_deg(h->rotor_angle + h->rotor_speed * dt * 60.0f);
    h->throttle = (h->input_collective + 1.0f) * 0.5f;
}

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

    if (p->in_vehicle && p->vehicle_type == VEH_HELICOPTER) {
        int heli_idx = shankpit_find_player_helicopter(p->id);
        if (heli_idx >= 0) {
            HelicopterState *h = &local_state.helicopters[heli_idx];
            shankpit_helicopter_simulate(h, p, SHANKPIT_NET_FIXED_DT);
            p->x = h->x;
            p->y = h->y - 1.4f;
            p->z = h->z;
            p->vx = h->vx;
            p->vy = h->vy;
            p->vz = h->vz;
            p->yaw = h->yaw;
            p->on_ground = h->grounded;
            p->in_shoot = 0;
            p->in_reload = 0;
            p->in_ability = 0;
            return;
        }
        p->in_vehicle = 0;
        p->vehicle_type = VEH_NONE;
    }

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
