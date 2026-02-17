#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <assert.h>

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

float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;

#define Z_FAR 8000.0f

int sock = -1;
struct sockaddr_in server_addr;

#define NET_CMD_HISTORY 3
UserCmd net_cmd_history[NET_CMD_HISTORY];
int net_cmd_history_count = 0;
int net_cmd_seq = 0;

static int net_local_pid = -1;
static int last_applied_scene_id = -999;

void net_connect();
void net_shutdown();

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
    local_state.players[0].scene_id = local_state.scene_id;
}

void draw_string(const char* str, float x, float y, float size) {
    TurtlePen pen = turtle_pen_create(x, y, size);
    turtle_draw_text(&pen, str);
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

// --- NEON BRUTALIST BLOCK RENDERER ---
void draw_map() {
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
    glRotatef(-draw_yaw, 0, 1, 0);
    if (p->in_vehicle) {
        draw_buggy_model();
    } else {
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

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    glColor3f(0, 1, 0);
    if (current_fov < 50.0f) { glBegin(GL_LINES); glVertex2f(0, 360); glVertex2f(1280, 360); glVertex2f(640, 0); glVertex2f(640, 720); glEnd(); } 
    else { glLineWidth(2.0f); glBegin(GL_LINES); glVertex2f(630, 360); glVertex2f(650, 360); glVertex2f(640, 350); glVertex2f(640, 370); glEnd(); }
    
    // --- HIT INDICATORS ---
    if (p->hit_feedback > 0) {
        if (p->hit_feedback >= 25) glColor3f(1.0f, 0.0f, 0.0f); // RED (Kill/High Dmg)
        else glColor3f(0.0f, 1.0f, 0.0f); // GREEN (Normal)
        
        glLineWidth(2.0f);
        draw_circle(640, 360, 20.0f, 16); // Hit Ring
        
        // DOUBLE RING FOR KILL
        if (p->hit_feedback >= 25) {
            draw_circle(640, 360, 28.0f, 16); // Outer Kill Ring
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

    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

static int target_in_view(PlayerState *p, float tx, float ty, float tz, float max_dist, float min_dot) {
    float rad_yaw = cam_yaw * 0.0174533f;
    float rad_pitch = cam_pitch * 0.0174533f;
    float fx = sinf(rad_yaw) * cosf(rad_pitch);
    float fy = sinf(rad_pitch);
    float fz = -cosf(rad_yaw) * cosf(rad_pitch);

    float ox = p->x;
    float oy = p->y + EYE_HEIGHT;
    float oz = p->z;

    float dx = tx - ox;
    float dy = ty - oy;
    float dz = tz - oz;
    float dist = sqrtf(dx * dx + dy * dy + dz * dz);
    if (dist > max_dist || dist <= 0.001f) return 0;
    dx /= dist; dy /= dist; dz /= dist;
    float dot = fx * dx + fy * dy + fz * dz;
    return dot >= min_dot;
}

static void draw_garage_portal_frame() {
    if (!scene_portal_active(local_state.scene_id)) return;
    float px = 0.0f, py = 0.0f, pz = 0.0f, pr = 0.0f;
    scene_portal_info(local_state.scene_id, &px, &py, &pz, &pr);

    glPushMatrix();
    glTranslatef(px, py, pz);
    glColor3f(0.2f, 0.9f, 1.0f);
    glLineWidth(3.0f);
    glBegin(GL_LINE_LOOP);
    glVertex3f(-pr, -2.0f, 0.0f);
    glVertex3f(pr, -2.0f, 0.0f);
    glVertex3f(pr, 6.0f, 0.0f);
    glVertex3f(-pr, 6.0f, 0.0f);
    glEnd();
    glPopMatrix();
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
    draw_string("PORTAL -> STADIUM", 40, 640, 6);

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

    float portal_x = 0.0f, portal_y = 0.0f, portal_z = 0.0f, portal_r = 0.0f;
    scene_portal_info(local_state.scene_id, &portal_x, &portal_y, &portal_z, &portal_r);
    int portal_target = target_in_view(p, portal_x, portal_y, portal_z, 30.0f, 0.75f);
    int pad_target = 0;
    if (scene_near_vehicle_pad(local_state.scene_id, p->x, p->z, 12.0f, NULL)) {
        int pad_idx = -1;
        if (scene_near_vehicle_pad(local_state.scene_id, p->x, p->z, 12.0f, &pad_idx)) {
            pad_target = target_in_view(p, pads[pad_idx].x, pads[pad_idx].y + 1.0f, pads[pad_idx].z, 20.0f, 0.7f);
        }
    }

    glColor3f(1.0f, 1.0f, 0.0f);
    if (portal_target) {
        draw_string("TRAVEL", 600, 350, 8);
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

void draw_scene(PlayerState *render_p) {
    glClearColor(0.02f, 0.02f, 0.05f, 1.0f); // DEEP SPACE BLUE BG
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT); glLoadIdentity();
    local_state.scene_id = render_p->scene_id;
    phys_set_scene(render_p->scene_id);
    float cam_y = render_p->in_vehicle ? 8.0f : (render_p->crouching ? 2.5f : EYE_HEIGHT);
    float cam_z_off = render_p->in_vehicle ? 10.0f : 0.0f;
    
    float rad = -cam_yaw * 0.01745f;
    float cx = sinf(rad) * cam_z_off;
    float cz = cosf(rad) * cam_z_off;
    
    glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-(render_p->x - cx), -(render_p->y + cam_y), -(render_p->z - cz));
    
    draw_grid(); 
    update_and_draw_trails();
    draw_map();
    draw_garage_vehicle_pads();
    draw_garage_portal_frame();
    draw_projectiles();
    if (render_p->in_vehicle) draw_player_3rd(render_p);
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active || p->scene_id != render_p->scene_id) continue;
        if (p == render_p) continue;
        draw_player_3rd(p);
    }
    draw_weapon_p(render_p); draw_hud(render_p); draw_garage_overlay(render_p);
    draw_travel_overlay();
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
    if (sock < 0) net_init();
    struct hostent *he = gethostbyname(SERVER_HOST);
    if (he) {
        server_addr.sin_family = AF_INET; 
        server_addr.sin_port = htons(SERVER_PORT); 
        memcpy(&server_addr.sin_addr, he->h_addr_list[0], he->h_length);
        char buffer[128];
        NetHeader *h = (NetHeader*)buffer;
        h->type = PACKET_CONNECT;
        h->scene_id = 0;
        sendto(sock, buffer, sizeof(NetHeader), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
        printf("Connected to %s...\n", SERVER_HOST);
    } else {
        printf("Failed to resolve %s\n", SERVER_HOST);
    }
}

UserCmd client_create_cmd(float fwd, float str, float yaw, float pitch, int shoot, int jump, int crouch, int reload, int use, int ability, int wpn_idx) {
    UserCmd cmd;
    memset(&cmd, 0, sizeof(UserCmd));
    cmd.sequence = ++net_cmd_seq; cmd.timestamp = SDL_GetTicks();
    cmd.yaw = yaw; cmd.pitch = pitch;
    cmd.fwd = fwd; cmd.str = str;
    if(shoot) cmd.buttons |= BTN_ATTACK; if(jump) cmd.buttons |= BTN_JUMP;
    if(crouch) cmd.buttons |= BTN_CROUCH;
    if(reload) cmd.buttons |= BTN_RELOAD;
    if(use) cmd.buttons |= BTN_USE;
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
    if (len < (int)sizeof(NetHeader)) return;
    if (len < (int)sizeof(NetHeader) + 1) return;

    int cursor = (int)sizeof(NetHeader);
    unsigned char count = *(unsigned char*)(buffer + cursor); cursor++;

    int needed = (int)sizeof(NetHeader) + 1 + (int)count * (int)sizeof(NetPlayer);
    if (len < needed) return;

    for(int i=0; i<count; i++) {
        if (cursor + (int)sizeof(NetPlayer) > len) break;

        NetPlayer *np = (NetPlayer*)(buffer + cursor);
        cursor += (int)sizeof(NetPlayer);

        int id = np->id;
        if (id <= 0 || id >= MAX_CLIENTS) continue;

        PlayerState *p = &local_state.players[id];
        p->active = 1;
        p->scene_id = np->scene_id;
        p->x = np->x; p->y = np->y; p->z = np->z;
        if (id != my_client_id) {
            p->yaw   = norm_yaw_deg(np->yaw);
            p->pitch = clamp_pitch_deg(np->pitch);
        }
        p->health = np->health;

        int safe_weapon = np->current_weapon;
        if (safe_weapon < 0 || safe_weapon >= MAX_WEAPONS) {
            safe_weapon = WPN_MAGNUM;
        }
        p->current_weapon = safe_weapon;

        p->is_shooting = np->is_shooting;
        p->in_vehicle = np->in_vehicle;
        p->storm_charges = np->storm_charges;
        p->hit_feedback = np->hit_feedback;
        p->ammo[p->current_weapon] = np->ammo;

        if (p->is_shooting) p->recoil_anim = 1.0f;
    }

    int render_id = (my_client_id > 0 && my_client_id < MAX_CLIENTS && local_state.players[my_client_id].active)
        ? my_client_id
        : 0;
    int scene_id = local_state.players[render_id].scene_id;
    if (scene_id < 0) {
        scene_id = local_state.scene_id;
    }
    if (scene_id != last_applied_scene_id && scene_id >= 0) {
        last_applied_scene_id = scene_id;
        client_apply_scene_id(scene_id, SDL_GetTicks());
    }
}


void net_tick() {
    char buffer[4096];
    struct sockaddr_in sender;
    socklen_t slen = sizeof(sender);
    int len = recvfrom(sock, buffer, 4096, 0, (struct sockaddr*)&sender, &slen);
    while (len > 0) {
        if (len < (int)sizeof(NetHeader)) {
            len = recvfrom(sock, buffer, 4096, 0, (struct sockaddr*)&sender, &slen);
            continue;
        }

        NetHeader *head = (NetHeader*)buffer;
        if (head->type == PACKET_SNAPSHOT) {
            net_process_snapshot(buffer, len);
        }
        if (head->type == PACKET_WELCOME) {
            my_client_id = head->client_id;

            // Server-assigned identity becomes our local simulation/render identity
            net_local_pid = my_client_id;

            if (my_client_id > 0 && my_client_id < MAX_CLIENTS) {
                PlayerState *lp = &local_state.players[my_client_id];

                // Pre-warm local slot so camera/render doesn't lag behind.
                lp->active   = 1;
                lp->scene_id = head->scene_id;

        
                if (head->scene_id >= 0) {
                    client_apply_scene_id(head->scene_id, SDL_GetTicks());
                }
            } else {
                 // invalid id => disable local sim/render identity
                net_local_pid = -1;
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
    lobby_init_labels();
    ui_bridge_init("127.0.0.1", 17777);
    if (ui_bridge_fetch_state(&ui_state)) {
        ui_use_server = 1;
        ui_last_poll = SDL_GetTicks();
    }
    
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if (e.type == SDL_WINDOWEVENT && e.window.event == SDL_WINDOWEVENT_FOCUS_GAINED && app_state != STATE_LOBBY) SDL_SetRelativeMouseMode(SDL_TRUE);
            if (e.type == SDL_MOUSEBUTTONDOWN && app_state != STATE_LOBBY) SDL_SetRelativeMouseMode(SDL_TRUE);
            
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
                if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) {
                    if (app_state == STATE_GAME_NET) net_shutdown();
                    app_state = STATE_LOBBY;
                    SDL_SetRelativeMouseMode(SDL_FALSE);
                    setup_lobby_2d();
                }
                if(e.type == SDL_MOUSEMOTION) {
                    float sens = (current_fov < 50.0f) ? 0.05f : 0.15f; 
                    cam_yaw -= e.motion.xrel * sens;
                    if(cam_yaw > 360) cam_yaw -= 360; if(cam_yaw < 0) cam_yaw += 360;
                    cam_pitch -= e.motion.yrel * sens;
                    if(cam_pitch > 89) cam_pitch = 89; if(cam_pitch < -89) cam_pitch = -89;
                }
            }
        }
        if (app_state != STATE_LOBBY) SDL_SetRelativeMouseMode(SDL_TRUE);
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
        else {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd-=1; if(k[SDL_SCANCODE_S]) fwd+=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            int jump = k[SDL_SCANCODE_SPACE]; int crouch = k[SDL_SCANCODE_LCTRL];
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int reload = k[SDL_SCANCODE_R];
            int use = k[SDL_SCANCODE_F];
            int ability = k[SDL_SCANCODE_E];
            if(k[SDL_SCANCODE_1]) wpn_req=0; if(k[SDL_SCANCODE_2]) wpn_req=1;
            if(k[SDL_SCANCODE_3]) wpn_req=2; if(k[SDL_SCANCODE_4]) wpn_req=3; if(k[SDL_SCANCODE_5]) wpn_req=4;

            int fov_pid = (app_state == STATE_GAME_NET && net_local_pid > 0 && local_state.players[net_local_pid].active)
                ? net_local_pid
                : 0;
            float target_fov = (local_state.players[fov_pid].current_weapon == WPN_SNIPER && (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_RIGHT))) ? 20.0f : 75.0f;
            current_fov += (target_fov - current_fov) * 0.2f;
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(current_fov, 1280.0/720.0, 0.1, Z_FAR); 
            glMatrixMode(GL_MODELVIEW);
            if (app_state == STATE_GAME_NET) {\
                net_local_pid = (my_client_id > 0 && my_client_id < MAX_CLIENTS) ? my_client_id : -1;
                net_tick();
                if (net_local_pid > 0) {
                    UserCmd cmd = client_create_cmd(fwd, str, cam_yaw, cam_pitch, shoot, jump, crouch, reload, use, ability, wpn_req);
                    net_send_cmd(cmd);
                }
            } else {
                local_state.players[0].in_use = use;
                if (use && local_state.players[0].vehicle_cooldown == 0 && local_state.transition_timer == 0) {
                    PlayerState *p0 = &local_state.players[0];
                    int in_garage = local_state.scene_id == SCENE_GARAGE_OSAKA;
                    int portal_id = -1;
                    if (scene_portal_triggered(p0, &portal_id)) {
                        int dest_scene = -1;
                        float sx = 0.0f, sy = 0.0f, sz = 0.0f;
                        if (portal_resolve_destination(p0->scene_id, portal_id, p0->id, &dest_scene, &sx, &sy, &sz)) {
                            p0->scene_id = dest_scene;
                            p0->x = sx; p0->y = sy; p0->z = sz;
                            p0->vx = 0.0f; p0->vy = 0.0f; p0->vz = 0.0f;
                            scene_request_transition(dest_scene);
                        }
                    } else if (in_garage && scene_near_vehicle_pad(local_state.scene_id, p0->x, p0->z, 6.0f, NULL)) {
                        p0->in_vehicle = !p0->in_vehicle;
                        p0->vehicle_cooldown = 30;
                    } else if (!in_garage) {
                        p0->in_vehicle = !p0->in_vehicle;
                        p0->vehicle_cooldown = 30;
                    }
                }
                if(local_state.players[0].vehicle_cooldown > 0) local_state.players[0].vehicle_cooldown--;
                unsigned int now_ms = SDL_GetTicks();
                local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn_req, jump, crouch, reload, ability, NULL, now_ms);
            }
            int render_pid = 0;
            if (app_state == STATE_GAME_NET &&
                my_client_id > 0 && my_client_id < MAX_CLIENTS &&
                local_state.players[my_client_id].active) {
                render_pid = my_client_id;
            }
            int sim_pid = (app_state == STATE_GAME_NET) ? net_local_pid : 0;
            if (app_state == STATE_GAME_NET &&
                my_client_id > 0 && my_client_id < MAX_CLIENTS &&
                local_state.players[my_client_id].active) {
                if (sim_pid != render_pid) {
                    printf("PID mismatch (active): my_client_id=%d sim_pid=%d render_pid=%d\n",
                           my_client_id, sim_pid, render_pid);
                }
                if (sim_pid != render_pid) {
                    printf("PID mismatch (active): my_client_id=%d sim_pid=%d render_pid=%d\n",
                    my_client_id, sim_pid, render_pid);
                    // fail open: trust render_pid
                    sim_pid = render_pid;
                }
            }
            PlayerState *render_p = &local_state.players[render_pid];
            draw_scene(render_p);
            SDL_GL_SwapWindow(win);
        }
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
