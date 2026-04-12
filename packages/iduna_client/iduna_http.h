#ifndef IDUNA_HTTP_H
#define IDUNA_HTTP_H

#include <stddef.h>

int iduna_http_post_json(const char *url,
                         const char *json_body,
                         const char *bearer_token,
                         char *out_buf,
                         size_t out_buf_sz,
                         long *out_status);

int iduna_http_get(const char *url,
                   const char *bearer_token,
                   char *out_buf,
                   size_t out_buf_sz,
                   long *out_status);

int iduna_json_get_string(const char *json,
                          const char *key,
                          char *out,
                          size_t out_sz);
int iduna_json_get_int(const char *json, const char *key, int *out_value);

#endif
