#ifndef SHANKPIT_SHARED_MOVEMENT_H
#define SHANKPIT_SHARED_MOVEMENT_H

#include <math.h>

#ifndef SHANKPIT_PI
#define SHANKPIT_PI 3.14159265358979323846f
#endif

typedef struct MoveIntent {
    float forward;
    float strafe;
    float control_yaw_deg;
    int wants_jump;
    int wants_sprint;
} MoveIntent;

typedef struct MoveWish {
    float dir_x;
    float dir_z;
    float magnitude;
    int wants_jump;
    int wants_sprint;
} MoveWish;

typedef struct YawBasis {
    float fwd_x;
    float fwd_z;
    float right_x;
    float right_z;
} YawBasis;

static inline YawBasis shankpit_yaw_basis_from_deg(float yaw_deg) {
    float r = -yaw_deg * (SHANKPIT_PI / 180.0f);

    YawBasis b;
    b.fwd_x = sinf(r);
    b.fwd_z = -cosf(r);
    b.right_x = cosf(r);
    b.right_z = sinf(r);
    return b;
}

static inline float shankpit_clampf(float v, float lo, float hi) {
    return (v < lo) ? lo : ((v > hi) ? hi : v);
}

static inline MoveWish shankpit_move_wish_from_intent(MoveIntent in) {
    MoveWish w;
    w.dir_x = 0.0f;
    w.dir_z = 0.0f;
    w.magnitude = 0.0f;
    w.wants_jump = in.wants_jump;
    w.wants_sprint = in.wants_sprint;

    in.forward = shankpit_clampf(in.forward, -1.0f, 1.0f);
    in.strafe = shankpit_clampf(in.strafe, -1.0f, 1.0f);

    YawBasis b = shankpit_yaw_basis_from_deg(in.control_yaw_deg);

    float x = b.fwd_x * in.forward + b.right_x * in.strafe;
    float z = b.fwd_z * in.forward + b.right_z * in.strafe;

    float len2 = x * x + z * z;
    if (len2 > 1e-8f) {
        float len = sqrtf(len2);
        w.dir_x = x / len;
        w.dir_z = z / len;

        float raw_mag = sqrtf(in.forward * in.forward + in.strafe * in.strafe);
        w.magnitude = (raw_mag > 1.0f) ? 1.0f : raw_mag;
    }

    return w;
}

#endif
