#ifndef IDUNA_STORAGE_H
#define IDUNA_STORAGE_H

#include <stddef.h>

int iduna_storage_save_jwt(const char *api_base, const char *access_token);
int iduna_storage_load_jwt(char *api_base, size_t api_base_sz, char *access_token, size_t access_token_sz);
int iduna_storage_clear(void);

#endif
