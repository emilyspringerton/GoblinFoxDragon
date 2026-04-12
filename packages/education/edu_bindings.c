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
            world->gate_open = 1;
            world->quest_open_the_gate = 1;
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
}
