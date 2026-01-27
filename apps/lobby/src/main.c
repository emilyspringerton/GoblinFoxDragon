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
    glLineWidth(1.0f); // Thinner line for small text
    glBegin(GL_LINES);
    // Standard block font
    if(c>='0'&&c<='9'){ glVertex2f(x,y);glVertex2f(x+s,y);glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y); }
    else if(c=='.'){ glVertex2f(x+s/2,y);glVertex2f(x+s/2,y+s*0.2); }
    else if(c=='V'){glVertex2f(x,y+s);glVertex2f(x+s/2,y);glVertex2f(x+s/2,y);glVertex2f(x+s,y+s);}
    else if(c=='E'){glVertex2f(x+s,y);glVertex2f(x,y);glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s,y+s);glVertex2f(x,y+s/2);glVertex2f(x+s*0.8,y+s/2);}
    else if(c=='L'){glVertex2f(x,y+s);glVertex2f(x,y);glVertex2f(x,y);glVertex2f(x+s,y);}
    else if(c==':'){glVertex2f(x+s/2,y+s*0.2);glVertex2f(x+s/2,y+s*0.3);glVertex2f(x+s/2,y+s*0.7);glVertex2f(x+s/2,y+s*0.8);}
    // Fallback box for others
    else if(c==' '){} 
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
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); glScalef(0.05f, 0.05f, 0.8f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); glScalef(0.15f, 0.2f, 0.5f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); glScalef(0.1f, 0.15f, 1.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); glScalef(0.25f, 0.15f, 0.8f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); glScalef(0.08f, 0.12f, 2.0f); break;
    }
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
}

void draw_weapon_p(PlayerState *p) {
    glPushMatrix(); glLoadIdentity();
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
    glVertex3f(-0.4, 0.8, -0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(-0.4, 0, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(-0.4, 0, 0.4); glVertex3f(-0.4, 0, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(0.4, 0.8, -0.4);
    glEnd();
}

void draw_player_3rd(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y + 2.0f, p->z);
    glRotatef(-p->yaw, 0, 1, 0); 
    if(p->health <= 0) glColor3f(0.2, 0, 0); else glColor3f(1, 0, 0); 
    
    // GOLDEN RATIO SCALE (1.618x)
    glPushMatrix(); glScalef(0.97f, 2.91f, 0.97f); 
    glBegin(GL_QUADS);
    glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
    glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
    glEnd(); glPopMatrix();
    
    glPushMatrix(); glTranslatef(0, 1.54f, 0); draw_head(p->current_weapon); glPopMatrix();
    glPushMatrix(); glTranslatef(0.5f, 1.0f, 0.5f); glRotatef(p->pitch, 1, 0, 0);   
    glScalef(0.8f, 0.8f, 0.8f); draw_gun_model(p->current_weapon); glPopMatrix(); glPopMatrix();
}

void draw_projectiles() { }

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    glColor3f(0, 1, 0);
    if (current_fov < 50.0f) { glBegin(GL_LINES); glVertex2f(0, 360); glVertex2f(1280, 360); glVertex2f(640, 0); glVertex2f(640, 720); glEnd(); } 
    else { glLineWidth(2.0f); glBegin(GL_LINES); glVertex2f(630, 360); glVertex2f(650, 360); glVertex2f(640, 350); glVertex2f(640, 370); glEnd(); }
    if (p->hit_feedback > 0) {
        if (p->hit_feedback == 20) glColor3f(1, 0, 1); else glColor3f(0, 1, 1);
        glBegin(GL_TRIANGLE_FAN); glVertex2f(640, 360); for(int i=0; i<=20; i++) { float ang = (float)i/20.0f*6.283f; glVertex2f(640+cosf(ang)*15, 360+sinf(ang)*15); } glEnd();
    }
    glColor3f(0.2f, 0, 0); glRectf(50, 50, 250, 70); glColor3f(1.0f, 0, 0); glRectf(50, 50, 50 + (p->health * 2), 70);
    glColor3f(0, 0, 0.2f); glRectf(50, 80, 250, 100); glColor3f(0.2f, 0.2f, 1.0f); glRectf(50, 80, 50 + (p->shield * 2), 100);
    int w = p->current_weapon; int ammo = (w == WPN_KNIFE) ? 99 : p->ammo[w];
    glColor3f(1, 1, 0); for(int i=0; i<ammo; i++) glRectf(1200 - (i*10), 50, 1205 - (i*10), 80);
    
    // --- OLD SPEEDOMETER (LEFT) ---
    // float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    // char buf[32]; sprintf(buf, "SPD: %d", (int)(speed * 100));
    // draw_string(buf, 50, 120, 10);
    
    // --- NEW TELEMETRY (BOTTOM RIGHT, YELLOW) ---
    float raw_speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    char vel_buf[32]; sprintf(vel_buf, "VEL: %.2f", raw_speed);
    glColor3f(1.0f, 1.0f, 0.0f); // Yellow
    // Right align approx: 1280 - (Chars * 15)
    draw_string(vel_buf, 1100, 50, 8); // Size 8 (Small-ish)

    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f); glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT); glLoadIdentity();
    float cam_h = render_p->crouching ? 2.5f : EYE_HEIGHT;
    glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0); glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);
    draw_grid(); draw_map(); draw_projectiles();
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
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT [BUILD 129 - TELEMETRY]", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    
    local_init_match(1, 0);
    
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            
            if (app_state == STATE_LOBBY) {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_d) { app_state = STATE_GAME_LOCAL; local_init_match(1, MODE_DEATHMATCH); }
                    if (e.key.keysym.sym == SDLK_b) { app_state = STATE_GAME_LOCAL; local_init_match(8, MODE_DEATHMATCH); }
                    if (app_state != STATE_LOBBY) {
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                }
            } else {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_ESCAPE) {
                        app_state = STATE_LOBBY;
                        SDL_SetRelativeMouseMode(SDL_FALSE);
                        glDisable(GL_DEPTH_TEST);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
                        glMatrixMode(GL_MODELVIEW); glLoadIdentity();
                    }
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

        if (app_state == STATE_LOBBY) {
             glClearColor(0.1f, 0.1f, 0.2f, 1.0f); glClear(GL_COLOR_BUFFER_BIT);
             glLoadIdentity(); glColor3f(1, 1, 0);
             draw_string("SHANKPIT", 400, 500, 20);
             draw_string("D: DEMO", 400, 400, 10);
             draw_string("B: BATTLE", 400, 350, 10);
             SDL_GL_SwapWindow(win);
        } 
        else if (app_state == STATE_GAME_LOCAL) {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd-=1; if(k[SDL_SCANCODE_S]) fwd+=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            int jump = k[SDL_SCANCODE_SPACE]; int crouch = k[SDL_SCANCODE_LCTRL];
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int reload = k[SDL_SCANCODE_R];
            if(k[SDL_SCANCODE_1]) wpn_req=0; if(k[SDL_SCANCODE_2]) wpn_req=1; 
            if(k[SDL_SCANCODE_3]) wpn_req=2; if(k[SDL_SCANCODE_4]) wpn_req=3; if(k[SDL_SCANCODE_5]) wpn_req=4;

            float target_fov = (local_state.players[0].current_weapon == WPN_SNIPER && (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_RIGHT))) ? 20.0f : 75.0f;
            current_fov += (target_fov - current_fov) * 0.2f;
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(current_fov, 1280.0/720.0, 0.1, 1000.0); glMatrixMode(GL_MODELVIEW);

            local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn_req, jump, crouch, reload, NULL, 0);
            draw_scene(&local_state.players[0]);
            SDL_GL_SwapWindow(win);
        }
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
