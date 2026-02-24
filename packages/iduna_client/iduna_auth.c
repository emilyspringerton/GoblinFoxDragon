#include "iduna_auth.h"

#include "iduna_http.h"
#include "iduna_storage.h"

#include <SDL2/SDL.h>
#include <stdio.h>
#include <string.h>

static int json_get_string(const char *json, const char *key, char *out, size_t out_len) {
    if (!json || !key || !out || out_len == 0) return -1;
    char needle[64];
    snprintf(needle, sizeof(needle), "\"%s\"", key);
    const char *p = strstr(json, needle);
    if (!p) return -1;
    p = strchr(p, ':');
    if (!p) return -1;
    p++;
    while (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r') p++;
    if (*p != '"') return -1;
    p++;

    size_t i = 0;
    while (*p && *p != '"' && i + 1 < out_len) {
        if (*p == '\\' && p[1]) p++;
        out[i++] = *p++;
    }
    out[i] = '\0';
    return (*p == '"') ? 0 : -1;
}

static int json_get_int(const char *json, const char *key, int *out_value) {
    if (!json || !key || !out_value) return -1;
    char needle[64];
    snprintf(needle, sizeof(needle), "\"%s\"", key);
    const char *p = strstr(json, needle);
    if (!p) return -1;
    p = strchr(p, ':');
    if (!p) return -1;
    p++;
    while (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r') p++;
    *out_value = atoi(p);
    return 0;
}

void iduna_auth_init(IdunaAuth *auth) {
    if (!auth) return;
    memset(auth, 0, sizeof(*auth));
    auth->state = IDUNA_IDLE;
    auth->poll_interval = 5.0;
    snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Press L to login.");
}

int iduna_auth_begin(IdunaAuth *auth, const char *api_base, double now_seconds) {
    if (!auth || !api_base || !api_base[0]) return -1;

    memset(auth->verification_url, 0, sizeof(auth->verification_url));
    memset(auth->user_code, 0, sizeof(auth->user_code));
    memset(auth->device_code, 0, sizeof(auth->device_code));
    memset(auth->exchange_code, 0, sizeof(auth->exchange_code));
    memset(auth->access_token, 0, sizeof(auth->access_token));

    snprintf(auth->api_base, sizeof(auth->api_base), "%s", api_base);
    auth->state = IDUNA_STARTING;
    snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Requesting device code...");

    char url[384];
    snprintf(url, sizeof(url), "%s/auth/device/start", auth->api_base);
    char body[4096];
    long http_status = 0;
    int rc = iduna_http_post_json(url, "{}", NULL, body, sizeof(body), &http_status);
    auth->last_http_status = (int)http_status;
    if (rc != 0 || http_status >= 400) {
        auth->state = IDUNA_ERROR;
        snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: start failed. Press R.");
        return -1;
    }

    int interval = 5;
    int expires = 600;
    if (json_get_string(body, "device_code", auth->device_code, sizeof(auth->device_code)) != 0 ||
        json_get_string(body, "user_code", auth->user_code, sizeof(auth->user_code)) != 0 ||
        json_get_string(body, "verification_url", auth->verification_url, sizeof(auth->verification_url)) != 0) {
        auth->state = IDUNA_ERROR;
        snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: malformed start response.");
        return -1;
    }
    json_get_int(body, "interval", &interval);
    json_get_int(body, "expires_in", &expires);

    auth->poll_interval = interval > 0 ? (double)interval : 5.0;
    auth->expires_at = now_seconds + (double)(expires > 0 ? expires : 600);
    auth->next_poll_at = now_seconds + auth->poll_interval;
    auth->state = IDUNA_SHOW_CODE;
    snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Open browser to complete login.");
    return 0;
}

void iduna_auth_open_browser(IdunaAuth *auth, double now_seconds) {
    if (!auth) return;
    if (auth->state != IDUNA_SHOW_CODE && auth->state != IDUNA_POLLING) return;
    if (auth->verification_url[0]) SDL_OpenURL(auth->verification_url);
    auth->state = IDUNA_POLLING;
    auth->next_poll_at = now_seconds + auth->poll_interval;
    snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Browser opened. Waiting...");
}

static void iduna_exchange(IdunaAuth *auth, double now_seconds) {
    char url[384];
    snprintf(url, sizeof(url), "%s/auth/token/exchange", auth->api_base);
    char req[256];
    snprintf(req, sizeof(req), "{\"exchange_code\":\"%s\"}", auth->exchange_code);
    char body[8192];
    long http_status = 0;
    int rc = iduna_http_post_json(url, req, NULL, body, sizeof(body), &http_status);
    auth->last_http_status = (int)http_status;
    if (rc != 0) {
        auth->state = IDUNA_POLLING;
        auth->next_poll_at = now_seconds + auth->poll_interval;
        snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: exchange network retry...");
        return;
    }

    if (http_status >= 400) {
        if (strstr(body, "ACCOUNT_SUSPENDED")) {
            auth->state = IDUNA_SUSPENDED;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Account suspended.");
        } else if (strstr(body, "HONOR_CODE_REQUIRED") || strstr(body, "HANDLE_REQUIRED")) {
            auth->state = IDUNA_POLLING;
            auth->next_poll_at = now_seconds + auth->poll_interval;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Finish registration in browser.");
        } else if (strstr(body, "EXCHANGE_CODE_INVALID")) {
            auth->state = IDUNA_ERROR;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Exchange invalid. Press R.");
        } else {
            auth->state = IDUNA_ERROR;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Exchange failed. Press R.");
        }
        return;
    }

    if (json_get_string(body, "access_token", auth->access_token, sizeof(auth->access_token)) != 0) {
        auth->state = IDUNA_ERROR;
        snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: token parse failed.");
        return;
    }

    auth->state = IDUNA_READY;
    snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Ready.");
}

void iduna_auth_update(IdunaAuth *auth, double now_seconds) {
    if (!auth) return;

    if (auth->state == IDUNA_SHOW_CODE && now_seconds >= auth->next_poll_at) {
        auth->state = IDUNA_POLLING;
    }

    if (auth->state == IDUNA_POLLING) {
        if (now_seconds >= auth->expires_at) {
            auth->state = IDUNA_EXPIRED;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Code expired. Press R.");
            return;
        }
        if (now_seconds < auth->next_poll_at) return;

        char url[384];
        snprintf(url, sizeof(url), "%s/auth/device/poll", auth->api_base);
        char req[256];
        snprintf(req, sizeof(req), "{\"device_code\":\"%s\"}", auth->device_code);
        char body[4096];
        long http_status = 0;
        int rc = iduna_http_post_json(url, req, NULL, body, sizeof(body), &http_status);
        auth->last_http_status = (int)http_status;

        if (rc != 0) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: network retry...");
            return;
        }

        int interval = (int)auth->poll_interval;
        int expires = 0;
        json_get_int(body, "interval", &interval);
        json_get_int(body, "expires_in", &expires);
        if (expires > 0) auth->expires_at = now_seconds + (double)expires;

        if (http_status == 429) {
            if (interval < (int)auth->poll_interval) interval = (int)auth->poll_interval;
            auth->next_poll_at = now_seconds + (double)interval;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: wait before polling...");
            return;
        }
        if (http_status == 400) {
            auth->state = IDUNA_EXPIRED;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Expired/invalid. Press R.");
            return;
        }
        if (http_status >= 400) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: poll failed, retrying...");
            return;
        }

        char status[32];
        if (json_get_string(body, "status", status, sizeof(status)) != 0) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            return;
        }

        if (strcmp(status, "pending") == 0) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: waiting for browser confirmation...");
        } else if (strcmp(status, "authorized") == 0) {
            if (json_get_string(body, "exchange_code", auth->exchange_code, sizeof(auth->exchange_code)) == 0) {
                auth->state = IDUNA_EXCHANGING;
                snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Authorized. Exchanging...");
            }
        }
    }

    if (auth->state == IDUNA_EXCHANGING) {
        iduna_exchange(auth, now_seconds);
    }
}

int iduna_auth_try_resume_saved(IdunaAuth *auth, const char *default_api_base) {
    if (!auth) return -1;

    char stored_base[256];
    char token[2048];
    if (iduna_storage_load_jwt(stored_base, sizeof(stored_base), token, sizeof(token)) != 0) {
        return iduna_auth_begin(auth, default_api_base, SDL_GetTicks() / 1000.0);
    }

    snprintf(auth->api_base, sizeof(auth->api_base), "%s", stored_base[0] ? stored_base : default_api_base);

    char url[384];
    snprintf(url, sizeof(url), "%s/me", auth->api_base);
    char body[4096];
    long http_status = 0;
    int rc = iduna_http_get(url, token, body, sizeof(body), &http_status);
    auth->last_http_status = (int)http_status;

    if (rc == 0 && http_status == 200 && strstr(body, "\"status\":\"active\"")) {
        snprintf(auth->access_token, sizeof(auth->access_token), "%s", token);
        auth->state = IDUNA_READY;
        auth->overlay_open = 0;
        snprintf(auth->status_msg, sizeof(auth->status_msg), "AUTH: Ready (saved session).");
        return 0;
    }

    iduna_storage_delete_jwt();
    return iduna_auth_begin(auth, default_api_base, SDL_GetTicks() / 1000.0);
}
