#ifndef IDUNA_HTTP_H
#define IDUNA_HTTP_H

#include <stddef.h>

int iduna_http_post_json(const char *url,
                         const char *json_body,
                         const char *auth_bearer_token,
                         char *out_body,
                         size_t out_body_len,
                         long *out_http_status);

int iduna_http_get(const char *url,
                   const char *auth_bearer_token,
                   char *out_body,
                   size_t out_body_len,
                   long *out_http_status);

#endif
