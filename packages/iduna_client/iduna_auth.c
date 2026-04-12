#include "iduna_auth.h"
#include "iduna_http.h"
#include "iduna_storage.h"

#include <SDL2/SDL.h>
#include <stdio.h>
#include <string.h>

static void iduna_set_status(IdunaAuth *auth, const char *msg) {
    if (!auth) return;
    snprintf(auth->status_msg, sizeof(auth->status_msg), "%s", msg ? msg : "");
}

void iduna_auth_init(IdunaAuth *auth) {
    if (!auth) return;
    memset(auth, 0, sizeof(*auth));
    auth->state = IDUNA_IDLE;
    auth->poll_interval = 3.0;
    iduna_set_status(auth, "AUTH: Idle.");
}

int iduna_auth_begin(IdunaAuth *auth, const char *api_base, double now_seconds) {
    if (!auth || !api_base) return 0;
    auth->state = IDUNA_STARTING;
    snprintf(auth->api_base, sizeof(auth->api_base), "%s", api_base);
    iduna_set_status(auth, "AUTH: Requesting device code...");

    char url[512];
    char response[4096];
    long http_status = 0;
    snprintf(url, sizeof(url), "%s/auth/device/start", auth->api_base);
    if (!iduna_http_post_json(url, "{}", NULL, response, sizeof(response), &http_status)) {
        auth->state = IDUNA_ERROR;
        iduna_set_status(auth, "AUTH: Network error starting flow.");
        return 0;
    }

    auth->last_http_status = (int)http_status;
    int expires = 300;
    int interval = 3;
    int ok = iduna_json_get_string(response, "device_code", auth->device_code, sizeof(auth->device_code))
        && iduna_json_get_string(response, "user_code", auth->user_code, sizeof(auth->user_code))
        && iduna_json_get_string(response, "verification_url", auth->verification_url, sizeof(auth->verification_url));
    iduna_json_get_int(response, "expires_in", &expires);
    iduna_json_get_int(response, "interval", &interval);

    if (!ok || http_status >= 400) {
        auth->state = IDUNA_ERROR;
        iduna_set_status(auth, "AUTH: Failed to start device login.");
        return 0;
    }

    auth->poll_interval = interval > 0 ? (double)interval : 3.0;
    auth->expires_at = now_seconds + (double)expires;
    auth->next_poll_at = now_seconds + auth->poll_interval;
    auth->state = IDUNA_SHOW_CODE;
    iduna_set_status(auth, "AUTH: Open browser to complete login.");
    return 1;
}

void iduna_auth_open_browser(IdunaAuth *auth, double now_seconds) {
    if (!auth) return;
    if ((auth->state == IDUNA_SHOW_CODE || auth->state == IDUNA_POLLING) && auth->verification_url[0]) {
        SDL_OpenURL(auth->verification_url);
        auth->state = IDUNA_POLLING;
        auth->next_poll_at = now_seconds + auth->poll_interval;
        iduna_set_status(auth, "AUTH: Browser opened. Waiting...");
    }
}

void iduna_auth_restart(IdunaAuth *auth, double now_seconds) {
    if (!auth) return;
    char api_base[256];
    snprintf(api_base, sizeof(api_base), "%s", auth->api_base);
    iduna_auth_init(auth);
    iduna_auth_begin(auth, api_base, now_seconds);
}

static void iduna_auth_exchange(IdunaAuth *auth) {
    char url[512];
    char body[256];
    char response[4096];
    long http_status = 0;

    snprintf(url, sizeof(url), "%s/auth/token/exchange", auth->api_base);
    snprintf(body, sizeof(body), "{\"exchange_code\":\"%s\"}", auth->exchange_code);
    if (!iduna_http_post_json(url, body, NULL, response, sizeof(response), &http_status)) {
        auth->state = IDUNA_POLLING;
        auth->next_poll_at = SDL_GetTicks() / 1000.0 + auth->poll_interval;
        iduna_set_status(auth, "AUTH: Exchange network issue. Retrying...");
        return;
    }
    auth->last_http_status = (int)http_status;

    if (http_status >= 400) {
        if (strstr(response, "ACCOUNT_SUSPENDED")) {
            auth->state = IDUNA_SUSPENDED;
            iduna_set_status(auth, "AUTH: Account suspended.");
            return;
        }
        if (strstr(response, "HONOR_CODE_REQUIRED") || strstr(response, "HANDLE_REQUIRED")) {
            auth->state = IDUNA_POLLING;
            auth->next_poll_at = SDL_GetTicks() / 1000.0 + auth->poll_interval;
            iduna_set_status(auth, "AUTH: Finish registration in browser.");
            return;
        }
        auth->state = IDUNA_ERROR;
        iduna_set_status(auth, "AUTH: Exchange invalid. Restart.");
        return;
    }

    if (!iduna_json_get_string(response, "access_token", auth->access_token, sizeof(auth->access_token))) {
        auth->state = IDUNA_ERROR;
        iduna_set_status(auth, "AUTH: Missing token in exchange.");
        return;
    }

    auth->state = IDUNA_READY;
    iduna_set_status(auth, "AUTH: Ready.");
}

void iduna_auth_update(IdunaAuth *auth, double now_seconds) {
    if (!auth) return;

    if (auth->state == IDUNA_SHOW_CODE && auth->device_code[0]) {
        auth->state = IDUNA_POLLING;
        if (auth->next_poll_at < now_seconds) auth->next_poll_at = now_seconds + auth->poll_interval;
    }

    if (auth->state == IDUNA_POLLING) {
        if (now_seconds >= auth->expires_at) {
            auth->state = IDUNA_EXPIRED;
            iduna_set_status(auth, "AUTH: Code expired. Press R to restart.");
            return;
        }
        if (now_seconds < auth->next_poll_at) return;

        char url[512];
        char body[256];
        char response[1024];
        long http_status = 0;
        int interval = (int)auth->poll_interval;

        snprintf(url, sizeof(url), "%s/auth/device/poll", auth->api_base);
        snprintf(body, sizeof(body), "{\"device_code\":\"%s\"}", auth->device_code);

        if (!iduna_http_post_json(url, body, NULL, response, sizeof(response), &http_status)) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            iduna_set_status(auth, "AUTH: Poll network issue. Retrying...");
            return;
        }
        auth->last_http_status = (int)http_status;
        iduna_json_get_int(response, "interval", &interval);
        if (interval > 0) auth->poll_interval = (double)interval;

        if (http_status == 429) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            iduna_set_status(auth, "AUTH: Polling too fast; backing off.");
            return;
        }
        if (http_status == 400) {
            auth->state = IDUNA_EXPIRED;
            iduna_set_status(auth, "AUTH: Expired/invalid. Restart.");
            return;
        }
        if (strstr(response, "\"status\":\"pending\"")) {
            auth->next_poll_at = now_seconds + auth->poll_interval;
            iduna_set_status(auth, "AUTH: Waiting for browser completion...");
            return;
        }
        if (strstr(response, "\"status\":\"authorized\"")) {
            if (iduna_json_get_string(response, "exchange_code", auth->exchange_code, sizeof(auth->exchange_code))) {
                auth->state = IDUNA_EXCHANGING;
                iduna_set_status(auth, "AUTH: Authorized. Exchanging...");
            } else {
                auth->state = IDUNA_ERROR;
                iduna_set_status(auth, "AUTH: Missing exchange code.");
            }
            return;
        }

        auth->next_poll_at = now_seconds + auth->poll_interval;
    }

    if (auth->state == IDUNA_EXCHANGING) {
        iduna_auth_exchange(auth);
    }
}

int iduna_auth_try_restore(IdunaAuth *auth, const char *api_base) {
    if (!auth || !api_base) return 0;
    char stored_base[256];
    char stored_tok[2048];
    if (!iduna_storage_load_jwt(stored_base, sizeof(stored_base), stored_tok, sizeof(stored_tok))) {
        return 0;
    }
    if (strcmp(stored_base, api_base) != 0) {
        return 0;
    }
    snprintf(auth->api_base, sizeof(auth->api_base), "%s", stored_base);
    snprintf(auth->access_token, sizeof(auth->access_token), "%s", stored_tok);
    auth->state = IDUNA_READY;
    iduna_set_status(auth, "AUTH: Restored token.");
    return 1;
}

int iduna_auth_validate_token(IdunaAuth *auth) {
    if (!auth || !auth->api_base[0] || !auth->access_token[0]) return 0;
    char url[512];
    char response[2048];
    long http_status = 0;
    snprintf(url, sizeof(url), "%s/me", auth->api_base);
    if (!iduna_http_get(url, auth->access_token, response, sizeof(response), &http_status)) {
        return 0;
    }
    auth->last_http_status = (int)http_status;
    if (http_status != 200) {
        return 0;
    }
    if (strstr(response, "suspended")) return 0;
    return 1;
}
