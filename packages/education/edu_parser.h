#ifndef EDU_PARSER_H
#define EDU_PARSER_H

#include "edu_bindings.h"

#define EDU_MAX_VARS 64
#define EDU_MAX_STRINGS 64

typedef struct {
    char name[32];
    int slot;
} EduVarEntry;

typedef struct {
    unsigned char *bytecode;
    int bytecode_cap;
    int bytecode_len;
    const char *strings[EDU_MAX_STRINGS];
    char string_storage[EDU_MAX_STRINGS][64];
    int string_count;
    int var_count;
    EduVarEntry vars[EDU_MAX_VARS];
    int ok;
    char error[160];
    int error_line;
    int error_col;
} EduCompileOutput;

int edu_compile_source(const char *source, EduCompileOutput *out);

#endif
