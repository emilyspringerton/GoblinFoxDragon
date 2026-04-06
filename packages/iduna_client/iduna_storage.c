#include "iduna_storage.h"
#include "iduna_http.h"

#include <SDL2/SDL.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

static int iduna_storage_path(char *out, size_t out_sz) {
    char *pref = SDL_GetPrefPath("RockBossStudios", "Kikoryu");
    if (!pref) return 0;
    snprintf(out, out_sz, "%siduna_auth.json", pref);
    SDL_free(pref);
    return 1;
}

int iduna_storage_save_jwt(const char *api_base, const char *access_token) {
    char path[512];
    if (!iduna_storage_path(path, sizeof(path))) return 0;
    FILE *f = fopen(path, "wb");
    if (!f) return 0;
    long now = (long)time(NULL);
    fprintf(f, "{\n  \"api_base\":\"%s\",\n  \"access_token\":\"%s\",\n  \"saved_at\":%ld\n}\n",
            api_base ? api_base : "", access_token ? access_token : "", now);
    fclose(f);
    return 1;
}

int iduna_storage_load_jwt(char *api_base, size_t api_base_sz, char *access_token, size_t access_token_sz) {
    char path[512];
    if (!iduna_storage_path(path, sizeof(path))) return 0;
    FILE *f = fopen(path, "rb");
    if (!f) return 0;
    char data[4096];
    size_t n = fread(data, 1, sizeof(data) - 1, f);
    fclose(f);
    data[n] = '\0';

    int ok_base = iduna_json_get_string(data, "api_base", api_base, api_base_sz);
    int ok_tok = iduna_json_get_string(data, "access_token", access_token, access_token_sz);
    if (!ok_base || !ok_tok) {
        if (api_base && api_base_sz) api_base[0] = '\0';
        if (access_token && access_token_sz) access_token[0] = '\0';
        return 0;
    }
    return 1;
}

int iduna_storage_clear(void) {
    char path[512];
    if (!iduna_storage_path(path, sizeof(path))) return 0;
    return remove(path) == 0;
}
