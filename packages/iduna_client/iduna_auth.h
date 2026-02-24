#ifndef IDUNA_AUTH_H
#define IDUNA_AUTH_H

#include <stddef.h>

typedef enum IdunaAuthState {
    IDUNA_IDLE = 0,
    IDUNA_STARTING,
    IDUNA_SHOW_CODE,
    IDUNA_POLLING,
    IDUNA_EXCHANGING,
    IDUNA_READY,
    IDUNA_ERROR,
    IDUNA_SUSPENDED,
    IDUNA_EXPIRED
} IdunaAuthState;

typedef struct IdunaAuth {
    IdunaAuthState state;
    char api_base[256];
    char verification_url[256];
    char user_code[32];
    char device_code[128];
    char exchange_code[128];
    char access_token[2048];
    double expires_at;
    double poll_interval;
    double next_poll_at;
    char status_msg[256];
    int last_http_status;
    int overlay_open;
} IdunaAuth;

void iduna_auth_init(IdunaAuth *auth);
int iduna_auth_begin(IdunaAuth *auth, const char *api_base, double now_seconds);
void iduna_auth_open_browser(IdunaAuth *auth, double now_seconds);
void iduna_auth_update(IdunaAuth *auth, double now_seconds);
int iduna_auth_try_resume_saved(IdunaAuth *auth, const char *default_api_base);

#endif
