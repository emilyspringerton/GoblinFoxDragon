#include <string.h>
#include <stdio.h>
#include "edu_script.h"

static void seed_script(EduScriptSlot *s, const char *name, const char *src) {
    snprintf(s->name, sizeof(s->name), "%s", name);
    snprintf(s->source, sizeof(s->source), "%s", src);
    s->source_len = (int)strlen(s->source);
}

void edu_script_init(EduScriptSystem *sys) {
    memset(sys, 0, sizeof(*sys));
    sys->world.crate_speed = 1;
    sys->world.grounded_test_platform = 1;
    sys->active_slot = 0;
    seed_script(&sys->slots[0], "Motion Terminal",
                "let speed = 2;\n"
                "set_crate_speed(speed);\n"
                "move_crate();\n"
                "print(speed);\n");
    seed_script(&sys->slots[1], "Gate Terminal",
                "if is_switch_on() {\n"
                "  open_gate();\n"
                "} else {\n"
                "  close_gate();\n"
                "}\n");
    seed_script(&sys->slots[2], "Collision Terminal",
                "let grounded = query_grounded(\"test_platform\");\n"
                "if grounded {\n"
                "  mark_quest_complete(\"STOP_THE_FALL\");\n"
                "}\n");
    snprintf(sys->compile_status, sizeof(sys->compile_status), "idle");
    snprintf(sys->run_status, sizeof(sys->run_status), "idle");
}

void edu_script_toggle_terminal(EduScriptSystem *sys) { sys->terminal_open = !sys->terminal_open; }

void edu_script_reset_active(EduScriptSystem *sys) {
    EduScriptSlot *s = &sys->slots[sys->active_slot];
    s->source[0] = 0;
    s->source_len = 0;
    s->compile_ok = 0;
    snprintf(sys->compile_status, sizeof(sys->compile_status), "reset");
    snprintf(sys->run_status, sizeof(sys->run_status), "reset");
}

void edu_script_insert_text(EduScriptSystem *sys, const char *text) {
    EduScriptSlot *s = &sys->slots[sys->active_slot];
    int len = (int)strlen(text);
    if (s->source_len + len >= EDU_MAX_SCRIPT_TEXT - 1) return;
    memcpy(s->source + s->source_len, text, (size_t)len);
    s->source_len += len;
    s->source[s->source_len] = '\0';
}

void edu_script_backspace(EduScriptSystem *sys) {
    EduScriptSlot *s = &sys->slots[sys->active_slot];
    if (s->source_len <= 0) return;
    s->source_len--;
    s->source[s->source_len] = '\0';
}

void edu_script_newline(EduScriptSystem *sys) { edu_script_insert_text(sys, "\n"); }

int edu_script_compile_active(EduScriptSystem *sys) {
    EduScriptSlot *s = &sys->slots[sys->active_slot];
    EduCompileOutput out;
    memset(&out, 0, sizeof(out));
    out.bytecode = s->bytecode;
    out.bytecode_cap = EDU_MAX_BYTECODE;
    int ok = edu_compile_source(s->source, &out);
    s->compile_ok = ok;
    s->bytecode_len = out.bytecode_len;
    s->string_count = out.string_count;
    for (int i = 0; i < out.string_count; i++) {
        snprintf(s->string_storage[i], sizeof(s->string_storage[i]), "%s", out.string_storage[i]);
        s->strings[i] = s->string_storage[i];
    }
    if (ok) {
        snprintf(sys->compile_status, sizeof(sys->compile_status), "ok (%d bytes)", s->bytecode_len);
    } else {
        snprintf(sys->compile_status, sizeof(sys->compile_status), "error %d:%d %.88s", out.error_line, out.error_col, out.error);
    }
    return ok;
}

int edu_script_run_active(EduScriptSystem *sys) {
    EduScriptSlot *s = &sys->slots[sys->active_slot];
    if (!s->compile_ok) {
        snprintf(sys->run_status, sizeof(sys->run_status), "compile first");
        return 0;
    }
    EduVm vm;
    EduVmLimits lim;
    memset(&lim, 0, sizeof(lim));
    lim.max_instructions = 256;
    lim.max_stack = 128;
    lim.max_vars = 64;
    EduVmProgramMeta meta;
    meta.strings = s->strings;
    meta.string_count = s->string_count;
    printf("[EDUSCRIPT] vm begin\n");
    int ok = edu_vm_exec(&vm, s->bytecode, s->bytecode_len, &meta, &sys->world, &lim, sys->run_status, (int)sizeof(sys->run_status));
    if (ok) {
        snprintf(sys->last_output, sizeof(sys->last_output), "%.96s | print=%d", vm.debug_line, sys->world.last_print);
        printf("[EDUSCRIPT] vm halt instructions=%d\n", vm.instruction_count);
    } else {
        snprintf(sys->last_output, sizeof(sys->last_output), "runtime failure");
        if (lim.halted_due_to_limit) printf("[EDUSCRIPT] vm error limit exceeded\n");
    }
    sys->last_instruction_count = vm.instruction_count;
    return ok;
}
