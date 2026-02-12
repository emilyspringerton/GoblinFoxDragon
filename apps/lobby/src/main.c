#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <ctype.h>

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #pragma comment(lib, "ws2_32.lib")
#else
    #include <sys/socket.h>
    #include <netinet/in.h>
    #include <arpa/inet.h>
    #include <unistd.h>
    #include <fcntl.h>
    #include <netdb.h>
#endif

#include <SDL2/SDL.h>
#include <SDL2/SDL_opengl.h>
#include <GL/glu.h>

#include "player_model.h"
#include "../../../packages/ui/turtle_text.h"
#define UI_BRIDGE_DECL static
#include "../../../packages/ui/ui_bridge.h"
#include "../../../packages/ui/ui_bridge.c"

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"
#include "../../../packages/simulation/local_game.h"

#define STATE_LOBBY 0
#define STATE_GAME_NET 1
#define STATE_GAME_LOCAL 2
#define STATE_RESULTS 3
#define STATE_LISTEN_SERVER 99

char SERVER_HOST[64] = "s.farthq.com";
int SERVER_PORT = 6969;

int app_state = STATE_LOBBY;
int wpn_req = 1; 
int my_client_id = -1;
int lobby_selection = 0;

UiState ui_state;
int ui_use_server = 0;
unsigned int ui_last_poll = 0;
int ui_edit_index = -1;
int ui_edit_len = 0;
unsigned int ui_last_click_ms = 0;
int ui_last_click_index = -1;
char ui_edit_buffer[64];
unsigned int travel_overlay_until_ms = 0;

float current_fov = 75.0f;
int g_ads_down = 0;

typedef struct {
    int ammo_shell_casings;
    int loaded;
} ClientSettings;

static ClientSettings g_settings;
static int g_server_force_shell_casings = -1; /* -1 no override, 0 forced off, 1 forced on */
static int g_show_settings = 0;
static int g_settings_selection = 0;

static ClientSettings settings_defaults(void) {
    ClientSettings s;
    memset(&s, 0, sizeof(s));
    s.ammo_shell_casings = 1;
    s.loaded = 1;
    return s;
}

static char* trim_in_place(char *s) {
    while (*s && isspace((unsigned char)*s)) s++;
    if (!*s) return s;
    char *end = s + strlen(s) - 1;
    while (end > s && isspace((unsigned char)*end)) {
        *end = '\0';
        end--;
    }
    return s;
}

static void strip_comment_in_place(char *s) {
    char *hash = strchr(s, '#');
    char *slash = strstr(s, "//");
    char *cut = NULL;
    if (hash) cut = hash;
    if (slash && (!cut || slash < cut)) cut = slash;
    if (cut) *cut = '\0';
}

static int parse_bool(const char *v, int *out) {
    if (!v || !out) return 0;
    if (SDL_strcasecmp(v, "true") == 0 || strcmp(v, "1") == 0) {
        *out = 1;
        return 1;
    }
    if (SDL_strcasecmp(v, "false") == 0 || strcmp(v, "0") == 0) {
        *out = 0;
        return 1;
    }
    return 0;
}

static void settings_load_if_needed(void) {
    if (g_settings.loaded) return;

    g_settings = settings_defaults();

    FILE *f = fopen("settings.toml", "rb");
    if (!f) {
        g_settings.loaded = 1;
        return;
    }

    char line[256];
    while (fgets(line, sizeof(line), f)) {
        strip_comment_in_place(line);
        char *trimmed = trim_in_place(line);
        if (!trimmed[0]) continue;

        char *eq = strchr(trimmed, '=');
        if (!eq) continue;
        *eq = '\0';

        char *key = trim_in_place(trimmed);
        char *value = trim_in_place(eq + 1);
        if (!key[0] || !value[0]) continue;

        if (strcmp(key, "ammo_shell_casings") == 0) {
            int parsed = 0;
            if (parse_bool(value, &parsed)) {
                g_settings.ammo_shell_casings = parsed;
            }
        }
    }

    fclose(f);
    g_settings.loaded = 1;
}

static void settings_save(void) {
    FILE *f = fopen("settings.toml.tmp", "wb");
    if (!f) return;

    const char *v = g_settings.ammo_shell_casings ? "true" : "false";
    fprintf(f, "ammo_shell_casings = %s\n", v);
    fflush(f);
    fclose(f);

    remove("settings.toml");
    if (rename("settings.toml.tmp", "settings.toml") != 0) {
        remove("settings.toml.tmp");
    }
}

static int shell_casings_enabled(void) {
    settings_load_if_needed();
    if (g_server_force_shell_casings != -1) return g_server_force_shell_casings;
    return g_settings.ammo_shell_casings;
}

static void settings_toggle_selected(void) {
    settings_load_if_needed();
    if (g_server_force_shell_casings != -1) return;
    if (g_settings_selection == 0) {
        g_settings.ammo_shell_casings = !g_settings.ammo_shell_casings;
        settings_save();
    }
}

static void settings_toggle_overlay(void) {
    settings_load_if_needed();
    g_show_settings = !g_show_settings;
}

static void settings_set_server_shell_casings_override(int v) {
    if (v < -1 || v > 1) {
        g_server_force_shell_casings = -1;
        return;
    }
    g_server_force_shell_casings = v;
}

/*
 * Camera/aim world convention (single source of truth):
 * +Y is up.
 * Yaw 0 faces +Z.
 * Positive yaw rotates toward +X.
 * Forward(yaw) = (sin(yaw), 0, cos(yaw)).
 * Right(yaw) = (cos(yaw), 0, -sin(yaw)).
 * Positive pitch looks up.
 */
typedef struct { float x, y, z; } CameraVec3;
typedef struct { CameraVec3 origin, dir; } AimRay;
typedef enum { CAM_FIRST = 0, CAM_THIRD = 1 } CamMode;
typedef struct {
    float yaw, pitch;
    float dist, height, side;
    CameraVec3 pos;
    int ads_down;
    CamMode mode;
} CameraState;

static CameraState g_cam = {
    0.0f, 0.0f,
    6.5f, 2.4f, 1.25f,
    {0.0f, 0.0f, 0.0f},
    0,
    CAM_THIRD
};

static float reticle_dx = 0.0f;
static float reticle_dy = 0.0f;

#define HUD_LOG_LINES 8
#define HUD_LOG_LINE_LEN 96

#ifndef SHANKPIT_HUD_STATE_DECLARED
#define SHANKPIT_HUD_STATE_DECLARED 1
static char hud_log[HUD_LOG_LINES][HUD_LOG_LINE_LEN];
static unsigned int hud_log_time[HUD_LOG_LINES];
typedef struct {
    int head;
    int was_dragon_heat;
    int was_huntsman_spiderlings;
} HudState;
static HudState g_hud_state = {0};
#endif

#define Z_FAR 8000.0f

int sock = -1;
struct sockaddr_in server_addr;

#define NET_CMD_HISTORY 3
UserCmd net_cmd_history[NET_CMD_HISTORY];
int net_cmd_history_count = 0;
int net_cmd_seq = 0;

void net_connect();
void net_shutdown();

typedef struct {
    int level;
    int xp;
    int unspent_points;
    int ability[5];
    int last_kills;
    unsigned int last_xp_tick;
    unsigned int match_start_ms;
    unsigned int match_end_ms;
    int camera_third_person;
} MatchProgression;

static MatchProgression match_prog;
#define MATCH_LENGTH_MS (8 * 60 * 1000)

static int xp_for_next_level(int level) {
    return 80 + (level - 1) * 35;
}

static void progression_reset(unsigned int now_ms) {
    memset(&match_prog, 0, sizeof(match_prog));
    match_prog.level = 1;
    match_prog.last_xp_tick = now_ms;
    match_prog.match_start_ms = now_ms;
    match_prog.match_end_ms = now_ms + MATCH_LENGTH_MS;
    match_prog.camera_third_person = 1;
}

static void progression_add_xp(int amount) {
    if (amount <= 0 || match_prog.level >= 10) return;
    match_prog.xp += amount;
    while (match_prog.level < 10) {
        int need = xp_for_next_level(match_prog.level);
        if (match_prog.xp < need) break;
        match_prog.xp -= need;
        match_prog.level++;
        match_prog.unspent_points++;
    }
}

static void progression_apply_bonuses(PlayerState *p) {
    if (!p) return;
    int move = match_prog.ability[0];
    int vitality = match_prog.ability[1];
    int handling = match_prog.ability[2];
    int shield = match_prog.ability[3];
    int storm = match_prog.ability[4];

    if (vitality > 0 && p->health < 100) {
        if ((SDL_GetTicks() % (70 - vitality * 8)) == 0) p->health += 1;
        if (p->health > 100) p->health = 100;
    }
    if (shield > 0 && p->shield < 100) {
        if ((SDL_GetTicks() % (80 - shield * 10)) == 0) p->shield += 1;
        if (p->shield > 100) p->shield = 100;
    }
    if (handling > 0 && p->attack_cooldown > 0) {
        int reduce = handling >= 4 ? 2 : 1;
        p->attack_cooldown -= reduce;
        if (p->attack_cooldown < 0) p->attack_cooldown = 0;
    }
    if (storm > 0 && p->ability_cooldown > 0 && (SDL_GetTicks() % 3 == 0)) {
        p->ability_cooldown -= storm;
        if (p->ability_cooldown < 0) p->ability_cooldown = 0;
    }
    if (move > 0 && !p->in_vehicle) {
        float boost = 1.0f + 0.035f * (float)move;
        p->vx *= boost;
        p->vz *= boost;
    }
}

static void progression_tick(PlayerState *p, unsigned int now_ms) {
    if (!p) return;
    if (match_prog.last_xp_tick == 0) match_prog.last_xp_tick = now_ms;
    if (now_ms - match_prog.last_xp_tick >= 1000) {
        progression_add_xp(5);
        match_prog.last_xp_tick = now_ms;
    }
    if (p->kills > match_prog.last_kills) {
        int delta = p->kills - match_prog.last_kills;
        progression_add_xp(delta * 60);
        match_prog.last_kills = p->kills;
    }
    progression_apply_bonuses(p);
}

static void progression_try_allocate(int idx) {
    if (idx < 0 || idx >= 5) return;
    if (match_prog.unspent_points <= 0) return;
    if (match_prog.ability[idx] >= 5) return;
    match_prog.ability[idx]++;
    match_prog.unspent_points--;
}


static void reset_client_render_state_for_net() {
    my_client_id = -1;
    memset(&local_state, 0, sizeof(local_state));
    memset(net_cmd_history, 0, sizeof(net_cmd_history));
    net_cmd_history_count = 0;
    net_cmd_seq = 0;
    travel_overlay_until_ms = 0;
    local_state.pending_scene = -1;
    local_state.scene_id = SCENE_GARAGE_OSAKA;
    phys_set_scene(local_state.scene_id);
}

void draw_string(const char* str, float x, float y, float size) {
    TurtlePen pen = turtle_pen_create(x, y, size);
    turtle_draw_text(&pen, str);
}

static const char* weapon_name(int weapon_id) {
    switch (weapon_id) {
        case WPN_KNIFE: return "KNIFE";
        case WPN_MAGNUM: return "MAGNUM";
        case WPN_AR: return "AR";
        case WPN_SHOTGUN: return "SHOTGUN";
        case WPN_SNIPER: return "SNIPER";
        default: return "UNKNOWN";
    }
}

static CameraVec3 forward_from_yaw(float yaw_deg) {
    float yaw = yaw_deg * 0.0174532925f;
    CameraVec3 v = { sinf(yaw), 0.0f, cosf(yaw) };
    return v;
}

static CameraVec3 right_from_yaw(float yaw_deg) {
    float yaw = yaw_deg * 0.0174532925f;
    CameraVec3 v = { cosf(yaw), 0.0f, -sinf(yaw) };
    return v;
}

static CameraVec3 forward_from_yaw_pitch(float yaw_deg, float pitch_deg) {
    float yaw = yaw_deg * 0.0174532925f;
    float pitch = pitch_deg * 0.0174532925f;
    float cp = cosf(pitch);
    CameraVec3 v = { sinf(yaw) * cp, sinf(pitch), cosf(yaw) * cp };
    return v;
}

static CamMode get_cam_mode(const PlayerState *me) {
    if (!me) return CAM_FIRST;
    if (me->in_vehicle) return CAM_THIRD;
    if (match_prog.camera_third_person && !g_cam.ads_down) return CAM_THIRD;
    return CAM_FIRST;
}

static void reticle_update_visual(const PlayerState *me, float dt) {
    CamMode mode = get_cam_mode(me);
    g_cam.mode = mode;

    float target_x = 0.0f;
    float target_y = 0.0f;
    if (mode == CAM_THIRD && !g_cam.ads_down) {
        target_x = 180.0f;
        target_y = 40.0f;
    }

    float safe_dt = dt;
    if (!isfinite(safe_dt) || safe_dt < 0.0f) safe_dt = 0.0f;
    if (safe_dt > 0.25f) safe_dt = 0.25f;
    float alpha = 1.0f - expf(-18.0f * safe_dt);
    reticle_dx += (target_x - reticle_dx) * alpha;
    reticle_dy += (target_y - reticle_dy) * alpha;

    if (reticle_dx > 420.0f) reticle_dx = 420.0f;
    if (reticle_dx < -420.0f) reticle_dx = -420.0f;
    if (reticle_dy > 220.0f) reticle_dy = 220.0f;
    if (reticle_dy < -220.0f) reticle_dy = -220.0f;
}

static AimRay get_aim_ray(const CameraState *cam, const PlayerState *me) {
    AimRay ray = {0};
    if (!cam || !me) return ray;

    float aim_yaw = cam->yaw;
    float aim_pitch = cam->pitch;
    if (reticle_dx != 0.0f || reticle_dy != 0.0f) {
        const float half_w = 1280.0f * 0.5f;
        const float half_h = 720.0f * 0.5f;
        float nx = reticle_dx / half_w;
        float ny = reticle_dy / half_h;
        float fov_rad = current_fov * 0.0174532925f;
        if (fov_rad < 0.01f) fov_rad = 0.01f;
        if (fov_rad > 3.0f) fov_rad = 3.0f;
        float tan_half_v = tanf(fov_rad * 0.5f);
        float tan_half_h = tan_half_v * (1280.0f / 720.0f);
        aim_yaw = norm_yaw_deg(aim_yaw + atanf(nx * tan_half_h) * 57.2957795f);
        aim_pitch = clamp_pitch_deg(aim_pitch + atanf(ny * tan_half_v) * 57.2957795f);
    }
    ray.dir = forward_from_yaw_pitch(aim_yaw, aim_pitch);

    float eye_y = me->in_vehicle ? 8.0f : (me->crouching ? 2.5f : EYE_HEIGHT);
    if (cam->mode == CAM_THIRD) {
        ray.origin = cam->pos;
    } else {
        ray.origin.x = me->x;
        ray.origin.y = me->y + eye_y;
        ray.origin.z = me->z;
    }
    return ray;
}


static void yaw_pitch_from_dir(CameraVec3 dir, float *yaw_out, float *pitch_out) {
    float yaw = atan2f(dir.x, dir.z) * 57.2957795f;
    float planar = sqrtf(dir.x * dir.x + dir.z * dir.z);
    float pitch = atan2f(dir.y, (planar > 0.0001f) ? planar : 0.0001f) * 57.2957795f;
    if (yaw_out) *yaw_out = norm_yaw_deg(yaw);
    if (pitch_out) *pitch_out = clamp_pitch_deg(pitch);
}

static void camera_update(CameraState *cam, const PlayerState *p, float dt) {
    if (!cam || !p) return;
    cam->mode = get_cam_mode(p);
    if (!isfinite(cam->yaw)) cam->yaw = 0.0f;
    if (!isfinite(cam->pitch)) cam->pitch = 0.0f;
    cam->yaw = norm_yaw_deg(cam->yaw);
    cam->pitch = clamp_pitch_deg(cam->pitch);

    if (cam->mode != CAM_THIRD) return;

    float pivot_x = p->x;
    float pivot_y = p->y + cam->height;
    float pivot_z = p->z;

    if (cam->pitch > 70.0f) cam->pitch = 70.0f;
    if (cam->pitch < -70.0f) cam->pitch = -70.0f;

    CameraVec3 forward = forward_from_yaw_pitch(cam->yaw, cam->pitch);
    CameraVec3 right = right_from_yaw(cam->yaw);

    float desired_x = pivot_x - forward.x * cam->dist + right.x * cam->side;
    float desired_y = pivot_y - forward.y * cam->dist;
    float desired_z = pivot_z - forward.z * cam->dist + right.z * cam->side;

    float dx = cam->pos.x - desired_x;
    float dy = cam->pos.y - desired_y;
    float dz = cam->pos.z - desired_z;
    float dist_err = sqrtf(dx * dx + dy * dy + dz * dz);
    if (dist_err > 30.0f || (cam->pos.x == 0.0f && cam->pos.y == 0.0f && cam->pos.z == 0.0f)) {
        cam->pos.x = desired_x;
        cam->pos.y = desired_y;
        cam->pos.z = desired_z;
        return;
    }

    float alpha = 1.0f - expf(-14.0f * dt);
    cam->pos.x += (desired_x - cam->pos.x) * alpha;
    cam->pos.y += (desired_y - cam->pos.y) * alpha;
    cam->pos.z += (desired_z - cam->pos.z) * alpha;
}

static void player_update_run_cycle(PlayerState *p, float dt) {
    if (!p || !p->active) return;
    float spd = sqrtf(p->vx * p->vx + p->vz * p->vz);
    float target = spd / (MAX_SPEED * 0.9f);
    if (target < 0.0f) target = 0.0f;
    if (target > 1.0f) target = 1.0f;
    if (!p->on_ground) target = 0.0f;
    p->run_weight += (target - p->run_weight) * 0.2f;
    p->run_phase += dt * (6.0f + 6.0f * p->run_weight);
}

static void camera_reseed_from_player(const PlayerState *p) {
    if (!p) return;
    g_cam.yaw = norm_yaw_deg(p->yaw);
    g_cam.pitch = clamp_pitch_deg(p->pitch);
    g_cam.pos.x = 0.0f;
    g_cam.pos.y = 0.0f;
    g_cam.pos.z = 0.0f;
    g_cam.ads_down = 0;
    g_ads_down = 0;
}

typedef enum {
    LOBBY_DEMO = 0,
    LOBBY_BATTLE,
    LOBBY_TDM,
    LOBBY_CTF,
    LOBBY_EVOLUTION,
    LOBBY_JOIN,
    LOBBY_COUNT
} LobbyAction;

char lobby_labels_mutable[LOBBY_COUNT][64];

static const char *LOBBY_LABELS[LOBBY_COUNT] = {
    "DEMO (SOLO)",
    "BATTLE (BOTS)",
    "TEAM DM (BOTS)",
    "LAN CTF",
    "EVOLUTION",
    "JOIN S.FARTHQ.COM"
};

static void lobby_init_labels() {
    for (int i = 0; i < LOBBY_COUNT; i++) {
        snprintf(lobby_labels_mutable[i], sizeof(lobby_labels_mutable[i]), "%s", LOBBY_LABELS[i]);
    }
}

static int lobby_menu_count() {
    if (ui_use_server && ui_state.entry_count > 0) {
        return ui_state.entry_count;
    }
    return LOBBY_COUNT;
}

static const char *lobby_menu_label(int idx) {
    if (ui_use_server && idx >= 0 && idx < ui_state.entry_count) {
        return ui_state.entries[idx].label;
    }
    return lobby_labels_mutable[idx];
}

static const char *lobby_menu_entry_id(int idx) {
    if (ui_use_server && idx >= 0 && idx < ui_state.entry_count) {
        return ui_state.entries[idx].id;
    }
    return NULL;
}

static void lobby_commit_edit(int index) {
    if (index < 0) return;
    if (ui_edit_len <= 0) return;
    ui_edit_buffer[ui_edit_len] = '\0';
    if (ui_use_server && index < ui_state.entry_count) {
        snprintf(ui_state.entries[index].label, UI_BRIDGE_LABEL_LEN, "%s", ui_edit_buffer);
    } else if (!ui_use_server && index < LOBBY_COUNT) {
        snprintf(lobby_labels_mutable[index], sizeof(lobby_labels_mutable[index]), "%s", ui_edit_buffer);
    }
}

static void lobby_start_edit(int index) {
    if (index < 0) return;
    ui_edit_index = index;
    ui_edit_len = 0;
    ui_edit_buffer[0] = '\0';
    SDL_StartTextInput();
}

static void lobby_end_edit(int commit) {
    if (commit) {
        lobby_commit_edit(ui_edit_index);
    }
    ui_edit_index = -1;
    ui_edit_len = 0;
    ui_edit_buffer[0] = '\0';
    SDL_StopTextInput();
}

static void lobby_apply_scene_id(const char *scene_id) {
    if (!scene_id || !scene_id[0]) return;
    if (strcmp(scene_id, "GARAGE_OSAKA") == 0) {
        scene_load(SCENE_GARAGE_OSAKA);
    } else if (strcmp(scene_id, "STADIUM") == 0) {
        scene_load(SCENE_STADIUM);
    } else if (strcmp(scene_id, "CITY") == 0) {
        scene_load(SCENE_CITY);
    }
}

static void lobby_apply_ui_state() {
    if (!ui_use_server) return;
    if (strcmp(ui_state.active_mode_id, "mode.join") == 0) {
        app_state = STATE_GAME_NET;
        reset_client_render_state_for_net();
        net_connect();
        return;
    }

    if (strcmp(ui_state.active_mode_id, "mode.demo") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(1, MODE_DEATHMATCH);
    } else if (strcmp(ui_state.active_mode_id, "mode.battle") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(12, MODE_DEATHMATCH);
    } else if (strcmp(ui_state.active_mode_id, "mode.tdm") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(12, MODE_TDM);
    } else if (strcmp(ui_state.active_mode_id, "mode.ctf") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(8, MODE_CTF);
    } else if (strcmp(ui_state.active_mode_id, "mode.evolution") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(8, MODE_EVOLUTION);
    } else if (strcmp(ui_state.active_mode_id, "mode.training") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(1, MODE_DEATHMATCH);
    } else if (strcmp(ui_state.active_mode_id, "mode.recorder") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(1, MODE_DEATHMATCH);
    } else if (strcmp(ui_state.active_mode_id, "mode.garage") == 0) {
        app_state = STATE_GAME_LOCAL;
        local_init_match(1, MODE_DEATHMATCH);
    } else {
        return;
    }

    lobby_apply_scene_id(ui_state.active_scene_id);
    if (local_state.scene_id == SCENE_GARAGE_OSAKA) {
        scene_load(SCENE_STADIUM);
    }
    progression_reset(SDL_GetTicks());
}

static void setup_lobby_2d() {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW);
    glLoadIdentity();
}

static void lobby_start_action(int action) {
    if (ui_use_server) {
        const char *entry_id = lobby_menu_entry_id(action);
        if (entry_id) {
            if (ui_bridge_send_intent_activate(entry_id, &ui_state)) {
                lobby_apply_ui_state();
                return;
            }
        }
    }
    if (action == LOBBY_JOIN) {
        app_state = STATE_GAME_NET;
        reset_client_render_state_for_net();
        net_connect();
    } else {
        app_state = STATE_GAME_LOCAL;
        switch (action) {
            case LOBBY_DEMO:
                local_init_match(1, MODE_DEATHMATCH);
                break;
            case LOBBY_BATTLE:
                local_init_match(12, MODE_DEATHMATCH);
                break;
            case LOBBY_TDM:
                local_init_match(12, MODE_TDM);
                break;
            case LOBBY_CTF:
                local_init_match(8, MODE_CTF);
                break;
            case LOBBY_EVOLUTION:
                local_init_match(8, MODE_EVOLUTION);
                break;
            default:
                break;
        }
    }

    if (app_state == STATE_GAME_LOCAL) {
        scene_load(SCENE_STADIUM);
        progression_reset(SDL_GetTicks());
    }

    if (app_state != STATE_LOBBY) {
        SDL_SetRelativeMouseMode(SDL_TRUE);
        glMatrixMode(GL_PROJECTION);
        glLoadIdentity();
        gluPerspective(75.0, 1280.0/720.0, 0.1, Z_FAR);
        glMatrixMode(GL_MODELVIEW);
        glEnable(GL_DEPTH_TEST);
    }
}

#define MAX_TRAILS 4096 
#define GRID_SIZE 50.0f
typedef struct { int cx, cz; float life; } Trail;
Trail trails[MAX_TRAILS];
int trail_head = 0;

void add_trail(int x, int z) {
    int prev = (trail_head - 1 + MAX_TRAILS) % MAX_TRAILS;
    if (trails[prev].cx == x && trails[prev].cz == z && trails[prev].life > 0.9f) return;
    trails[trail_head].cx = x; trails[trail_head].cz = z;
    trails[trail_head].life = 1.0f;
    trail_head = (trail_head + 1) % MAX_TRAILS;
}

void update_and_draw_trails() {
    glEnable(GL_BLEND); glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (p->active && p->on_ground && p->scene_id == local_state.scene_id) {
            int gx = (int)floorf(p->x / GRID_SIZE) * (int)GRID_SIZE + (int)(GRID_SIZE/2);
            int gz = (int)floorf(p->z / GRID_SIZE) * (int)GRID_SIZE + (int)(GRID_SIZE/2);
            add_trail(gx, gz);
        }
    }
    glLineWidth(2.0f);
    for(int i=0; i<MAX_TRAILS; i++) {
        if (trails[i].life > 0) {
            float s = (GRID_SIZE / 2.0f) - 4.0f;
            // Trails are now HOT PINK
            glColor4f(1.0f, 0.0f, 0.8f, trails[i].life); 
            glBegin(GL_LINE_LOOP);
            glVertex3f(trails[i].cx - s, 0.2f, trails[i].cz - s);
            glVertex3f(trails[i].cx + s, 0.2f, trails[i].cz - s);
            glVertex3f(trails[i].cx + s, 0.2f, trails[i].cz + s);
            glVertex3f(trails[i].cx - s, 0.2f, trails[i].cz + s);
            glEnd();
            trails[i].life -= 0.02f;
        }
    }
    glDisable(GL_BLEND);
}

void draw_grid() {
    glLineWidth(1.0f); glBegin(GL_LINES); 
    // THE MATRIX FLOOR (Cyan)
    glColor3f(0.0f, 1.0f, 1.0f); 
    for(int i=-4000; i<=4000; i+=50) { 
        glVertex3f(i, 0.1f, -4000); glVertex3f(i, 0.1f, 4000);
        glVertex3f(-4000, 0.1f, i); glVertex3f(4000, 0.1f, i); 
    }
    glEnd();
}

static void draw_box_primitive(float w, float h, float d) {
    glPushMatrix();
    glScalef(w, h, d);
    glBegin(GL_QUADS);
    glVertex3f(-0.5f, 0.5f, 0.5f); glVertex3f(0.5f, 0.5f, 0.5f); glVertex3f(0.5f, 0.5f, -0.5f); glVertex3f(-0.5f, 0.5f, -0.5f);
    glVertex3f(-0.5f, -0.5f, 0.5f); glVertex3f(0.5f, -0.5f, 0.5f); glVertex3f(0.5f, -0.5f, -0.5f); glVertex3f(-0.5f, -0.5f, -0.5f);
    glVertex3f(-0.5f, -0.5f, 0.5f); glVertex3f(0.5f, -0.5f, 0.5f); glVertex3f(0.5f, 0.5f, 0.5f); glVertex3f(-0.5f, 0.5f, 0.5f);
    glVertex3f(-0.5f, -0.5f, -0.5f); glVertex3f(0.5f, -0.5f, -0.5f); glVertex3f(0.5f, 0.5f, -0.5f); glVertex3f(-0.5f, 0.5f, -0.5f);
    glVertex3f(-0.5f, -0.5f, -0.5f); glVertex3f(-0.5f, -0.5f, 0.5f); glVertex3f(-0.5f, 0.5f, 0.5f); glVertex3f(-0.5f, 0.5f, -0.5f);
    glVertex3f(0.5f, -0.5f, 0.5f); glVertex3f(0.5f, -0.5f, -0.5f); glVertex3f(0.5f, 0.5f, -0.5f); glVertex3f(0.5f, 0.5f, 0.5f);
    glEnd();
    glPopMatrix();
}

static void draw_hydrants(void) {
    if (local_state.scene_id != SCENE_CITY) return;
    for (int i = 0; i < map_geo_props_count; i++) {
        Box h = map_geo_props[i];
        glPushMatrix();
        glTranslatef(h.x, 0.0f, h.z);

        glColor3f(0.85f, 0.08f, 0.08f);
        glPushMatrix(); glTranslatef(0.0f, 0.18f, 0.0f); draw_box_primitive(0.72f, 0.26f, 0.72f); glPopMatrix();

        glColor3f(0.93f, 0.15f, 0.13f);
        glPushMatrix(); glTranslatef(0.0f, 0.58f, 0.0f); draw_box_primitive(0.50f, 0.54f, 0.50f); glPopMatrix();

        glColor3f(1.0f, 0.35f, 0.35f);
        glPushMatrix(); glTranslatef(0.0f, 0.89f, 0.0f); draw_box_primitive(0.44f, 0.16f, 0.44f); glPopMatrix();

        glColor3f(0.8f, 0.08f, 0.08f);
        glPushMatrix(); glTranslatef(0.34f, 0.62f, 0.0f); draw_box_primitive(0.18f, 0.14f, 0.18f); glPopMatrix();
        glPushMatrix(); glTranslatef(-0.34f, 0.62f, 0.0f); draw_box_primitive(0.18f, 0.14f, 0.18f); glPopMatrix();

        glPopMatrix();
    }
}

// --- NEON BRUTALIST BLOCK RENDERER ---
void draw_map() {
    if (local_state.scene_id == SCENE_VOXWORLD) {
        const float step = 40.0f;
        for (float x = -1000.0f; x <= 1000.0f; x += step) {
            for (float z = -1000.0f; z <= 1000.0f; z += step) {
                float h00 = phys_vox_height_at(x, z);
                float h10 = phys_vox_height_at(x + step, z);
                float h11 = phys_vox_height_at(x + step, z + step);
                float h01 = phys_vox_height_at(x, z + step);

                glColor3f(0.12f + (h00 * 0.01f), 0.45f, 0.2f + (h00 * 0.005f));
                glBegin(GL_QUADS);
                glVertex3f(x, h00, z);
                glVertex3f(x + step, h10, z);
                glVertex3f(x + step, h11, z + step);
                glVertex3f(x, h01, z + step);
                glEnd();
            }
        }
        return;
    }

    // Enable blending for that glassy look if we wanted, but solid matte is cleaner for now
    // glDisable(GL_BLEND);

    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        
        // 1. PROCEDURAL NEON COLOR
        // Generate a color based on world position. 
        // This creates gradients across the city.
        float nr = 0.5f + 0.5f * sinf(b.x * 0.005f + b.y * 0.01f);
        float ng = 0.5f + 0.5f * sinf(b.z * 0.005f + 2.0f);
        float nb = 0.5f + 0.5f * sinf(b.x * 0.005f + 4.0f);
        
        // Boost brightness for the "Neon" effect
        if(nr > 0.8f) nr = 1.0f;
        if(ng > 0.8f) ng = 1.0f;
        if(nb > 0.8f) nb = 1.0f;

        glPushMatrix(); 
        glTranslatef(b.x, b.y, b.z); 
        glScalef(b.w, b.h, b.d);
        
        // 2. THE SOLID CORE (Deep Matte Black)
        // We render the faces dark so the edges pop.
        glBegin(GL_QUADS); 
        glColor3f(0.02f, 0.02f, 0.02f); // Almost black
        
        // Top
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        // Bottom
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,-0.5);
        // Front
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        // Back
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        // Left
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
        // Right
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
        glEnd();
        
        // 3. THE NEON CAGE (Wireframe)
        glLineWidth(2.0f); 
        glColor3f(nr, ng, nb); 
        
        // Using LINE_LOOP for top/bottom and LINES for pillars is efficient enough
        // Top Loop
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        
        // Bottom Loop
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, -0.5, 0.5); glVertex3f(0.5, -0.5, 0.5); glVertex3f(0.5, -0.5, -0.5); glVertex3f(-0.5, -0.5, -0.5);
        glEnd();
        
        // Vertical Pillars
        glBegin(GL_LINES);
        glVertex3f(-0.5, -0.5, 0.5); glVertex3f(-0.5, 0.5, 0.5);
        glVertex3f(0.5, -0.5, 0.5); glVertex3f(0.5, 0.5, 0.5);
        glVertex3f(0.5, -0.5, -0.5); glVertex3f(0.5, 0.5, -0.5);
        glVertex3f(-0.5, -0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();

        glPopMatrix();
    }

    draw_hydrants();
}

void draw_buggy_model() {
    // Chassis - Cyber Grey
    glColor3f(0.2f, 0.2f, 0.2f);
    glPushMatrix(); glScalef(2.0f, 1.0f, 3.5f); 
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1);
    glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd(); 
    
    // Neon Trim for Buggy
    glLineWidth(2.0f); glColor3f(1.0f, 0.0f, 0.0f); // Red Trim
    glBegin(GL_LINES);
    glVertex3f(-1,1,1); glVertex3f(1,1,1);
    glVertex3f(1,1,1); glVertex3f(1,1,-1);
    glVertex3f(1,1,-1); glVertex3f(-1,1,-1);
    glVertex3f(-1,1,-1); glVertex3f(-1,1,1);
    glEnd();
    
    glPopMatrix();
    
    // Wheels - Neon Blue Rims
    glColor3f(0.1f, 0.1f, 0.1f);
    float wx[] = {-2.2, 2.2, -2.2, 2.2};
    float wz[] = {2.5, 2.5, -2.5, -2.5};
    for(int i=0; i<4; i++) {
        glPushMatrix();
        glTranslatef(wx[i], -0.5f, wz[i]); glScalef(0.8f, 1.5f, 1.5f);
        glBegin(GL_QUADS); 
        glColor3f(0.1f, 0.1f, 0.1f); // Tire
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,-0.5,-0.5);
        glVertex3f(0.5,-0.5,-0.5);
        glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,-0.5,0.5);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5); glVertex3f(-0.5,-0.5,-0.5);
        glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glEnd(); 
        
        // Rim Line
        glLineWidth(2.0f); glColor3f(0.0f, 1.0f, 1.0f); 
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.51, 0.5, 0.5); glVertex3f(-0.51, -0.5, 0.5); glVertex3f(-0.51, -0.5, -0.5); glVertex3f(-0.51, 0.5, -0.5);
        glEnd();
        
        glPopMatrix();
    }
}

void draw_gun_model(int weapon_id) {
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); glScalef(0.05f, 0.05f, 0.8f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); glScalef(0.15f, 0.2f, 0.5f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); glScalef(0.1f, 0.15f, 1.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); glScalef(0.25f, 0.15f, 0.8f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); glScalef(0.08f, 0.12f, 2.0f); break;
    }
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1);
    glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
    
    // Wireframe Gun
    glLineWidth(1.0f); glColor3f(1.0f, 1.0f, 1.0f);
    glBegin(GL_LINES);
    glVertex3f(-1,1,1); glVertex3f(1,1,1);
    glVertex3f(1,1,1); glVertex3f(1,1,-1);
    glVertex3f(-1,1,1); glVertex3f(-1,-1,1);
    glEnd();
}

static void draw_box(float w, float h, float d) {
    glPushMatrix();
    glScalef(w, h, d);
    glBegin(GL_QUADS);
    // Front
    glVertex3f(-0.5f,-0.5f,0.5f); glVertex3f(0.5f,-0.5f,0.5f); glVertex3f(0.5f,0.5f,0.5f); glVertex3f(-0.5f,0.5f,0.5f);
    // Back
    glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(-0.5f,0.5f,-0.5f); glVertex3f(0.5f,0.5f,-0.5f); glVertex3f(0.5f,-0.5f,-0.5f);
    // Left
    glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(-0.5f,-0.5f,0.5f); glVertex3f(-0.5f,0.5f,0.5f); glVertex3f(-0.5f,0.5f,-0.5f);
    // Right
    glVertex3f(0.5f,-0.5f,-0.5f); glVertex3f(0.5f,0.5f,-0.5f); glVertex3f(0.5f,0.5f,0.5f); glVertex3f(0.5f,-0.5f,0.5f);
    // Top
    glVertex3f(-0.5f,0.5f,0.5f); glVertex3f(0.5f,0.5f,0.5f); glVertex3f(0.5f,0.5f,-0.5f); glVertex3f(-0.5f,0.5f,-0.5f);
    // Bottom
    glVertex3f(-0.5f,-0.5f,0.5f); glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(0.5f,-0.5f,-0.5f); glVertex3f(0.5f,-0.5f,0.5f);
    glEnd();
    glPopMatrix();
}

static void draw_box_outline(float w, float h, float d) {
    glPushMatrix();
    glScalef(w, h, d);
    glLineWidth(2.0f);
    glColor3f(1.0f, 1.0f, 0.0f);

    glBegin(GL_LINE_LOOP);
    glVertex3f(-0.5f, 0.5f, 0.5f); glVertex3f(0.5f, 0.5f, 0.5f);
    glVertex3f(0.5f, -0.5f, 0.5f); glVertex3f(-0.5f, -0.5f, 0.5f);
    glEnd();

    glBegin(GL_LINE_LOOP);
    glVertex3f(-0.5f, 0.5f, -0.5f); glVertex3f(0.5f, 0.5f, -0.5f);
    glVertex3f(0.5f, -0.5f, -0.5f); glVertex3f(-0.5f, -0.5f, -0.5f);
    glEnd();

    glBegin(GL_LINES);
    glVertex3f(-0.5f, 0.5f, 0.5f); glVertex3f(-0.5f, 0.5f, -0.5f);
    glVertex3f(0.5f, 0.5f, 0.5f); glVertex3f(0.5f, 0.5f, -0.5f);
    glVertex3f(0.5f, -0.5f, 0.5f); glVertex3f(0.5f, -0.5f, -0.5f);
    glVertex3f(-0.5f, -0.5f, 0.5f); glVertex3f(-0.5f, -0.5f, -0.5f);
    glEnd();

    glPopMatrix();
}

static void draw_ronin_shell(void) {
    // Jacket core (cropped waist, broad shoulders)
    glColor3f(0.1f, 0.1f, 0.1f); // Matte black
    glPushMatrix();
    glTranslatef(0.0f, 0.9f, 0.0f);
    draw_box(RONIN_TORSO_W, RONIN_TORSO_H, RONIN_TORSO_D);
    draw_box_outline(RONIN_TORSO_W, RONIN_TORSO_H, RONIN_TORSO_D);
    // Shoulder pads
    glPushMatrix(); glTranslatef(-RONIN_SHOULDER_PAD_OFFSET, 0.35f, 0.0f); draw_box(RONIN_SHOULDER_PAD_W, RONIN_SHOULDER_PAD_H, RONIN_SHOULDER_PAD_D); draw_box_outline(RONIN_SHOULDER_PAD_W, RONIN_SHOULDER_PAD_H, RONIN_SHOULDER_PAD_D); glPopMatrix();
    glPushMatrix(); glTranslatef(RONIN_SHOULDER_PAD_OFFSET, 0.35f, 0.0f); draw_box(RONIN_SHOULDER_PAD_W, RONIN_SHOULDER_PAD_H, RONIN_SHOULDER_PAD_D); draw_box_outline(RONIN_SHOULDER_PAD_W, RONIN_SHOULDER_PAD_H, RONIN_SHOULDER_PAD_D); glPopMatrix();
    // Sleeves
    glPushMatrix(); glTranslatef(-RONIN_SLEEVE_OFFSET, -0.25f, 0.0f); draw_box(RONIN_SLEEVE_W, RONIN_SLEEVE_H, RONIN_SLEEVE_D); draw_box_outline(RONIN_SLEEVE_W, RONIN_SLEEVE_H, RONIN_SLEEVE_D); glPopMatrix();
    glPushMatrix(); glTranslatef(RONIN_SLEEVE_OFFSET, -0.25f, 0.0f); draw_box(RONIN_SLEEVE_W, RONIN_SLEEVE_H, RONIN_SLEEVE_D); draw_box_outline(RONIN_SLEEVE_W, RONIN_SLEEVE_H, RONIN_SLEEVE_D); glPopMatrix();
    // Red satin lining at hem
    glColor3f(0.6f, 0.0f, 0.0f);
    glBegin(GL_QUADS);
    glVertex3f(-0.68f, RONIN_LINING_Y_BOTTOM, 0.39f); glVertex3f(0.68f, RONIN_LINING_Y_BOTTOM, 0.39f);
    glVertex3f(0.68f, RONIN_LINING_Y_TOP, 0.39f); glVertex3f(-0.68f, RONIN_LINING_Y_TOP, 0.39f);
    glEnd();
    glPopMatrix();

    // Tech cargo pants (baggy)
    glColor3f(0.18f, 0.18f, 0.2f); // Charcoal
    glPushMatrix(); glTranslatef(-RONIN_PANTS_OFFSET, 0.0f, 0.0f); draw_box(RONIN_PANTS_W, RONIN_PANTS_H, RONIN_PANTS_D); draw_box_outline(RONIN_PANTS_W, RONIN_PANTS_H, RONIN_PANTS_D); glPopMatrix();
    glPushMatrix(); glTranslatef(RONIN_PANTS_OFFSET, 0.0f, 0.0f); draw_box(RONIN_PANTS_W, RONIN_PANTS_H, RONIN_PANTS_D); draw_box_outline(RONIN_PANTS_W, RONIN_PANTS_H, RONIN_PANTS_D); glPopMatrix();
}

static void draw_storm_mask(void) {
    // Head base
    glColor3f(0.06f, 0.06f, 0.06f);
    draw_box(RONIN_HEAD_W, RONIN_HEAD_H, RONIN_HEAD_D);
    draw_box_outline(RONIN_HEAD_W, RONIN_HEAD_H, RONIN_HEAD_D);
    // Faceplate
    glColor3f(0.2f, 0.2f, 0.22f);
    glPushMatrix(); glTranslatef(0.0f, -0.05f, 0.37f); draw_box(RONIN_FACEPLATE_W, RONIN_FACEPLATE_H, RONIN_FACEPLATE_D); draw_box_outline(RONIN_FACEPLATE_W, RONIN_FACEPLATE_H, RONIN_FACEPLATE_D); glPopMatrix();
    // Cyan vents
    glColor3f(0.0f, 1.0f, 1.0f);
    glPushMatrix(); glTranslatef(RONIN_VENT_OFFSET_X, -0.08f, 0.42f); draw_box(RONIN_VENT_W, RONIN_VENT_H, RONIN_VENT_D); draw_box_outline(RONIN_VENT_W, RONIN_VENT_H, RONIN_VENT_D); glPopMatrix();
    glPushMatrix(); glTranslatef(-RONIN_VENT_OFFSET_X, -0.08f, 0.42f); draw_box(RONIN_VENT_W, RONIN_VENT_H, RONIN_VENT_D); draw_box_outline(RONIN_VENT_W, RONIN_VENT_H, RONIN_VENT_D); glPopMatrix();
    // Broken horn silhouette (single jagged horn)
    glColor3f(0.08f, 0.08f, 0.08f);
    glPushMatrix(); glTranslatef(RONIN_HORN_OFFSET_X, 0.52f, 0.05f); draw_box(RONIN_HORN_W, RONIN_HORN_H, RONIN_HORN_D); draw_box_outline(RONIN_HORN_W, RONIN_HORN_H, RONIN_HORN_D); glPopMatrix();
}

void draw_weapon_p(PlayerState *p) {
    if (p->in_vehicle) return; 
    glPushMatrix();
    glLoadIdentity();
    float kick = p->recoil_anim * 0.2f;
    float reload_dip = (p->reload_timer > 0) ? sinf(p->reload_timer * 0.2f) * 0.5f - 0.5f : 0.0f;
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    float bob = sinf(SDL_GetTicks() * 0.015f) * speed * 0.15f; 
    float x_offset = (current_fov < 50.0f) ? 0.25f : 0.4f;
    glTranslatef(x_offset, -0.5f + kick + reload_dip + (bob * 0.5f), -1.2f + (kick * 0.5f) + bob);
    glRotatef(-p->recoil_anim * 10.0f, 1, 0, 0);
    draw_gun_model(p->current_weapon);
    glPopMatrix();
}

void draw_head(int weapon_id) {
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); break;
    }
    glBegin(GL_QUADS);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(-0.4, 0, 0.4);
    glVertex3f(-0.4, 0.8, -0.4); glVertex3f(0.4, 0.8, -0.4);
    glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(-0.4, 0, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(-0.4, 0, 0.4);
    glVertex3f(-0.4, 0, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(0.4, 0.8, -0.4);
    glEnd();
}

void draw_player_3rd(PlayerState *p) {
    float draw_yaw = norm_yaw_deg(p->yaw);
    float draw_pitch = clamp_pitch_deg(p->pitch);
    float draw_recoil = (p->is_shooting > 0) ? 1.0f : p->recoil_anim;

    glPushMatrix();
    glTranslatef(p->x, p->y + 0.2f, p->z);
    glRotatef(draw_yaw, 0, 1, 0);
    if (p->in_vehicle) {
        draw_buggy_model();
    } else {
        float s = fabsf(sinf(p->run_phase));
        float amp = 0.10f * p->run_weight;
        float scale_y = 1.0f - amp * s;
        float scale_xz = 1.0f + amp * 0.6f * s;
        glScalef(scale_xz, scale_y, scale_xz);
        draw_ronin_shell();
        glPushMatrix();
        glTranslatef(0.0f, 1.85f, 0.0f);
        glRotatef(draw_pitch, 1, 0, 0);
        draw_storm_mask();
        glPopMatrix();
        glPushMatrix(); glTranslatef(0.6f, 1.1f, 0.55f);
        glRotatef(draw_pitch, 1, 0, 0);
        glRotatef(-draw_recoil * 10.0f, 1, 0, 0);
        glTranslatef(0.0f, 0.0f, -draw_recoil * 0.08f);
        glScalef(0.8f, 0.8f, 0.8f); draw_gun_model(p->current_weapon); glPopMatrix(); 
    }
    glPopMatrix();
}

// --- NEW HELPER: Wireframe Circle ---
void draw_circle(float x, float y, float r, int segments) {
    glBegin(GL_LINE_LOOP);
    for(int i=0; i<segments; i++) {
        float theta = 2.0f * 3.1415926f * (float)i / (float)segments;
        float cx = r * cosf(theta);
        float cy = r * sinf(theta);
        glVertex2f(x + cx, y + cy);
    }
    glEnd();
}

static void hud_log_push(const char *msg) {
    if (!msg || !msg[0]) return;
    snprintf(hud_log[g_hud_state.head], HUD_LOG_LINE_LEN, "%s", msg);
    hud_log_time[g_hud_state.head] = SDL_GetTicks();
    g_hud_state.head = (g_hud_state.head + 1) % HUD_LOG_LINES;
}

static void hud_log_draw(void) {
    float x0 = 18.0f, y0 = 180.0f;
    float w = 420.0f, h = 120.0f;

    glColor3f(0.05f, 0.06f, 0.08f);
    glRectf(x0, y0, x0 + w, y0 + h);

    glColor3f(0.25f, 0.8f, 1.0f);
    glBegin(GL_LINE_LOOP);
    glVertex2f(x0, y0);
    glVertex2f(x0 + w, y0);
    glVertex2f(x0 + w, y0 + h);
    glVertex2f(x0, y0 + h);
    glEnd();

    glColor3f(0.75f, 0.85f, 0.95f);
    draw_string("LOG", x0 + 10.0f, y0 + h - 16.0f, 5);

    float line_y = y0 + 14.0f;
    int idx = (g_hud_state.head - 1 + HUD_LOG_LINES) % HUD_LOG_LINES;

    for (int i = 0; i < HUD_LOG_LINES; i++) {
        const char *line = hud_log[idx];
        if (line[0]) {
            unsigned int age_ms = SDL_GetTicks() - hud_log_time[idx];
            float fade = 1.0f - ((float)age_ms / 15000.0f);
            if (fade < 0.45f) fade = 0.45f;
            glColor3f(0.85f * fade, 0.9f * fade, 0.95f * fade);
            draw_string(line, x0 + 10.0f, line_y, 4);
            line_y += 12.0f;
            if (line_y > y0 + h - 24.0f) break;
        }
        idx = (idx - 1 + HUD_LOG_LINES) % HUD_LOG_LINES;
    }
}

static void draw_huntsman_widget(void) {
    if (local_state.scene_id != SCENE_CITY || !city_huntsman.active) return;

    float x0 = 860.0f, y0 = 168.0f;
    float w = 395.0f, h = 148.0f;
    glColor3f(0.05f, 0.04f, 0.06f);
    glRectf(x0, y0, x0 + w, y0 + h);
    glColor3f(0.55f, 0.9f, 0.65f);
    glBegin(GL_LINE_LOOP);
    glVertex2f(x0, y0); glVertex2f(x0 + w, y0); glVertex2f(x0 + w, y0 + h); glVertex2f(x0, y0 + h);
    glEnd();

    int sacs_alive = HUNTSMAN_SACS - city_huntsman.sacs_destroyed;
    if (sacs_alive < 0) sacs_alive = 0;

    glColor3f(0.75f, 0.95f, 0.8f);
    draw_string("GREEN RONIN'S HUNTSMAN", x0 + 10.0f, y0 + h - 16.0f, 5);

    char line[128];
    snprintf(line, sizeof(line), "PHASE: %d", city_huntsman.phase);
    glColor3f(0.9f, 0.9f, 0.85f);
    draw_string(line, x0 + 10.0f, y0 + h - 34.0f, 4);

    snprintf(line, sizeof(line), "ANCHOR CLUSTERS: %d/2", city_huntsman.clusters_cleared);
    draw_string(line, x0 + 10.0f, y0 + h - 50.0f, 4);
    snprintf(line, sizeof(line), "SACS ALIVE: %d/3", sacs_alive);
    draw_string(line, x0 + 10.0f, y0 + h - 66.0f, 4);
    snprintf(line, sizeof(line), "SPIDERLINGS: %d", city_huntsman.spiderlings_alive);
    draw_string(line, x0 + 10.0f, y0 + h - 82.0f, 4);

    if (city_huntsman.vulnerable) {
        unsigned int now = SDL_GetTicks();
        unsigned int remain = (city_huntsman.vulnerable_until_ms > now) ? (city_huntsman.vulnerable_until_ms - now) : 0;
        snprintf(line, sizeof(line), "VULNERABLE: %u", remain / 1000);
        glColor3f(1.0f, 0.45f, 0.25f);
        draw_string(line, x0 + 10.0f, y0 + 14.0f, 5);
    } else {
        glColor3f(0.8f, 0.8f, 0.6f);
        draw_string("VULNERABLE: LOCKED", x0 + 10.0f, y0 + 14.0f, 4);
    }
}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    float reticle_cx = 640.0f + reticle_dx;
    float reticle_cy = 360.0f + reticle_dy;

    glColor3f(0, 1, 0);
    if (current_fov < 50.0f) {
        glBegin(GL_LINES);
        glVertex2f(0, reticle_cy); glVertex2f(1280, reticle_cy);
        glVertex2f(reticle_cx, 0); glVertex2f(reticle_cx, 720);
        glEnd();
    } else {
        glLineWidth(2.0f);
        glBegin(GL_LINES);
        glVertex2f(reticle_cx - 10.0f, reticle_cy); glVertex2f(reticle_cx + 10.0f, reticle_cy);
        glVertex2f(reticle_cx, reticle_cy - 10.0f); glVertex2f(reticle_cx, reticle_cy + 10.0f);
        glEnd();
    }

    // --- HIT INDICATORS ---
    if (p->hit_feedback > 0) {
        if (p->hit_feedback >= 25) glColor3f(1.0f, 0.0f, 0.0f); // RED (Kill/High Dmg)
        else glColor3f(0.0f, 1.0f, 0.0f); // GREEN (Normal)
        
        glLineWidth(2.0f);
        draw_circle(reticle_cx, reticle_cy, 20.0f, 16); // Hit Ring
        
        // DOUBLE RING FOR KILL
        if (p->hit_feedback >= 25) {
            draw_circle(reticle_cx, reticle_cy, 28.0f, 16); // Outer Kill Ring
        }
    }

    glColor3f(0.2f, 0, 0); glRectf(50, 50, 250, 70); glColor3f(1.0f, 0, 0);
    glRectf(50, 50, 50 + (p->health * 2), 70);
    glColor3f(0, 0, 0.2f); glRectf(50, 80, 250, 100); glColor3f(0.2f, 0.2f, 1.0f);
    glRectf(50, 80, 50 + (p->shield * 2), 100);
    
    if (p->in_vehicle) {
        glColor3f(0.0f, 1.0f, 0.0f);
        draw_string("BUGGY ONLINE", 50, 120, 12);
    }

    if (p->storm_charges > 0) {
        char storm_buf[32];
        sprintf(storm_buf, "STORM ARROWS: %d", p->storm_charges);
        glColor3f(1.0f, 0.2f, 0.2f);
        draw_string(storm_buf, 50, 140, 8);
    } else if (p->ability_cooldown == 0) {
        glColor3f(0.0f, 0.8f, 1.0f);
        draw_string("E: STORM ARROWS READY", 50, 140, 8);
    }
    
    float raw_speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    char vel_buf[32]; sprintf(vel_buf, "VEL: %.2f", raw_speed);
    glColor3f(1.0f, 1.0f, 0.0f); draw_string(vel_buf, 1100, 50, 8); 

    int wpn = p->current_weapon;
    if (wpn < 0 || wpn >= MAX_WEAPONS) wpn = WPN_KNIFE;
    int ammo = p->ammo[wpn];
    char ammo_buf[96];
    snprintf(ammo_buf, sizeof(ammo_buf), "%s AMMO: %d", weapon_name(wpn), ammo);
    glColor3f(0.95f, 0.95f, 0.95f);
    draw_string(ammo_buf, 960, 100, 8);

    if (p->scene_id == SCENE_CITY && (fabsf(p->x) > CITY_SOFT_X || fabsf(p->z) > CITY_SOFT_Z)) {
        glColor3f(1.0f, 0.55f, 0.2f);
        draw_string("LEAVING DISTRICT... TURN BACK", 430, 680, 7);
    }
    if (p->scene_id == SCENE_CITY && city_fields.initialized) {
        int district_count[5] = {0,0,0,0,0};
        int alive = 0;
        for (int i = 0; i < MAX_CITY_NPCS; i++) {
            if (!city_npcs.npcs[i].active) continue;
            alive++;
            int d = city_npcs.npcs[i].home_district;
            if (d >= 0 && d < 5) district_count[d]++;
        }
        float sm = city_fields_sample(p->x, p->z, city_fields.market);
        float se = city_fields_sample(p->x, p->z, city_fields.entropy);
        float ss = city_fields_sample(p->x, p->z, city_fields.security);
        char city_dbg[192];
        snprintf(city_dbg, sizeof(city_dbg), "CITY NPC:%d D0:%d D1:%d D2:%d D3:%d D4:%d M%.2f E%.2f S%.2f", alive, district_count[0], district_count[1], district_count[2], district_count[3], district_count[4], sm, se, ss);
        glColor3f(0.55f, 1.0f, 0.75f);
        draw_string(city_dbg, 260, 24, 5);
        char city_agents[96];
        snprintf(city_agents, sizeof(city_agents), "BOIDS:%d GOB:%d FOX:%d", CITY_BOIDS, CITY_GOBLIN_AGENTS, CITY_FOX_AGENTS);
        glColor3f(0.45f, 0.85f, 1.0f);
        draw_string(city_agents, 560, 46, 5);
        int is_dragon_heat = (SDL_GetTicks() < city_fields.dragon_until_ms);
        if (is_dragon_heat && !g_hud_state.was_dragon_heat) {
            hud_log_push("OMENS: DRAGON HEAT");
        }
        g_hud_state.was_dragon_heat = is_dragon_heat;
    }
    if (p->scene_id == SCENE_CITY && city_huntsman.active) {
        if (city_huntsman.spiderlings_alive > g_hud_state.was_huntsman_spiderlings) {
            hud_log_push("Egg sac hatched spiderlings");
        }
        g_hud_state.was_huntsman_spiderlings = city_huntsman.spiderlings_alive;
    } else {
        g_hud_state.was_huntsman_spiderlings = 0;
    }

    if (app_state == STATE_GAME_LOCAL) {
        char lvl_buf[64];
        snprintf(lvl_buf, sizeof(lvl_buf), "LVL %d  XP %d/%d  PTS %d", match_prog.level, match_prog.xp, xp_for_next_level(match_prog.level), match_prog.unspent_points);
        glColor3f(0.9f, 0.9f, 0.3f);
        draw_string(lvl_buf, 40, 690, 6);

        char ability_buf[192];
        snprintf(ability_buf, sizeof(ability_buf), "F1 MOB %d  F2 VIT %d  F3 HAND %d  F4 SHLD %d  F5 STORM %d", 
                 match_prog.ability[0], match_prog.ability[1], match_prog.ability[2], match_prog.ability[3], match_prog.ability[4]);
        glColor3f(0.5f, 0.9f, 0.9f);
        draw_string(ability_buf, 40, 665, 5);
        glColor3f(0.9f, 0.8f, 0.5f);
        draw_string(match_prog.camera_third_person ? "V: 1ST PERSON" : "V: 3RD PERSON", 1030, 665, 5);

        char cam_dbg[96];
        snprintf(cam_dbg, sizeof(cam_dbg), "CAM:%s ADS:%d YAW:%0.1f", (g_cam.mode == CAM_THIRD) ? "3P" : "1P", g_cam.ads_down, norm_yaw_deg(g_cam.yaw));
        glColor3f(0.7f, 0.9f, 0.4f);
        draw_string(cam_dbg, 900, 640, 5);

        unsigned int now = SDL_GetTicks();
        unsigned int remain = (match_prog.match_end_ms > now) ? (match_prog.match_end_ms - now) : 0;
        char time_buf[48];
        snprintf(time_buf, sizeof(time_buf), "MATCH %02u:%02u", (remain / 1000) / 60, (remain / 1000) % 60);
        glColor3f(1.0f, 0.7f, 0.3f);
        draw_string(time_buf, 1090, 690, 6);

        glColor3f(0.7f, 0.85f, 1.0f);
        draw_string("O: SETTINGS", 1080, 640, 5);
    }

    hud_log_draw();
    draw_huntsman_widget();

    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

static int target_in_view(PlayerState *p, float tx, float ty, float tz, float max_dist, float min_dot) {
    AimRay ray = get_aim_ray(&g_cam, p);

    float dx = tx - ray.origin.x;
    float dy = ty - ray.origin.y;
    float dz = tz - ray.origin.z;
    float dist = sqrtf(dx * dx + dy * dy + dz * dz);
    if (dist > max_dist || dist <= 0.001f) return 0;
    dx /= dist; dy /= dist; dz /= dist;
    float dot = ray.dir.x * dx + ray.dir.y * dy + ray.dir.z * dz;
    return dot >= min_dot;
}

static void draw_scene_portals() {
    if (!scene_portal_active(local_state.scene_id)) return;
    PortalDef portals[4];
    int portal_count = scene_portals(local_state.scene_id, portals, 4);
    for (int i = 0; i < portal_count; i++) {
        glPushMatrix();
        glTranslatef(portals[i].x, portals[i].y, portals[i].z);
        if (portals[i].portal_id == PORTAL_ID_CITY_GATE) {
            glColor3f(0.9f, 0.5f, 1.0f);
        } else {
            glColor3f(0.2f, 0.9f, 1.0f);
        }
        glLineWidth(3.0f);
        glBegin(GL_LINE_LOOP);
        glVertex3f(-portals[i].radius, -2.0f, 0.0f);
        glVertex3f(portals[i].radius, -2.0f, 0.0f);
        glVertex3f(portals[i].radius, 6.0f, 0.0f);
        glVertex3f(-portals[i].radius, 6.0f, 0.0f);
        glEnd();
        glPopMatrix();
    }
}

static void draw_garage_vehicle_pads() {
    int pad_count = 0;
    const VehiclePad *pads = scene_vehicle_pads(local_state.scene_id, &pad_count);
    if (!pads || pad_count == 0) return;
    for (int i = 0; i < pad_count; i++) {
        glPushMatrix();
        glTranslatef(pads[i].x, pads[i].y + 0.1f, pads[i].z);
        glColor3f(0.4f, 0.8f, 0.4f);
        glBegin(GL_LINE_LOOP);
        glVertex3f(-3.0f, 0.0f, -3.0f);
        glVertex3f(3.0f, 0.0f, -3.0f);
        glVertex3f(3.0f, 0.0f, 3.0f);
        glVertex3f(-3.0f, 0.0f, 3.0f);
        glEnd();
        glPopMatrix();
    }
}

static void draw_garage_overlay(PlayerState *p) {
    if (local_state.scene_id != SCENE_GARAGE_OSAKA) return;

    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity();
    glOrtho(0, 1280, 0, 720, -1, 1);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();

    glColor3f(0.2f, 1.0f, 1.0f);
    draw_string("OSAKA GARAGE", 40, 670, 10);
    glColor3f(0.9f, 0.9f, 0.9f);
    draw_string("PORTAL A -> STADIUM", 40, 640, 6);
    draw_string("PORTAL B -> CITY", 40, 620, 6);

    int pad_count = 0;
    const VehiclePad *pads = scene_vehicle_pads(local_state.scene_id, &pad_count);
    float list_y = 600.0f;
    for (int i = 0; i < pad_count; i++) {
        int occupied = 0;
        for (int j = 0; j < MAX_CLIENTS; j++) {
            PlayerState *other = &local_state.players[j];
            if (!other->active) continue;
            float dx = other->x - pads[i].x;
            float dz = other->z - pads[i].z;
            if (other->in_vehicle && (dx * dx + dz * dz) < 16.0f) {
                occupied = 1;
                break;
            }
        }
        char line[128];
        snprintf(line, sizeof(line), "%s [%s]", pads[i].label, occupied ? "OCCUPIED" : "IDLE");
        glColor3f(occupied ? 1.0f : 0.5f, occupied ? 0.6f : 0.9f, 0.6f);
        draw_string(line, 40, list_y, 6);
        list_y -= 20.0f;
    }

    int portal_target = 0;
    PortalDef portals[4];
    int portal_count = scene_portals(local_state.scene_id, portals, 4);
    for (int i = 0; i < portal_count; i++) {
        if (target_in_view(p, portals[i].x, portals[i].y, portals[i].z, 30.0f, 0.75f)) {
            portal_target = 1;
            break;
        }
    }
    int pad_target = 0;
    if (scene_near_vehicle_pad(local_state.scene_id, p->x, p->z, 12.0f, NULL)) {
        int pad_idx = -1;
        if (scene_near_vehicle_pad(local_state.scene_id, p->x, p->z, 12.0f, &pad_idx)) {
            pad_target = target_in_view(p, pads[pad_idx].x, pads[pad_idx].y + 1.0f, pads[pad_idx].z, 20.0f, 0.7f);
        }
    }

    glColor3f(1.0f, 1.0f, 0.0f);
    if (portal_target) {
        draw_string("PRESS F TO TRAVEL", 540, 350, 8);
    } else if (pad_target) {
        draw_string(p->in_vehicle ? "EXIT VEHICLE" : "ENTER VEHICLE", 560, 350, 8);
    }

    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPopMatrix();
    glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_projectiles() {
    glDisable(GL_TEXTURE_2D);
    glPointSize(6.0f);
    glBegin(GL_POINTS);
    for (int i = 0; i < MAX_PROJECTILES; i++) {
        Projectile *p = &local_state.projectiles[i];
        if (!p->active) continue;
        if (p->scene_id != local_state.scene_id) continue;
        if (p->bounces_left > 0) glColor3f(1.0f, 0.4f, 0.1f);
        else glColor3f(1.0f, 0.0f, 0.0f);
        glVertex3f(p->x, p->y, p->z);
    }
    glEnd();
}

static void draw_city_npc_primitive(const CityNpc *n) {
    float r = 0.5f, g = 0.5f, b = 0.5f;
    float body_w = 1.4f, body_h = 3.2f, body_d = 1.4f;
    float head_w = 1.0f, head_h = 1.0f, head_d = 1.0f;

    switch (n->type) {
        case ENT_PEASANT: r = 0.6f; g = 0.7f; b = 0.45f; body_w = 1.2f; body_h = 3.0f; break;
        case ENT_GUARD: r = 0.3f; g = 0.45f; b = 0.85f; body_w = 1.4f; body_h = 3.4f; break;
        case ENT_HUNTER: r = 0.35f; g = 0.55f; b = 0.3f; body_w = 1.2f; body_h = 3.2f; break;
        case ENT_CULTIST: r = 0.65f; g = 0.15f; b = 0.75f; body_w = 1.1f; body_h = 3.6f; head_h = 1.3f; break;
        case ENT_GOBLIN: r = 0.2f; g = 0.95f; b = 0.2f; body_w = 1.5f; body_h = 2.2f; head_w = 0.9f; break;
        case ENT_ORC: r = 0.3f; g = 0.6f; b = 0.25f; body_w = 1.9f; body_h = 4.2f; head_w = 1.2f; head_h = 1.2f; break;
        case ENT_PILLAGER_MARAUDER: r = 0.85f; g = 0.3f; b = 0.25f; body_w = 1.5f; body_h = 3.5f; break;
        case ENT_PILLAGER_DESTROYER: r = 0.75f; g = 0.2f; b = 0.2f; body_w = 2.0f; body_h = 4.0f; head_w = 1.4f; break;
        case ENT_PILLAGER_CORRUPTOR: r = 0.75f; g = 0.1f; b = 0.55f; body_w = 1.4f; body_h = 3.6f; break;
        case ENT_DRAGON: r = 0.1f; g = 0.85f; b = 0.25f; body_w = 6.0f; body_h = 9.5f; body_d = 10.0f; head_w = 4.4f; head_h = 3.2f; head_d = 6.2f; break;
    }

    glPushMatrix();
    glTranslatef(n->x, n->y, n->z);
    glRotatef(n->yaw, 0.0f, 1.0f, 0.0f);
    glColor3f(r, g, b);
    glPushMatrix(); glTranslatef(0.0f, body_h * 0.5f, 0.0f); draw_box(body_w, body_h, body_d); draw_box_outline(body_w, body_h, body_d); glPopMatrix();
    glPushMatrix(); glTranslatef(0.0f, body_h + head_h * 0.5f, 0.0f); draw_box(head_w, head_h, head_d); draw_box_outline(head_w, head_h, head_d); glPopMatrix();
    if (n->type == ENT_GUARD) {
        glPushMatrix(); glTranslatef(-0.9f, body_h * 0.85f, 0.0f); draw_box(0.6f, 0.7f, 1.4f); draw_box_outline(0.6f, 0.7f, 1.4f); glPopMatrix();
        glPushMatrix(); glTranslatef(0.9f, body_h * 0.85f, 0.0f); draw_box(0.6f, 0.7f, 1.4f); draw_box_outline(0.6f, 0.7f, 1.4f); glPopMatrix();
    } else if (n->type == ENT_CULTIST) {
        glPushMatrix(); glTranslatef(0.0f, body_h + 0.9f, 0.45f); draw_box(0.35f, 0.45f, 0.35f); draw_box_outline(0.35f, 0.45f, 0.35f); glPopMatrix();
    } else if (n->type == ENT_DRAGON) {
        glPushMatrix(); glTranslatef(0.0f, body_h * 0.8f, -4.8f); draw_box(2.2f, 1.8f, 4.5f); draw_box_outline(2.2f, 1.8f, 4.5f); glPopMatrix();
        glPushMatrix(); glTranslatef(-4.5f, body_h * 0.7f, -1.2f); draw_box(3.8f, 0.8f, 2.2f); draw_box_outline(3.8f, 0.8f, 2.2f); glPopMatrix();
        glPushMatrix(); glTranslatef(4.5f, body_h * 0.7f, -1.2f); draw_box(3.8f, 0.8f, 2.2f); draw_box_outline(3.8f, 0.8f, 2.2f); glPopMatrix();
    }
    glPopMatrix();
}

static void draw_city_boids(void) {
    if (local_state.scene_id != SCENE_CITY || !city_fields.initialized) return;
    glDisable(GL_TEXTURE_2D);
    glColor3f(0.7f, 0.9f, 1.0f);
    glPointSize(3.0f);
    glBegin(GL_POINTS);
    for (int i = 0; i < CITY_BOIDS; i++) {
        glVertex3f(city_fields.boid_x[i], 80.0f + sinf((float)i) * 16.0f, city_fields.boid_z[i]);
    }
    glEnd();
}

static void draw_huntsman_arena_fx(void) {
    if (local_state.scene_id != SCENE_CITY || !city_huntsman.active) return;

    glLineWidth(2.0f);
    glColor3f(0.6f, 0.8f, 0.7f);
    for (int i = 0; i < HUNTSMAN_ANCHORS; i++) {
        if (city_huntsman.anchors_hp[i] <= 0) continue;
        glPushMatrix();
        glTranslatef(HUNTSMAN_ANCHOR_POS[i][0], 42.0f, HUNTSMAN_ANCHOR_POS[i][1]);
        draw_box(3.0f, 3.0f, 3.0f);
        draw_box_outline(3.0f, 3.0f, 3.0f);
        glPopMatrix();
    }

    glColor3f(0.8f, 0.85f, 0.9f);
    glBegin(GL_LINES);
    for (int c = 0; c < 4; c++) {
        int a = c * 2;
        int b = a + 1;
        if (city_huntsman.anchors_hp[a] <= 0 || city_huntsman.anchors_hp[b] <= 0) continue;
        glVertex3f(HUNTSMAN_ANCHOR_POS[a][0], 42.0f, HUNTSMAN_ANCHOR_POS[a][1]);
        glVertex3f(HUNTSMAN_ANCHOR_POS[b][0], 42.0f, HUNTSMAN_ANCHOR_POS[b][1]);
    }
    glEnd();

    glColor3f(0.9f, 0.95f, 0.8f);
    for (int i = 0; i < HUNTSMAN_SACS; i++) {
        if (city_huntsman.sacs_hp[i] <= 0) continue;
        glPushMatrix();
        glTranslatef(HUNTSMAN_SAC_POS[i][0], 6.0f, HUNTSMAN_SAC_POS[i][1]);
        draw_box(5.0f, 7.0f, 5.0f);
        draw_box_outline(5.0f, 7.0f, 5.0f);
        glPopMatrix();
    }
}

static void draw_city_npcs(void) {
    if (local_state.scene_id != SCENE_CITY && local_state.scene_id != SCENE_STADIUM) return;
    draw_huntsman_arena_fx();
    for (int i = 0; i < MAX_CITY_NPCS; i++) {
        if (!city_npcs.npcs[i].active) continue;
        draw_city_npc_primitive(&city_npcs.npcs[i]);
    }
}

static void client_apply_scene_id(int scene_id, unsigned int now_ms) {
    if (scene_id < 0) return;
    if (local_state.scene_id != scene_id) {
        local_state.scene_id = scene_id;
        phys_set_scene(scene_id);
        travel_overlay_until_ms = now_ms + 500;
        for (int i = 0; i < MAX_PROJECTILES; i++) {
            local_state.projectiles[i].active = 0;
        }
    }
}

static void draw_travel_overlay() {
    unsigned int now_ms = SDL_GetTicks();
    if (travel_overlay_until_ms <= now_ms) return;
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity();
    glOrtho(0, 1280, 0, 720, -1, 1);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();

    glColor3f(0.9f, 0.9f, 0.2f);
    draw_string("TRAVELING...", 520, 360, 8);

    glMatrixMode(GL_PROJECTION); glPopMatrix();
    glMatrixMode(GL_MODELVIEW); glPopMatrix();
    glEnable(GL_DEPTH_TEST);
}

static void draw_settings_overlay(void) {
    if (!g_show_settings) return;

    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity();
    glOrtho(0, 1280, 0, 720, -1, 1);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();

    glColor3f(0.05f, 0.08f, 0.12f);
    glRectf(280.0f, 170.0f, 1000.0f, 580.0f);
    glColor3f(0.2f, 0.8f, 1.0f);
    glBegin(GL_LINE_LOOP);
    glVertex2f(280.0f, 170.0f);
    glVertex2f(1000.0f, 170.0f);
    glVertex2f(1000.0f, 580.0f);
    glVertex2f(280.0f, 580.0f);
    glEnd();

    settings_load_if_needed();

    glColor3f(0.9f, 0.95f, 1.0f);
    draw_string("SETTINGS", 560.0f, 540.0f, 9.0f);
    draw_string("----------------", 500.0f, 515.0f, 5.0f);

    if (g_settings_selection == 0) {
        glColor3f(0.95f, 0.9f, 0.25f);
        draw_string(">", 340.0f, 450.0f, 8.0f);
    }

    if (g_server_force_shell_casings == 0) {
        glColor3f(1.0f, 0.5f, 0.5f);
        draw_string("Ammo shell casings: FORCED OFF (server)", 380.0f, 450.0f, 6.0f);
    } else if (g_server_force_shell_casings == 1) {
        glColor3f(1.0f, 0.75f, 0.5f);
        draw_string("Ammo shell casings: FORCED ON (server)", 380.0f, 450.0f, 6.0f);
    } else {
        glColor3f(0.85f, 0.95f, 0.85f);
        draw_string(shell_casings_enabled() ? "Ammo shell casings: ON" : "Ammo shell casings: OFF", 380.0f, 450.0f, 6.0f);
    }

    glColor3f(0.75f, 0.85f, 0.95f);
    draw_string("[W/S] Select  [Enter/Space] Toggle  [O/Esc] Back", 330.0f, 255.0f, 5.0f);

    glMatrixMode(GL_PROJECTION); glPopMatrix();
    glMatrixMode(GL_MODELVIEW); glPopMatrix();
    glEnable(GL_DEPTH_TEST);
}

void draw_scene(PlayerState *render_p, float dt) {
    glClearColor(0.02f, 0.02f, 0.05f, 1.0f); // DEEP SPACE BLUE BG
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT); glLoadIdentity();
    local_state.scene_id = render_p->scene_id;
    phys_set_scene(render_p->scene_id);

    camera_update(&g_cam, render_p, dt);
    CameraVec3 look = forward_from_yaw_pitch(g_cam.yaw, g_cam.pitch);

    if (g_cam.mode == CAM_THIRD) {
        CameraVec3 right = right_from_yaw(g_cam.yaw);
        float aim_x = render_p->x - right.x * 0.35f;
        float aim_y = render_p->y + g_cam.height;
        float aim_z = render_p->z - right.z * 0.35f;
        gluLookAt(g_cam.pos.x, g_cam.pos.y, g_cam.pos.z,
                  aim_x, aim_y, aim_z,
                  0.0f, 1.0f, 0.0f);
    } else {
        float eye_y = render_p->in_vehicle ? 8.0f : (render_p->crouching ? 2.5f : EYE_HEIGHT);
        float eye_x = render_p->x;
        float eye_z = render_p->z;
        gluLookAt(eye_x, render_p->y + eye_y, eye_z,
                  eye_x + look.x, render_p->y + eye_y + look.y, eye_z + look.z,
                  0.0f, 1.0f, 0.0f);
    }
    
    draw_grid(); 
    update_and_draw_trails();
    draw_map();
    draw_garage_vehicle_pads();
    draw_scene_portals();
    draw_projectiles();
    draw_city_boids();
    draw_city_npcs();
    if (render_p->in_vehicle) draw_player_3rd(render_p);
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active || p->scene_id != render_p->scene_id) continue;
        if (p == render_p) continue;
        draw_player_3rd(p);
    }
    if (g_cam.mode != CAM_THIRD) {
        draw_weapon_p(render_p);
    } else {
        PlayerState tmp = *render_p;
        AimRay aim_ray = get_aim_ray(&g_cam, render_p);
        yaw_pitch_from_dir(aim_ray.dir, &tmp.yaw, &tmp.pitch);
        draw_player_3rd(&tmp);
    }
    draw_hud(render_p); draw_garage_overlay(render_p);
    draw_travel_overlay();
    draw_settings_overlay();
}



static void draw_results_screen(void) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
    glClearColor(0.01f, 0.02f, 0.04f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT);

    glColor3f(0.9f, 0.9f, 0.2f);
    draw_string("MATCH COMPLETE", 470, 610, 10);

    PlayerState *p = &local_state.players[0];
    char line[128];
    glColor3f(0.5f, 0.9f, 0.9f);
    snprintf(line, sizeof(line), "KILLS: %d   DEATHS: %d", p->kills, p->deaths);
    draw_string(line, 470, 540, 7);
    snprintf(line, sizeof(line), "LEVEL REACHED: %d", match_prog.level);
    draw_string(line, 470, 505, 7);
    snprintf(line, sizeof(line), "ABILITIES  MOB:%d VIT:%d HAND:%d SHLD:%d STORM:%d", match_prog.ability[0], match_prog.ability[1], match_prog.ability[2], match_prog.ability[3], match_prog.ability[4]);
    draw_string(line, 300, 465, 6);

    glColor3f(0.9f, 0.6f, 0.4f);
    draw_string("ENTER: RETURN TO LOBBY", 450, 380, 7);
}

static void draw_lobby_buttons(int menu_count, float base_x, float base_y, float gap, float size) {
    const float colors[][3] = {
        {0.1f, 0.45f, 0.95f}, // blue
        {0.75f, 0.2f, 0.75f}, // magenta
        {0.0f, 0.75f, 0.75f}, // teal
        {0.2f, 0.6f, 0.6f},   // light teal
        {0.0f, 0.4f, 0.45f},  // dark teal
        {0.4f, 0.5f, 0.1f},   // olive
        {0.85f, 0.2f, 0.2f},  // red
        {0.9f, 0.75f, 0.1f},  // yellow
        {0.2f, 0.8f, 0.2f}    // green
    };
    int color_count = (int)(sizeof(colors) / sizeof(colors[0]));

    for (int i = 0; i < menu_count; i++) {
        float y = base_y - (gap * i);
        const float *c = colors[i % color_count];
        glColor3f(c[0], c[1], c[2]);
        glRectf(base_x, y, base_x + size, y + size);

        if (i == lobby_selection) {
            glColor3f(0.05f, 0.05f, 0.05f);
            glLineWidth(2.0f);
            glBegin(GL_LINE_LOOP);
            glVertex2f(base_x, y);
            glVertex2f(base_x + size, y);
            glVertex2f(base_x + size, y + size);
            glVertex2f(base_x, y + size);
            glEnd();
        }

        glColor3f(0.05f, 0.05f, 0.05f);
        draw_string(lobby_menu_label(i), base_x + 12.0f, y + size * 0.55f, 5);
        if (ui_edit_index == i) {
            glColor3f(0.95f, 0.9f, 0.2f);
            draw_string(ui_edit_buffer, base_x + 12.0f, y + size * 0.25f, 5);
        }
    }
}

static int lobby_hit_test(float mx, float my, int menu_count, float base_x, float base_y, float gap, float size) {
    for (int i = 0; i < menu_count; i++) {
        float y = base_y - (gap * i);
        if (mx >= base_x && mx <= base_x + size && my >= y && my <= y + size) {
            return i;
        }
    }
    return -1;
}

void net_init() {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    #ifdef _WIN32
    u_long mode = 1;
    ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif
}

void net_shutdown() {
    if (sock >= 0) {
        char buffer[128];
        NetHeader *h = (NetHeader*)buffer;
        memset(h, 0, sizeof(NetHeader));
        h->type = PACKET_DISCONNECT;
        sendto(sock, buffer, sizeof(NetHeader), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));

        #ifdef _WIN32
        closesocket(sock);
        #else
        close(sock);
        #endif
        sock = -1;
    }
    reset_client_render_state_for_net();
}

void net_connect() {
    reset_client_render_state_for_net();
    if (sock < 0) net_init();
    struct hostent *he = gethostbyname(SERVER_HOST);
    if (he) {
        server_addr.sin_family = AF_INET; 
        server_addr.sin_port = htons(SERVER_PORT); 
        memcpy(&server_addr.sin_addr, he->h_addr_list[0], he->h_length);
        char buffer[128];
        NetHeader *h = (NetHeader*)buffer;
        memset(h, 0, sizeof(NetHeader));
        h->type = PACKET_CONNECT;
        h->scene_id = 0;
        sendto(sock, buffer, sizeof(NetHeader), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
        printf("Connected to %s...\n", SERVER_HOST);
    } else {
        printf("Failed to resolve %s\n", SERVER_HOST);
    }
}

UserCmd client_create_cmd(float fwd, float str, float yaw, float pitch, int shoot, int jump, int crouch, int reload, int use, int bike, int ability, int wpn_idx) {
    UserCmd cmd;
    memset(&cmd, 0, sizeof(UserCmd));
    cmd.sequence = ++net_cmd_seq; cmd.timestamp = SDL_GetTicks();
    cmd.yaw = yaw; cmd.pitch = pitch;
    cmd.fwd = fwd; cmd.str = str;
    if(shoot) cmd.buttons |= BTN_ATTACK; if(jump) cmd.buttons |= BTN_JUMP;
    if(crouch) cmd.buttons |= BTN_CROUCH;
    if(reload) cmd.buttons |= BTN_RELOAD;
    if(use) cmd.buttons |= BTN_USE;
    if(bike) cmd.buttons |= BTN_VEHICLE_2;
    if(ability) cmd.buttons |= BTN_ABILITY_1;
    cmd.weapon_idx = wpn_idx; return cmd;
}

void net_send_cmd(UserCmd cmd) {
    char packet_data[256];
    int cursor = 0;

    for (int i = NET_CMD_HISTORY - 1; i > 0; i--) {
        net_cmd_history[i] = net_cmd_history[i - 1];
    }
    net_cmd_history[0] = cmd;
    if (net_cmd_history_count < NET_CMD_HISTORY) net_cmd_history_count++;

    NetHeader head; head.type = PACKET_USERCMD;
    head.client_id = 0;
    head.scene_id = 0;
    memcpy(packet_data + cursor, &head, sizeof(NetHeader)); cursor += sizeof(NetHeader);

    unsigned char count = (unsigned char)net_cmd_history_count;
    memcpy(packet_data + cursor, &count, 1); cursor += 1;

    for (int i = 0; i < net_cmd_history_count; i++) {
        memcpy(packet_data + cursor, &net_cmd_history[i], sizeof(UserCmd));
        cursor += sizeof(UserCmd);
    }

    sendto(sock, packet_data, cursor, 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
}

void net_process_snapshot(char *buffer, int len) {
    int cursor = sizeof(NetHeader);
    unsigned char count = *(unsigned char*)(buffer + cursor); cursor++;
    
    for(int i=0; i<count; i++) {
        NetPlayer *np = (NetPlayer*)(buffer + cursor);
        cursor += sizeof(NetPlayer);
        
        int id = np->id;
        if (id > 0 && id < MAX_CLIENTS) {
            PlayerState *p = &local_state.players[id];
            p->active = 1;
            p->scene_id = np->scene_id;
            p->x = np->x; p->y = np->y; p->z = np->z;
            p->yaw = norm_yaw_deg(np->yaw); p->pitch = clamp_pitch_deg(np->pitch);
            p->health = np->health;
            p->current_weapon = np->current_weapon;
            p->ammo[p->current_weapon] = np->ammo;
            p->is_shooting = np->is_shooting;
            p->in_vehicle = np->in_vehicle;
            p->storm_charges = np->storm_charges;
            
            // --- SYNC HIT MARKER ---
            p->hit_feedback = np->hit_feedback;

            if (p->is_shooting) p->recoil_anim = 1.0f;
        } else if (id == 0) {
            local_state.players[0].scene_id = np->scene_id;
            local_state.players[0].yaw = norm_yaw_deg(np->yaw);
            local_state.players[0].pitch = clamp_pitch_deg(np->pitch);
            local_state.players[0].current_weapon = np->current_weapon;
            local_state.players[0].ammo[local_state.players[0].current_weapon] = np->ammo;
            local_state.players[0].in_vehicle = np->in_vehicle; 
            local_state.players[0].storm_charges = np->storm_charges;
            local_state.players[0].hit_feedback = np->hit_feedback;
        }
    }

    int render_id = (my_client_id > 0 && my_client_id < MAX_CLIENTS && local_state.players[my_client_id].active)
        ? my_client_id
        : 0;
    client_apply_scene_id(local_state.players[render_id].scene_id, SDL_GetTicks());
}

void net_tick() {
    char buffer[4096];
    struct sockaddr_in sender;
    socklen_t slen = sizeof(sender);
    int len = recvfrom(sock, buffer, 4096, 0, (struct sockaddr*)&sender, &slen);
    while (len > 0) {
        NetHeader *head = (NetHeader*)buffer;
        if (head->type == PACKET_SNAPSHOT) {
            net_process_snapshot(buffer, len);
        }
        if (head->type == PACKET_WELCOME) {
            my_client_id = head->client_id;
            if (my_client_id > 0 && my_client_id < MAX_CLIENTS) {
                local_state.players[my_client_id].active = 1;
                local_state.players[my_client_id].scene_id = head->scene_id;
                client_apply_scene_id(head->scene_id, SDL_GetTicks());
            }
            printf("✅ JOINED SERVER AS CLIENT ID: %d\n", my_client_id);
        }
        len = recvfrom(sock, buffer, 4096, 0, (struct sockaddr*)&sender, &slen);
    }
}

int main(int argc, char* argv[]) {
    for(int i=1; i<argc; i++) {
        if(strcmp(argv[i], "--host") == 0 && i+1<argc) {
            strncpy(SERVER_HOST, argv[++i], 63);
        }
    }

    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT [BUILD 181 - CTF RELOADED]", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    
    local_init_match(1, 0);
    progression_reset(SDL_GetTicks());
    settings_load_if_needed();
    settings_set_server_shell_casings_override(-1);
    lobby_init_labels();
    ui_bridge_init("127.0.0.1", 17777);
    if (ui_bridge_fetch_state(&ui_state)) {
        ui_use_server = 1;
        ui_last_poll = SDL_GetTicks();
    }
    
    int running = 1;
    unsigned int last_frame_ms = SDL_GetTicks();
    while(running) {
        unsigned int now_frame_ms = SDL_GetTicks();
        float dt = (float)(now_frame_ms - last_frame_ms) / 1000.0f;
        if (dt < 0.001f) dt = 0.001f;
        if (dt > 0.050f) dt = 0.050f;
        last_frame_ms = now_frame_ms;
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if (e.type == SDL_WINDOWEVENT && e.window.event == SDL_WINDOWEVENT_FOCUS_GAINED && (app_state == STATE_GAME_LOCAL || app_state == STATE_GAME_NET)) SDL_SetRelativeMouseMode(SDL_TRUE);
            if (e.type == SDL_MOUSEBUTTONDOWN && (app_state == STATE_GAME_LOCAL || app_state == STATE_GAME_NET)) SDL_SetRelativeMouseMode(SDL_TRUE);
            
            if (app_state == STATE_LOBBY) {
                if (e.type == SDL_TEXTINPUT && ui_edit_index >= 0) {
                    size_t len = strlen(e.text.text);
                    if (len > 0 && ui_edit_len < (int)sizeof(ui_edit_buffer) - 1) {
                        int copy = (int)len;
                        if (ui_edit_len + copy >= (int)sizeof(ui_edit_buffer) - 1) {
                            copy = (int)sizeof(ui_edit_buffer) - 1 - ui_edit_len;
                        }
                        memcpy(ui_edit_buffer + ui_edit_len, e.text.text, (size_t)copy);
                        ui_edit_len += copy;
                        ui_edit_buffer[ui_edit_len] = '\0';
                    }
                }
                if (e.type == SDL_KEYDOWN) {
                    if (ui_edit_index >= 0) {
                        if (e.key.keysym.sym == SDLK_BACKSPACE && ui_edit_len > 0) {
                            ui_edit_len--;
                            ui_edit_buffer[ui_edit_len] = '\0';
                        }
                        if (e.key.keysym.sym == SDLK_RETURN || e.key.keysym.sym == SDLK_KP_ENTER) {
                            lobby_end_edit(1);
                        }
                        if (e.key.keysym.sym == SDLK_ESCAPE) {
                            lobby_end_edit(0);
                        }
                        continue;
                    }
                    if (e.key.keysym.sym == SDLK_UP) {
                        int count = lobby_menu_count();
                        lobby_selection = (lobby_selection + count - 1) % count;
                    }
                    if (e.key.keysym.sym == SDLK_DOWN) {
                        int count = lobby_menu_count();
                        lobby_selection = (lobby_selection + 1) % count;
                    }
                    // P0: no single-click or single-key activation in Emily UI.
                }
                if (e.type == SDL_MOUSEBUTTONDOWN && e.button.button == SDL_BUTTON_LEFT) {
                    int menu_count = lobby_menu_count();
                    float base_x = 360.0f;
                    float base_y = 520.0f;
                    float size = 70.0f;
                    float gap = 85.0f;
                    float mx = (float)e.button.x;
                    float my = 720.0f - (float)e.button.y;
                    int hit = lobby_hit_test(mx, my, menu_count, base_x, base_y, gap, size);
                    if (hit >= 0) {
                        unsigned int now = SDL_GetTicks();
                        if (ui_last_click_index == hit && ui_last_click_ms > 0) {
                            unsigned int delta = now - ui_last_click_ms;
                            if (delta <= 250) {
                                lobby_selection = hit;
                                lobby_start_action(hit);
                                ui_last_click_ms = 0;
                                ui_last_click_index = -1;
                            } else if (delta <= 700) {
                                lobby_selection = hit;
                                lobby_start_edit(hit);
                                ui_last_click_ms = 0;
                                ui_last_click_index = -1;
                            } else {
                                ui_last_click_ms = now;
                                ui_last_click_index = hit;
                            }
                        } else {
                            ui_last_click_ms = now;
                            ui_last_click_index = hit;
                        }
                    }
                }
                if (e.type == SDL_MOUSEBUTTONDOWN && e.button.button == SDL_BUTTON_LEFT) {
                    int menu_count = lobby_menu_count();
                    float base_x = 360.0f;
                    float base_y = 520.0f;
                    float size = 70.0f;
                    float gap = 85.0f;
                    float mx = (float)e.button.x;
                    float my = 720.0f - (float)e.button.y;
                    int hit = lobby_hit_test(mx, my, menu_count, base_x, base_y, gap, size);
                    if (hit >= 0) {
                        unsigned int now = SDL_GetTicks();
                        if (ui_last_click_index == hit && ui_last_click_ms > 0) {
                            unsigned int delta = now - ui_last_click_ms;
                            if (delta <= 250) {
                                lobby_selection = hit;
                                lobby_start_action(hit);
                                ui_last_click_ms = 0;
                                ui_last_click_index = -1;
                            } else if (delta <= 700) {
                                lobby_selection = hit;
                                lobby_start_edit(hit);
                                ui_last_click_ms = 0;
                                ui_last_click_index = -1;
                            } else {
                                ui_last_click_ms = now;
                                ui_last_click_index = hit;
                            }
                        } else {
                            ui_last_click_ms = now;
                            ui_last_click_index = hit;
                        }
                    }
                }
            } else {
                if (e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_o &&
                    (app_state == STATE_GAME_LOCAL || app_state == STATE_GAME_NET)) {
                    settings_toggle_overlay();
                }
                if (g_show_settings) {
                    if (e.type == SDL_KEYDOWN) {
                        if (e.key.keysym.sym == SDLK_ESCAPE || e.key.keysym.sym == SDLK_o) {
                            g_show_settings = 0;
                        } else if (e.key.keysym.sym == SDLK_w || e.key.keysym.sym == SDLK_UP ||
                                   e.key.keysym.sym == SDLK_s || e.key.keysym.sym == SDLK_DOWN) {
                            g_settings_selection = 0;
                        } else if (e.key.keysym.sym == SDLK_RETURN || e.key.keysym.sym == SDLK_KP_ENTER ||
                                   e.key.keysym.sym == SDLK_SPACE) {
                            settings_toggle_selected();
                        }
                    }
                } else {
                    if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) {
                        if (app_state == STATE_GAME_NET) net_shutdown();
                        app_state = STATE_LOBBY;
                        SDL_SetRelativeMouseMode(SDL_FALSE);
                        setup_lobby_2d();
                    }
                    if (app_state == STATE_RESULTS) {
                        if (e.type == SDL_KEYDOWN && (e.key.keysym.sym == SDLK_RETURN || e.key.keysym.sym == SDLK_KP_ENTER)) {
                            app_state = STATE_LOBBY;
                            SDL_SetRelativeMouseMode(SDL_FALSE);
                            setup_lobby_2d();
                        }
                    } else {
                        if (e.type == SDL_KEYDOWN) {
                            if (e.key.keysym.sym == SDLK_v || e.key.keysym.sym == SDLK_TAB) {
                                match_prog.camera_third_person = !match_prog.camera_third_person;
                                if (match_prog.camera_third_person) {
                                    int cam_id = (app_state == STATE_GAME_NET && my_client_id > 0 && my_client_id < MAX_CLIENTS && local_state.players[my_client_id].active)
                                        ? my_client_id
                                        : 0;
                                    camera_reseed_from_player(&local_state.players[cam_id]);
                                }
                            }
                            if (app_state == STATE_GAME_LOCAL) {
                                if (e.key.keysym.sym == SDLK_F1) progression_try_allocate(0);
                                if (e.key.keysym.sym == SDLK_F2) progression_try_allocate(1);
                                if (e.key.keysym.sym == SDLK_F3) progression_try_allocate(2);
                                if (e.key.keysym.sym == SDLK_F4) progression_try_allocate(3);
                                if (e.key.keysym.sym == SDLK_F5) progression_try_allocate(4);
                            }
                        }
                        if(e.type == SDL_MOUSEMOTION) {
                            float sens = (current_fov < 50.0f) ? 0.05f : 0.15f;
                            g_cam.yaw = norm_yaw_deg(g_cam.yaw - e.motion.xrel * sens);
                            g_cam.pitch = clamp_pitch_deg(g_cam.pitch - e.motion.yrel * sens);
                        }
                    }
                }
            }
        }
        if (app_state == STATE_GAME_LOCAL || app_state == STATE_GAME_NET) SDL_SetRelativeMouseMode(g_show_settings ? SDL_FALSE : SDL_TRUE);
        if (app_state == STATE_LOBBY) {
             unsigned int now = SDL_GetTicks();
             if (ui_use_server && (now - ui_last_poll) > 1000) {
                 if (ui_bridge_fetch_state(&ui_state)) {
                     ui_last_poll = now;
                     int count = lobby_menu_count();
                     if (lobby_selection >= count) lobby_selection = 0;
                 }
             }
             glClearColor(0.02f, 0.02f, 0.05f, 1.0f); // Dark Lobby
             glClear(GL_COLOR_BUFFER_BIT);
             setup_lobby_2d();
             glColor3f(0, 1, 1); // CYAN TEXT
             draw_string("SHANKPIT", 430, 560, 12);
             glColor3f(0.5f, 0.8f, 0.9f);
             draw_string("SELECT MODE", 500, 520, 6);

             float base_x = 360.0f;
             float base_y = 520.0f;
             float size = 70.0f;
             float gap = 85.0f;
             int menu_count = lobby_menu_count();
             draw_lobby_buttons(menu_count, base_x, base_y, gap, size);

             glColor3f(0.4f, 0.6f, 0.7f);
             draw_string("DOUBLE-CLICK: FAST=OPEN / SLOW=RENAME", 320, 140, 5);
             SDL_GL_SwapWindow(win);
        }
        else if (app_state == STATE_RESULTS) {
            draw_results_screen();
            SDL_GL_SwapWindow(win);
        }
        else {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if (!g_show_settings) {
                if(k[SDL_SCANCODE_W]) fwd += 1.0f;
                if(k[SDL_SCANCODE_S]) fwd -= 1.0f;
                if(k[SDL_SCANCODE_D]) str -= 1.0f;
                if(k[SDL_SCANCODE_A]) str += 1.0f;
            }
            int jump = (!g_show_settings && k[SDL_SCANCODE_SPACE]);
            int crouch = (!g_show_settings && k[SDL_SCANCODE_LCTRL]);
            int shoot = (!g_show_settings && (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT)));
            g_cam.ads_down = (!g_show_settings && ((SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_RIGHT)) != 0));
            g_ads_down = g_cam.ads_down;
            reticle_update_visual(&local_state.players[0], dt);
            int reload = (!g_show_settings && k[SDL_SCANCODE_R]);
            int use = (!g_show_settings && k[SDL_SCANCODE_F]);
            int bike = (!g_show_settings && k[SDL_SCANCODE_G]);
            int ability = (!g_show_settings && k[SDL_SCANCODE_E]);
            if (!g_show_settings) {
                if(k[SDL_SCANCODE_1]) wpn_req=0; if(k[SDL_SCANCODE_2]) wpn_req=1;
                if(k[SDL_SCANCODE_3]) wpn_req=2; if(k[SDL_SCANCODE_4]) wpn_req=3; if(k[SDL_SCANCODE_5]) wpn_req=4;
            }

            float target_fov = 75.0f;
            if (g_cam.ads_down) {
                if (local_state.players[0].current_weapon == WPN_SNIPER) target_fov = 20.0f;
                else target_fov = 55.0f;
            }
            current_fov += (target_fov - current_fov) * 0.2f;
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(current_fov, 1280.0/720.0, 0.1, Z_FAR); 
            glMatrixMode(GL_MODELVIEW);
            float aim_yaw = 0.0f, aim_pitch = 0.0f;
            AimRay aim_ray = get_aim_ray(&g_cam, &local_state.players[0]);
            yaw_pitch_from_dir(aim_ray.dir, &aim_yaw, &aim_pitch);
            if (app_state == STATE_GAME_NET) {
                UserCmd cmd = client_create_cmd(fwd, str, aim_yaw, aim_pitch, shoot, jump, crouch, reload, use, bike, ability, wpn_req);
                net_send_cmd(cmd);
                net_tick();
            } else {
                local_state.players[0].in_use = use;
                local_state.players[0].in_bike = bike;
                if (local_state.transition_timer == 0) {
                    PlayerState *p0 = &local_state.players[0];
                    int use_pressed = use && !p0->use_was_down;
                    int bike_pressed = bike && !p0->bike_was_down;
                    if (p0->vehicle_cooldown == 0 && use_pressed) {
                        int in_garage = local_state.scene_id == SCENE_GARAGE_OSAKA;
                        int portal_id = -1;
                        if (scene_portal_triggered(p0, &portal_id)) {
                            int dest_scene = -1;
                            float sx = 0.0f, sy = 0.0f, sz = 0.0f;
                            if (portal_resolve_destination(p0->scene_id, portal_id, p0->id, &dest_scene, &sx, &sy, &sz)) {
                                p0->scene_id = dest_scene;
                                p0->x = sx; p0->y = sy; p0->z = sz;
                                p0->vx = 0.0f; p0->vy = 0.0f; p0->vz = 0.0f;
                                p0->in_vehicle = 0;
                                p0->vehicle_type = VEH_NONE;
                                p0->bike_gear = 0;
                                scene_request_transition(dest_scene);
                            }
                        } else if ((in_garage && scene_near_vehicle_pad(local_state.scene_id, p0->x, p0->z, 6.0f, NULL)) || !in_garage) {
                            if (p0->vehicle_type == VEH_BUGGY) {
                                p0->vehicle_type = VEH_NONE; p0->in_vehicle = 0;
                            } else {
                                p0->vehicle_type = VEH_BUGGY; p0->in_vehicle = 1; p0->bike_gear = 0;
                            }
                            p0->vehicle_cooldown = 30;
                        }
                    } else if (p0->vehicle_cooldown == 0 && bike_pressed) {
                        int in_garage = local_state.scene_id == SCENE_GARAGE_OSAKA;
                        if ((in_garage && scene_near_vehicle_pad(local_state.scene_id, p0->x, p0->z, 6.0f, NULL)) || !in_garage) {
                            if (p0->vehicle_type == VEH_BIKE) {
                                p0->vehicle_type = VEH_NONE; p0->in_vehicle = 0; p0->bike_gear = 0;
                            } else {
                                p0->vehicle_type = VEH_BIKE; p0->in_vehicle = 1; p0->bike_gear = 1;
                            }
                            p0->vehicle_cooldown = 30;
                        }
                    }
                    p0->use_was_down = use;
                    p0->bike_was_down = bike;
                }
                if(local_state.players[0].vehicle_cooldown > 0) local_state.players[0].vehicle_cooldown--;
                unsigned int now_ms = SDL_GetTicks();
                local_update(fwd, str, aim_yaw, aim_pitch, shoot, wpn_req, jump, crouch, reload, use, bike, ability, NULL, now_ms);
                progression_tick(&local_state.players[0], now_ms);
                if (app_state == STATE_GAME_LOCAL && now_ms >= match_prog.match_end_ms) {
                    app_state = STATE_RESULTS;
                    SDL_SetRelativeMouseMode(SDL_FALSE);
                    setup_lobby_2d();
                }
            }
            static int cam_last_render_id = -1;
            static int cam_last_deaths = -1;
            PlayerState *render_p = &local_state.players[0];
            int render_id = 0;
            if (app_state == STATE_GAME_NET &&
                my_client_id > 0 && my_client_id < MAX_CLIENTS &&
                local_state.players[my_client_id].active) {
                render_p = &local_state.players[my_client_id];
                render_id = my_client_id;
            }
            if (render_id != cam_last_render_id) {
                cam_last_render_id = render_id;
                cam_last_deaths = render_p->deaths;
                camera_reseed_from_player(render_p);
            } else if (render_p->deaths > cam_last_deaths) {
                cam_last_deaths = render_p->deaths;
                camera_reseed_from_player(render_p);
            }
            for (int i = 0; i < MAX_CLIENTS; i++) {
                player_update_run_cycle(&local_state.players[i], dt);
            }
            draw_scene(render_p, dt);
            SDL_GL_SwapWindow(win);
        }
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
