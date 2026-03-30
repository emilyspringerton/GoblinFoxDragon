#ifndef EDU_BYTECODE_H
#define EDU_BYTECODE_H

#include <stdint.h>

#define EDU_MAX_BYTECODE 1024

typedef enum {
    EDU_OP_PUSH_INT = 1,
    EDU_OP_PUSH_BOOL,
    EDU_OP_PUSH_STR,
    EDU_OP_LOAD_VAR,
    EDU_OP_STORE_VAR,
    EDU_OP_POP,
    EDU_OP_ADD,
    EDU_OP_SUB,
    EDU_OP_MUL,
    EDU_OP_DIV,
    EDU_OP_EQ,
    EDU_OP_LT,
    EDU_OP_GT,
    EDU_OP_JMP,
    EDU_OP_JMP_IF_FALSE,
    EDU_OP_CALL_BUILTIN,
    EDU_OP_RETURN,
    EDU_OP_HALT
} EduOpcode;

typedef struct {
    unsigned char *data;
    int len;
    int cap;
    int error;
} EduBytecodeWriter;

void edu_bc_init(EduBytecodeWriter *w, unsigned char *buffer, int capacity);
int edu_bc_emit_u8(EduBytecodeWriter *w, uint8_t v);
int edu_bc_emit_i32(EduBytecodeWriter *w, int v);
int edu_bc_patch_i32(EduBytecodeWriter *w, int at, int v);
int edu_bc_emit_jump(EduBytecodeWriter *w, EduOpcode op);

#endif
