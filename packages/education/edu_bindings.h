#ifndef EDU_BINDINGS_H
#define EDU_BINDINGS_H

typedef struct {
    int crate_speed;
    int crate_x;
    int gate_open;
    int switch_on;
    int grounded_test_platform;
    int quest_make_it_move;
    int quest_stop_the_fall;
    int quest_open_the_gate;
    int last_print;
    char last_message[96];
} EduWorldState;

typedef enum {
    EDU_BUILTIN_SET_CRATE_SPEED = 0,
    EDU_BUILTIN_MOVE_CRATE,
    EDU_BUILTIN_STOP_CRATE,
    EDU_BUILTIN_OPEN_GATE,
    EDU_BUILTIN_CLOSE_GATE,
    EDU_BUILTIN_SET_SWITCH,
    EDU_BUILTIN_IS_SWITCH_ON,
    EDU_BUILTIN_QUERY_GROUNDED,
    EDU_BUILTIN_MARK_QUEST_COMPLETE,
    EDU_BUILTIN_GET_QUEST_STATE,
    EDU_BUILTIN_PRINT,
    EDU_BUILTIN_SHOW_MESSAGE,
    EDU_BUILTIN_COUNT
} EduBuiltinId;

int edu_binding_lookup(const char *name, int name_len, int *out_id, int *out_argc);
int edu_binding_call(EduWorldState *world, int builtin_id, const int *args, int argc,
                     const char *const *strings, int string_count, int *out_ret,
                     char *debug_out, int debug_cap);

#endif
