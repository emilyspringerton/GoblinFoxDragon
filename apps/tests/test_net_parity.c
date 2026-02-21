#include <stdio.h>
#include <string.h>
#include <math.h>
#include <netinet/in.h>

#include "../../packages/common/net_sim.h"
#include "../../packages/simulation/local_game.h"

static void init_player(PlayerState *p) {
    memset(p, 0, sizeof(*p));
    p->active = 1;
    p->state = STATE_ALIVE;
    p->scene_id = SCENE_GARAGE_OSAKA;
    p->on_ground = 1;
    p->health = 100;
    p->shield = 100;
}

static float dist3(const PlayerState *a, const PlayerState *b) {
    float dx = a->x - b->x;
    float dy = a->y - b->y;
    float dz = a->z - b->z;
    return sqrtf(dx*dx + dy*dy + dz*dz);
}

static void sim_cmd(PlayerState *p, UserCmd *cmd, unsigned int now_ms) {
    shankpit_apply_usercmd_inputs(p, cmd);
    shankpit_simulate_movement_tick(p, now_ms);
}

static int run_sequence(const char *name, UserCmd *cmds, int count) {
    PlayerState server, client;
    init_player(&server);
    init_player(&client);

    float max_corr = 0.0f;
    for (int i = 0; i < count; i++) {
        unsigned int now_ms = (unsigned int)(i * 16);

        sim_cmd(&server, &cmds[i], now_ms);
        sim_cmd(&client, &cmds[i], now_ms);

        float corr = dist3(&server, &client);
        if (corr > max_corr) max_corr = corr;
    }

    printf("%s max_corr=%.6f\n", name, max_corr);
    return (max_corr < 0.0001f) ? 0 : 1;
}

int main(void) {
    memset(&local_state, 0, sizeof(local_state));
    phys_set_scene(SCENE_GARAGE_OSAKA);

    int failed = 0;
    const int N = 120;
    UserCmd cmds[120];
    memset(cmds, 0, sizeof(cmds));

    for (int i = 0; i < N; i++) { cmds[i].sequence = (unsigned int)(i + 1); cmds[i].yaw = 0.0f; cmds[i].fwd = 1.0f; }
    failed |= run_sequence("straight_w_yaw0", cmds, N);

    for (int i = 0; i < N; i++) { cmds[i].sequence = (unsigned int)(i + 1); cmds[i].yaw = 90.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f; }
    failed |= run_sequence("straight_w_yaw90", cmds, N);

    for (int i = 0; i < N; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 0.0f; cmds[i].str = 1.0f; }
    failed |= run_sequence("strafe_d", cmds, N);

    for (int i = 0; i < N; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 0.0f; cmds[i].str = -1.0f; }
    failed |= run_sequence("strafe_a", cmds, N);

    for (int i = 0; i < N; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 1.0f; cmds[i].str = 1.0f; }
    failed |= run_sequence("diag_wd", cmds, N);

    for (int i = 0; i < N; i++) { cmds[i].yaw = (float)i * 2.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f; }
    failed |= run_sequence("turn_while_moving", cmds, N);

    for (int i = 0; i < N; i++) {
        cmds[i].yaw = 0.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f;
        if ((i % 20) == 0) cmds[i].buttons = BTN_JUMP;
        else cmds[i].buttons = 0;
    }
    failed |= run_sequence("move_jump_repeat", cmds, N);

    printf("net parity test %s\n", failed ? "FAILED" : "PASSED");
    return failed ? 1 : 0;
}
