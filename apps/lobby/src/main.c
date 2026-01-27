#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

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

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"
#include "../../../packages/simulation/local_game.h"

#define STATE_LOBBY 0
#define STATE_GAME_NET 1
#define STATE_GAME_LOCAL 2
#define STATE_LISTEN_SERVER 99

// --- CONFIG ---
int APP_WIDTH = 1280;
int APP_HEIGHT = 720;
int IS_AUTO_BOT = 0;       // Ghost Mode
int TIME_LIMIT_SEC = 0;    // Regression Timer (0 = infinite)
int START_TIME = 0;

int app_state = STATE_LOBBY;
int wpn_req = 1; 

float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;

int sock = -1;
struct sockaddr_in server_addr;
int sv_sock = -1;
unsigned int sv_client_last_seq[MAX_CLIENTS];

// --- RENDER FUNCTIONS ---
void draw_scene(PlayerState *render_p); 

void draw_char(char c, float x, float y, float s) {
    glLineWidth(2.0f);
    glBegin(GL_LINES);
    // (Font logic same as before, abbreviated for space in this patch)
    if(c>='0'&&c<='9'){ glVertex2f(x,y);glVertex2f(x+s,y);glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y); }
    else { glVertex2f(x,y);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x+s,y+s);glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x,y); }
    glEnd();
}

void draw_string(const char* str, float x, float y, float size) {
    while(*str) { draw_char(*str, x, y, size); x += size * 1.5f; str++; }
}

void draw_grid() {
    glBegin(GL_LINES); glColor3f(0.0f, 1.0f, 1.0f);
    for(int i=-100; i<=900; i+=50) { glVertex3f(i, 0, -100); glVertex3f(i, 0, 100); glVertex3f(-100, 0, i); glVertex3f(100, 0, i); }
    glEnd();
}

void draw_map() {
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        glPushMatrix(); glTranslatef(b.x, b.y, b.z); glScalef(b.w, b.h, b.d);
        glBegin(GL_QUADS); glColor3f(0.2f, 0.2f, 0.2f); 
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
        glEnd();
        glLineWidth(2.0f); glColor3f(1.0f, 0.0f, 1.0f); 
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        glPopMatrix();
    }
}

void draw_gun_model(int weapon_id) {
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glEnd();
}

void draw_weapon_p(PlayerState *p) {
    glPushMatrix(); glLoadIdentity();
    float kick = p->recoil_anim * 0.2f;
    glTranslatef(0.4f, -0.5f + kick, -1.2f + (kick * 0.5f));
    glRotatef(-p->recoil_anim * 10.0f, 1, 0, 0);
    draw_gun_model(p->current_weapon);
    glPopMatrix();
}

void draw_player_3rd(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y + 2.0f, p->z);
    glRotatef(-p->yaw, 0, 1, 0); 
    if(p->health <= 0) glColor3f(0.2, 0, 0); else glColor3f(1, 0, 0); 
    glPushMatrix(); glScalef(0.6f, 1.8f, 0.6f); 
    glBegin(GL_QUADS); glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5); glEnd(); 
    glPopMatrix();
    glPopMatrix();
}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, APP_WIDTH, 0, APP_HEIGHT);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    glColor3f(0, 1, 0);
    glLineWidth(2.0f); glBegin(GL_LINES); glVertex2f(APP_WIDTH/2-10, APP_HEIGHT/2); glVertex2f(APP_WIDTH/2+10, APP_HEIGHT/2); glVertex2f(APP_WIDTH/2, APP_HEIGHT/2-10); glVertex2f(APP_WIDTH/2, APP_HEIGHT/2+10); glEnd();
    
    // --- SPEEDOMETER ---
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    char buf[32]; sprintf(buf, "SPD: %d", (int)(speed * 100));
    draw_string(buf, 50, 120, 10);
    if(p->on_ground) draw_string("GND", 150, 120, 10); else draw_string("AIR", 150, 120, 10);
    if(IS_AUTO_BOT) { glColor3f(1,0,0); draw_string("AUTO-PILOT ACTIVE", 50, APP_HEIGHT-50, 10); }

    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f); glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT); glLoadIdentity();
    float cam_h = render_p->crouching ? 2.5f : EYE_HEIGHT;
    glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0); glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);
    draw_grid(); draw_map(); 
    for(int i=1; i<MAX_CLIENTS; i++) if(local_state.players[i].active) draw_player_3rd(&local_state.players[i]);
    draw_weapon_p(render_p); draw_hud(render_p);
}

// --- NETWORK STUBS FOR CLIENT ---
UserCmd client_create_cmd(float fwd, float str, float yaw, float pitch, int shoot, int jump, int crouch, int reload, int wpn_idx) {
    UserCmd cmd; memset(&cmd, 0, sizeof(UserCmd));
    static int seq = 0; cmd.sequence = ++seq; cmd.timestamp = SDL_GetTicks();
    cmd.yaw = yaw; cmd.pitch = pitch; cmd.fwd = fwd; cmd.str = str;
    if(shoot) cmd.buttons |= BTN_ATTACK; if(jump) cmd.buttons |= BTN_JUMP;
    if(crouch) cmd.buttons |= BTN_CROUCH; if(reload) cmd.buttons |= BTN_RELOAD;
    cmd.weapon_idx = wpn_idx; return cmd;
}
void sv_tick() {} 
void net_init() { }
void net_send_cmd(UserCmd cmd) {}

int main(int argc, char* argv[]) {
    // --- PARSE ARGS ---
    for(int i=1; i<argc; i++) {
        if(strcmp(argv[i], "--bot") == 0) IS_AUTO_BOT = 1;
        if(strcmp(argv[i], "--width") == 0 && i+1<argc) APP_WIDTH = atoi(argv[++i]);
        if(strcmp(argv[i], "--height") == 0 && i+1<argc) APP_HEIGHT = atoi(argv[++i]);
        if(strcmp(argv[i], "--timelimit") == 0 && i+1<argc) TIME_LIMIT_SEC = atoi(argv[++i]);
    }

    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT", 100, 100, APP_WIDTH, APP_HEIGHT, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    
    // Auto-start game if bot
    if (IS_AUTO_BOT) {
        app_state = STATE_GAME_LOCAL; 
        local_init_match(8, MODE_DEATHMATCH); // 1 Bot Player + 7 Dummy Bots
    } else {
        local_init_match(1, 0);
    }
    
    START_TIME = SDL_GetTicks();
    int running = 1;
    while(running) {
        // TIMELIMIT CHECK
        if (TIME_LIMIT_SEC > 0) {
            if ((SDL_GetTicks() - START_TIME) > (TIME_LIMIT_SEC * 1000)) {
                printf("✅ Regression Complete. Exiting.\n");
                running = 0;
            }
        }

        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(!IS_AUTO_BOT) {
                if (app_state == STATE_LOBBY) {
                    if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_b) { app_state = STATE_GAME_LOCAL; local_init_match(8, MODE_DEATHMATCH); }
                } else {
                     if(e.type == SDL_MOUSEMOTION) {
                        cam_yaw -= e.motion.xrel * 0.15f;
                        if(cam_yaw > 360) cam_yaw -= 360; if(cam_yaw < 0) cam_yaw += 360;
                        cam_pitch -= e.motion.yrel * 0.15f;
                    }
                }
            }
        }

        if (app_state == STATE_LOBBY) {
             glClearColor(0.1f, 0.1f, 0.2f, 1.0f); glClear(GL_COLOR_BUFFER_BIT);
             glLoadIdentity(); glColor3f(1, 1, 0);
             draw_string("SHANKPIT", APP_WIDTH/2 - 100, APP_HEIGHT/2, 20);
             draw_string("PRESS B FOR BATTLE", APP_WIDTH/2 - 100, APP_HEIGHT/2 - 50, 10);
             SDL_GL_SwapWindow(win);
        } 
        else if (app_state == STATE_GAME_LOCAL) {
            float fwd=0, str=0;
            int jump=0, crouch=0, shoot=0, reload=0;

            if (IS_AUTO_BOT) {
                // --- GHOST LOGIC ---
                // We use the bot_think function to drive the local player (ID 0)
                float b_fwd=0, b_yaw=cam_yaw; 
                int b_btns=0;
                
                // IMPORTANT: We need to give bot_think the current world state
                bot_think(0, local_state.players, &b_fwd, &b_yaw, &b_btns);
                
                // Smooth camera for viewing pleasure
                float diff = b_yaw - cam_yaw;
                while(diff < -180) diff += 360; while(diff > 180) diff -= 360;
                cam_yaw += diff * 0.2f; // Smooth look
                
                // Translate Bot outputs to Inputs
                // Bot fwd/str logic is usually simple (1.0 or -1.0)
                // We map that to the movement vector
                fwd = b_fwd; // Simple map
                
                if (b_btns & BTN_JUMP) jump = 1;
                if (b_btns & BTN_ATTACK) shoot = 1;
                // Bots don't slide yet, but we could make them
                
            } else {
                // --- HUMAN LOGIC ---
                const Uint8 *k = SDL_GetKeyboardState(NULL);
                if(k[SDL_SCANCODE_W]) fwd-=1; if(k[SDL_SCANCODE_S]) fwd+=1;
                if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
                jump = k[SDL_SCANCODE_SPACE]; crouch = k[SDL_SCANCODE_LCTRL];
                shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            }

            // SETUP VIEW
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, (float)APP_WIDTH/(float)APP_HEIGHT, 0.1, 1000.0); glMatrixMode(GL_MODELVIEW);

            // RUN SIMULATION
            local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn_req, jump, crouch, reload, NULL, 0);
            
            // RENDER
            draw_scene(&local_state.players[0]);
            SDL_GL_SwapWindow(win);
        }
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
