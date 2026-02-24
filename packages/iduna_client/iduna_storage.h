#ifndef IDUNA_STORAGE_H
#define IDUNA_STORAGE_H

#include <stddef.h>

int iduna_storage_save_jwt(const char *api_base, const char *access_token);
int iduna_storage_load_jwt(char *out_api_base, size_t api_base_len, char *out_access_token, size_t token_len);
int iduna_storage_delete_jwt(void);

#endif
