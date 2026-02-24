#include "iduna_storage.h"

#include <SDL2/SDL.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

static int iduna_storage_path(char *out_path, size_t out_path_len) {
    char *pref = SDL_GetPrefPath("RockBossStudios", "Kikoryu");
    if (!pref) return -1;
    snprintf(out_path, out_path_len, "%siduna_token.json", pref);
    SDL_free(pref);
    return 0;
}

static int iduna_extract_json_string(const char *json, const char *key, char *out, size_t out_len) {
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

int iduna_storage_save_jwt(const char *api_base, const char *access_token) {
    if (!api_base || !access_token) return -1;
    char path[1024];
    if (iduna_storage_path(path, sizeof(path)) != 0) return -1;

    FILE *f = fopen(path, "wb");
    if (!f) return -1;
    fprintf(f,
            "{\n"
            "  \"api_base\":\"%s\",\n"
            "  \"access_token\":\"%s\",\n"
            "  \"saved_at\":%ld\n"
            "}\n",
            api_base,
            access_token,
            (long)time(NULL));
    fclose(f);
    return 0;
}

int iduna_storage_load_jwt(char *out_api_base, size_t api_base_len, char *out_access_token, size_t token_len) {
    if (!out_api_base || !out_access_token || api_base_len == 0 || token_len == 0) return -1;

    char path[1024];
    if (iduna_storage_path(path, sizeof(path)) != 0) return -1;
    FILE *f = fopen(path, "rb");
    if (!f) return -1;

    char json[8192];
    size_t n = fread(json, 1, sizeof(json) - 1, f);
    fclose(f);
    json[n] = '\0';

    if (iduna_extract_json_string(json, "api_base", out_api_base, api_base_len) != 0) return -1;
    if (iduna_extract_json_string(json, "access_token", out_access_token, token_len) != 0) return -1;
    return 0;
}

int iduna_storage_delete_jwt(void) {
    char path[1024];
    if (iduna_storage_path(path, sizeof(path)) != 0) return -1;
    return remove(path);
}
