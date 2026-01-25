#ifndef CONFIG_LOADER_H
#define CONFIG_LOADER_H

#include <stdio.h>
#include <string.h>
#include <stdlib.h>

typedef struct {
    char master_ip[32];
    int master_port;
    int game_port;
    char player_name[32];
    float sensitivity;
    int width, height;
} GameConfig;

void load_config(const char* filename, GameConfig* cfg) {
    // Defaults
    strcpy(cfg->master_ip, "127.0.0.1");
    cfg->master_port = 8080;
    cfg->game_port = 6969;
    strcpy(cfg->player_name, "Guest");
    cfg->sensitivity = 0.1f;
    cfg->width = 1280;
    cfg->height = 720;

    FILE* f = fopen(filename, "r");
    if (!f) return;

    char line[128];
    while (fgets(line, sizeof(line), f)) {
        char key[64], val[64];
        if (sscanf(line, "%[^=]=%s", key, val) == 2) {
            if (strcmp(key, "MasterIP") == 0) strcpy(cfg->master_ip, val);
            if (strcmp(key, "MasterPort") == 0) cfg->master_port = atoi(val);
            if (strcmp(key, "GamePort") == 0) cfg->game_port = atoi(val);
            if (strcmp(key, "PlayerName") == 0) strcpy(cfg->player_name, val);
            if (strcmp(key, "Sensitivity") == 0) cfg->sensitivity = atof(val);
            if (strcmp(key, "Width") == 0) cfg->width = atoi(val);
            if (strcmp(key, "Height") == 0) cfg->height = atoi(val);
        }
    }
    fclose(f);
}

#endif
