#include <stdio.h>
#include <string.h>
#include <math.h>
#include <netinet/in.h>

#include "../../packages/common/net_sim.h"
#include "../../packages/simulation/local_game.h"

#define PARITY_TICKS 240
#define PARITY_DT_MS 16u
#define PARITY_PASS_EPS 0.0001f

typedef struct {
    float max_corr;
    float avg_corr;
    float max_vel_delta;
    unsigned int grounded_mismatch;
    unsigned int jump_mismatch;
} ScenarioMetrics;

static void init_player(PlayerState *p) {
    memset(p, 0, sizeof(*p));
    p->active = 1;
    p->id = 1;
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

static float vel_delta3(const PlayerState *a, const PlayerState *b) {
    float dx = a->vx - b->vx;
    float dy = a->vy - b->vy;
    float dz = a->vz - b->vz;
    return sqrtf(dx * dx + dy * dy + dz * dz);
}

static void sim_cmd(PlayerState *p, UserCmd *cmd, unsigned int now_ms) {
    shankpit_apply_usercmd_inputs(p, cmd);
    shankpit_simulate_movement_tick(p, now_ms);
}

static void setup_default(PlayerState *server, PlayerState *client) {
    (void)server;
    (void)client;
}

static void setup_wall_contact(PlayerState *server, PlayerState *client) {
    server->x = 0.0f;
    server->z = 6.0f;
    client->x = server->x;
    client->z = server->z;
}

typedef void (*FillCmdsFn)(UserCmd *cmds, int count);
typedef void (*SetupFn)(PlayerState *server, PlayerState *client);

typedef struct {
    const char *name;
    FillCmdsFn fill;
    SetupFn setup;
} Scenario;

static void fill_straight_w_yaw0(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f; }
}

static void fill_straight_w_yaw90(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = 90.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f; }
}

static void fill_strafe_d(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 0.0f; cmds[i].str = 1.0f; }
}

static void fill_strafe_a(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 0.0f; cmds[i].str = -1.0f; }
}

static void fill_diag_wd(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 1.0f; cmds[i].str = 1.0f; }
}

static void fill_turn_while_moving(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = (float)i * 2.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f; }
}

static void fill_move_jump_repeat(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) {
        cmds[i].yaw = 0.0f;
        cmds[i].fwd = 1.0f;
        cmds[i].str = 0.0f;
        cmds[i].buttons = ((i % 20) == 0) ? BTN_JUMP : 0;
    }
}

static void fill_high_speed_stress(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) {
        cmds[i].yaw = (float)((i * 7) % 360);
        cmds[i].fwd = (i & 1) ? 1.0f : 0.94f;
        cmds[i].str = (i % 3 == 0) ? 0.85f : -0.85f;
        cmds[i].buttons = ((i % 31) == 0) ? BTN_JUMP : 0;
    }
}

static void fill_wall_contact(UserCmd *cmds, int count) {
    for (int i = 0; i < count; i++) { cmds[i].yaw = 0.0f; cmds[i].fwd = 1.0f; cmds[i].str = 0.0f; }
}

static int run_sequence(const Scenario *scenario, UserCmd *cmds, int count) {
    PlayerState server, client;
    init_player(&server);
    init_player(&client);
    if (scenario->setup) scenario->setup(&server, &client);

    ScenarioMetrics m;
    memset(&m, 0, sizeof(m));

    for (int i = 0; i < count; i++) {
        unsigned int now_ms = (unsigned int)(i * PARITY_DT_MS);

        sim_cmd(&server, &cmds[i], now_ms);
        sim_cmd(&client, &cmds[i], now_ms);

        float corr = dist3(&server, &client);
        float vel_delta = vel_delta3(&server, &client);
        if (corr > m.max_corr) m.max_corr = corr;
        if (vel_delta > m.max_vel_delta) m.max_vel_delta = vel_delta;
        m.avg_corr += corr;
        if (server.on_ground != client.on_ground) m.grounded_mismatch++;
        if (server.in_jump != client.in_jump) m.jump_mismatch++;
    }
    m.avg_corr /= (float)count;

    printf("[PARITY] %-20s max=%.6f avg=%.6f vel=%.6f gmis=%u jmis=%u\n",
           scenario->name,
           m.max_corr,
           m.avg_corr,
           m.max_vel_delta,
           m.grounded_mismatch,
           m.jump_mismatch);
    return (m.max_corr < PARITY_PASS_EPS && m.max_vel_delta < PARITY_PASS_EPS) ? 0 : 1;
}

int main(void) {
    memset(&local_state, 0, sizeof(local_state));
    phys_set_scene(SCENE_GARAGE_OSAKA);

    int failed = 0;
    UserCmd cmds[PARITY_TICKS];
    memset(cmds, 0, sizeof(cmds));

    const Scenario scenarios[] = {
        {"straight_w_yaw0", fill_straight_w_yaw0, setup_default},
        {"straight_w_yaw90", fill_straight_w_yaw90, setup_default},
        {"strafe_d", fill_strafe_d, setup_default},
        {"strafe_a", fill_strafe_a, setup_default},
        {"diag_wd", fill_diag_wd, setup_default},
        {"turn_while_moving", fill_turn_while_moving, setup_default},
        {"move_jump_repeat", fill_move_jump_repeat, setup_default},
        {"high_speed_stress", fill_high_speed_stress, setup_default},
        {"wall_contact", fill_wall_contact, setup_wall_contact}
    };

    const int scenario_count = (int)(sizeof(scenarios) / sizeof(scenarios[0]));
    for (int s = 0; s < scenario_count; s++) {
        memset(cmds, 0, sizeof(cmds));
        for (int i = 0; i < PARITY_TICKS; i++) {
            cmds[i].sequence = (unsigned int)(i + 1);
            cmds[i].timestamp = (unsigned int)(i * PARITY_DT_MS);
        }
        scenarios[s].fill(cmds, PARITY_TICKS);
        failed |= run_sequence(&scenarios[s], cmds, PARITY_TICKS);
    }

    printf("net parity test %s\n", failed ? "FAILED" : "PASSED");
    return failed ? 1 : 0;
}
