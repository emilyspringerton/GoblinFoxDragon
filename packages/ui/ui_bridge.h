#ifndef UI_BRIDGE_H
#define UI_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

#define UI_BRIDGE_MAX_ENTRIES 16
#define UI_BRIDGE_ID_LEN 64
#define UI_BRIDGE_LABEL_LEN 96
#define UI_BRIDGE_KIND_LEN 16
#define UI_BRIDGE_SCENE_LEN 32
#define UI_BRIDGE_MODE_LEN 32

typedef struct {
    char id[UI_BRIDGE_ID_LEN];
    char label[UI_BRIDGE_LABEL_LEN];
    char kind[UI_BRIDGE_KIND_LEN];
    int enabled;
} UiMenuEntry;

typedef struct {
    char protocol_version[8];
    char active_scene_id[UI_BRIDGE_SCENE_LEN];
    char active_mode_id[UI_BRIDGE_MODE_LEN];
    int menu_open;
    int entry_count;
    UiMenuEntry entries[UI_BRIDGE_MAX_ENTRIES];
} UiState;

int ui_bridge_init(const char *host, int port);
int ui_bridge_fetch_state(UiState *out_state);
int ui_bridge_send_intent_activate(const char *entry_id, UiState *out_state);

#ifdef __cplusplus
}
#endif

#endif
