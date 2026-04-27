#include <string.h>
#include <stdio.h>
#include "edu_bindings.h"

typedef struct {
    const char *name;
    int id;
    int argc;
} BuiltinDef;

static const BuiltinDef G_BUILTINS[] = {
    {"set_crate_speed", EDU_BUILTIN_SET_CRATE_SPEED, 1},
    {"move_crate", EDU_BUILTIN_MOVE_CRATE, 0},
    {"stop_crate", EDU_BUILTIN_STOP_CRATE, 0},
    {"open_gate", EDU_BUILTIN_OPEN_GATE, 0},
    {"close_gate", EDU_BUILTIN_CLOSE_GATE, 0},
    {"set_switch", EDU_BUILTIN_SET_SWITCH, 1},
    {"is_switch_on", EDU_BUILTIN_IS_SWITCH_ON, 0},
    {"query_grounded", EDU_BUILTIN_QUERY_GROUNDED, 1},
    {"mark_quest_complete", EDU_BUILTIN_MARK_QUEST_COMPLETE, 1},
    {"get_quest_state", EDU_BUILTIN_GET_QUEST_STATE, 1},
    {"print", EDU_BUILTIN_PRINT, 1},
    {"show_message", EDU_BUILTIN_SHOW_MESSAGE, 1},
    {"scan_gate", EDU_BUILTIN_SCAN_GATE, 0},
    {"scan_portal", EDU_BUILTIN_SCAN_PORTAL, 0},
    {"scan_enemy_count", EDU_BUILTIN_SCAN_ENEMY_COUNT, 0},
    {"raise_bridge", EDU_BUILTIN_RAISE_BRIDGE, 0},
    {"stabilize_portal", EDU_BUILTIN_STABILIZE_PORTAL, 1},
    {"open_portal", EDU_BUILTIN_OPEN_PORTAL, 0},
    {"mark_enemy", EDU_BUILTIN_MARK_ENEMY, 1},
    {"slow_enemy", EDU_BUILTIN_SLOW_ENEMY, 1},
};

int edu_binding_lookup(const char *name, int name_len, int *out_id, int *out_argc) {
    for (int i = 0; i < (int)(sizeof(G_BUILTINS) / sizeof(G_BUILTINS[0])); i++) {
        if ((int)strlen(G_BUILTINS[i].name) == name_len && strncmp(name, G_BUILTINS[i].name, name_len) == 0) {
            *out_id = G_BUILTINS[i].id;
            *out_argc = G_BUILTINS[i].argc;
            return 1;
        }
    }
    return 0;
}

static const char *safe_string(const char *const *strings, int count, int idx) {
    if (idx < 0 || idx >= count || !strings[idx]) return "";
    return strings[idx];
}

int edu_binding_call(EduWorldState *world, int builtin_id, const int *args, int argc,
                     const char *const *strings, int string_count, int *out_ret,
                     char *debug_out, int debug_cap) {
    (void)argc;
    *out_ret = 0;
#define EDU_FORBIDDEN(msg) do { snprintf(debug_out, (size_t)debug_cap, "forbidden: %s", msg); return 0; } while (0)
    switch (builtin_id) {
        case EDU_BUILTIN_SET_CRATE_SPEED:
            world->crate_speed = args[0];
            snprintf(debug_out, (size_t)debug_cap, "set_crate_speed(%d)", args[0]);
            return 1;
        case EDU_BUILTIN_MOVE_CRATE:
            world->crate_x += world->crate_speed;
            if (world->crate_x > 20) world->quest_make_it_move = 1;
            snprintf(debug_out, (size_t)debug_cap, "move_crate -> x=%d", world->crate_x);
            return 1;
        case EDU_BUILTIN_STOP_CRATE:
            world->crate_speed = 0;
            snprintf(debug_out, (size_t)debug_cap, "stop_crate");
            return 1;
        case EDU_BUILTIN_OPEN_GATE:
            if (!world->can_open_gate) EDU_FORBIDDEN("open_gate capability locked");
            world->gate_open = 1;
            world->quest_open_the_gate = 1;
            printf("[REALITY] gate opened\n");
            snprintf(debug_out, (size_t)debug_cap, "open_gate");
            return 1;
        case EDU_BUILTIN_CLOSE_GATE:
            world->gate_open = 0;
            snprintf(debug_out, (size_t)debug_cap, "close_gate");
            return 1;
        case EDU_BUILTIN_SET_SWITCH:
            world->switch_on = args[0] ? 1 : 0;
            snprintf(debug_out, (size_t)debug_cap, "set_switch(%d)", world->switch_on);
            return 1;
        case EDU_BUILTIN_IS_SWITCH_ON:
            *out_ret = world->switch_on ? 1 : 0;
            snprintf(debug_out, (size_t)debug_cap, "is_switch_on -> %d", *out_ret);
            return 1;
        case EDU_BUILTIN_QUERY_GROUNDED: {
            const char *id = safe_string(strings, string_count, args[0]);
            *out_ret = (strcmp(id, "test_platform") == 0) ? world->grounded_test_platform : 0;
            snprintf(debug_out, (size_t)debug_cap, "query_grounded(%s) -> %d", id, *out_ret);
            return 1;
        }
        case EDU_BUILTIN_MARK_QUEST_COMPLETE: {
            const char *id = safe_string(strings, string_count, args[0]);
            if (strcmp(id, "STOP_THE_FALL") == 0) world->quest_stop_the_fall = 1;
            if (strcmp(id, "OPEN_THE_GATE") == 0) world->quest_open_the_gate = 1;
            if (strcmp(id, "MAKE_IT_MOVE") == 0) world->quest_make_it_move = 1;
            snprintf(debug_out, (size_t)debug_cap, "mark_quest_complete(%s)", id);
            return 1;
        }
        case EDU_BUILTIN_GET_QUEST_STATE: {
            const char *id = safe_string(strings, string_count, args[0]);
            if (strcmp(id, "STOP_THE_FALL") == 0) *out_ret = world->quest_stop_the_fall;
            else if (strcmp(id, "OPEN_THE_GATE") == 0) *out_ret = world->quest_open_the_gate;
            else if (strcmp(id, "MAKE_IT_MOVE") == 0) *out_ret = world->quest_make_it_move;
            snprintf(debug_out, (size_t)debug_cap, "get_quest_state(%s) -> %d", id, *out_ret);
            return 1;
        }
        case EDU_BUILTIN_SCAN_GATE:
            *out_ret = world->gate_open ? 1 : 0;
            snprintf(debug_out, (size_t)debug_cap, "scan_gate -> %d", *out_ret);
            return 1;
        case EDU_BUILTIN_SCAN_PORTAL:
            *out_ret = world->portal_stability;
            snprintf(debug_out, (size_t)debug_cap, "scan_portal -> %d", *out_ret);
            return 1;
        case EDU_BUILTIN_SCAN_ENEMY_COUNT:
            *out_ret = world->enemy_count;
            snprintf(debug_out, (size_t)debug_cap, "scan_enemy_count -> %d", *out_ret);
            return 1;
        case EDU_BUILTIN_RAISE_BRIDGE:
            if (!world->can_raise_bridge) EDU_FORBIDDEN("raise_bridge capability locked");
            world->bridge_raised = 1;
            world->bridge_angle = 35;
            printf("[REALITY] bridge raised\n");
            snprintf(debug_out, (size_t)debug_cap, "raise_bridge");
            return 1;
        case EDU_BUILTIN_STABILIZE_PORTAL: {
            if (!world->can_modify_portal) EDU_FORBIDDEN("stabilize_portal capability locked");
            int amount = args[0];
            if (amount < 0) amount = 0;
            world->portal_stability += amount;
            if (world->portal_stability > 150) world->portal_stability = 150;
            printf("[REALITY] portal stability=%d\n", world->portal_stability);
            snprintf(debug_out, (size_t)debug_cap, "stabilize_portal(%d) -> %d", args[0], world->portal_stability);
            return 1;
        }
        case EDU_BUILTIN_OPEN_PORTAL:
            if (!world->can_modify_portal) EDU_FORBIDDEN("open_portal capability locked");
            if (world->portal_stability < 100) EDU_FORBIDDEN("portal stability below threshold 100");
            world->portal_open = 1;
            if (world->gate_open && world->bridge_raised) world->trial_complete = 1;
            printf("[REALITY] portal opened\n");
            snprintf(debug_out, (size_t)debug_cap, "open_portal");
            return 1;
        case EDU_BUILTIN_MARK_ENEMY: {
            if (!world->can_affect_enemies) EDU_FORBIDDEN("mark_enemy capability locked");
            int idx = args[0];
            if (idx < 0 || idx >= 4 || idx >= world->enemy_count) EDU_FORBIDDEN("enemy index out of range");
            world->enemy_marked[idx] = 1;
            snprintf(debug_out, (size_t)debug_cap, "mark_enemy(%d)", idx);
            return 1;
        }
        case EDU_BUILTIN_SLOW_ENEMY: {
            if (!world->can_affect_enemies) EDU_FORBIDDEN("slow_enemy capability locked");
            int idx = args[0];
            if (idx < 0 || idx >= 4 || idx >= world->enemy_count) EDU_FORBIDDEN("enemy index out of range");
            world->enemy_slowed[idx] = 1;
            printf("[RIFT] hound slowed\n");
            snprintf(debug_out, (size_t)debug_cap, "slow_enemy(%d)", idx);
            return 1;
        }
        case EDU_BUILTIN_PRINT:
            world->last_print = args[0];
            snprintf(debug_out, (size_t)debug_cap, "print(%d)", args[0]);
            return 1;
        case EDU_BUILTIN_SHOW_MESSAGE: {
            const char *id = safe_string(strings, string_count, args[0]);
            snprintf(world->last_message, sizeof(world->last_message), "%s", id);
            snprintf(debug_out, (size_t)debug_cap, "show_message(%s)", id);
            return 1;
        }
        default:
            snprintf(debug_out, (size_t)debug_cap, "unknown builtin=%d", builtin_id);
            return 0;
    }
#undef EDU_FORBIDDEN
}
