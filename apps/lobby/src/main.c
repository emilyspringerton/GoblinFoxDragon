#include "../../packages/common/protocol.h"
#ifndef STATE_MENU
#define STATE_MENU 0
#define STATE_PLAYING 1
#define MODE_BOTS 0
#define MODE_DEV 1
#define MODE_SERV 2
#define MODE_NET 3
#define SCREEN_HEIGHT 600
#endif
#include "../../packages/common/protocol.h"
#ifndef STATE_MENU
#define STATE_MENU 0
#define STATE_PLAYING 1
#define MODE_BOTS 0
#define MODE_DEV 1
#define MODE_SERV 2
#define MODE_NET 3
#define SCREEN_HEIGHT 600
#endif

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
int app_state = STATE_LOBBY;

float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;

int sock = -1;
struct sockaddr_in server_addr;

void net_init() {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    #ifdef _WIN32
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif

    struct hostent *he = gethostbyname("s.farthq.com");
    if (he) {
        server_addr.sin_family = AF_INET;
        server_addr.sin_port = htons(5314); 
        memcpy(&server_addr.sin_addr, he->h_addr_list[0], he->h_length);
    } else {
        server_addr.sin_family = AF_INET;
        server_addr.sin_port = htons(6969);
        server_addr.sin_addr.s_addr = inet_addr("127.0.0.1");
    }

// Reuse Title Screen functions...
void draw_char(char c, float x, float y, float w, float h) {
    glBegin(GL_LINES);
    // (Simplified for HUD brevity - same vector font logic as previous phase)
    // Just drawing standard box for missing chars
    glVertex2f(x, y); glVertex2f(x+w, y);
    glVertex2f(x+w, y); glVertex2f(x+w, y+h);
    glVertex2f(x+w, y+h); glVertex2f(x, y+h);
    glVertex2f(x, y+h); glVertex2f(x, y);
    glEnd();
void draw_text_string(const char* s, float x, float y, float size) {
    float cursor = x;
    while(*s) {

        }
        draw_char(*s, cursor, y, size*0.6f, size);
        cursor += size * 0.8f;
        s++;
    }
void draw_lobby_screen() { /* Kept simple for now */ }

void draw_grid() {
    glBegin(GL_LINES);
    glColor3f(0.0f, 1.0f, 1.0f);
    for(int i=-100; i<=900; i+=5) {
        glVertex3f(i, 0, -100); glVertex3f(i, 0, 100);
        glVertex3f(-100, 0, i); glVertex3f(100, 0, i);
    }
    glEnd();

void draw_map() {
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        glPushMatrix();
        glTranslatef(b.x, b.y, b.z);
        glScalef(b.w, b.h, b.d);
        glBegin(GL_QUADS);
        glColor3f(0.2f, 0.2f, 0.2f); 
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
        glEnd();
        glLineWidth(2.0f);
        glColor3f(1.0f, 0.0f, 1.0f); 
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        glPopMatrix();
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

void draw_weapon_p(PlayerState *p) {
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

// --- NEW: DRAW HEAD ---
void draw_head(int weapon_id) {
    // Color matches gun
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); break;
    }
    
    glBegin(GL_QUADS);
    // Simple Cube
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(-0.4, 0, 0.4);
    glVertex3f(-0.4, 0.8, -0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(-0.4, 0, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(-0.4, 0, 0.4); glVertex3f(-0.4, 0, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(0.4, 0.8, -0.4);
    glEnd();

void draw_player_3rd(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y + 2.0f, p->z);
    glRotatef(-p->yaw, 0, 1, 0); 
    
    if(p->health <= 0) glColor3f(0.2, 0, 0); 
    else glColor3f(1, 0, 0); 
    
    glPushMatrix();
    glScalef(1.2f, 3.8f, 1.2f);
    glBegin(GL_QUADS);
    glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
    glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
    glEnd();
    glPopMatrix();
    
    // DRAW HEAD
    glPushMatrix();
    glTranslatef(0, 2.0f, 0); // On top of body
    draw_head(p->current_weapon);
    glPopMatrix();
    
    // Weapon
    glPushMatrix();
    glTranslatef(0.5f, 1.0f, 0.5f);
    glRotatef(p->pitch, 1, 0, 0);   
    glScalef(0.8f, 0.8f, 0.8f);
    draw_gun_model(p->current_weapon);
    glPopMatrix();
    glPopMatrix();

void draw_projectiles() {}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    
    // CROSSHAIR
    glColor3f(0, 1, 0);
    if (current_fov < 50.0f) {
        glBegin(GL_LINES); glVertex2f(0, 360); glVertex2f(1280, 360); glVertex2f(640, 0); glVertex2f(640, 720); glEnd();
    } else {
        glLineWidth(2.0f);
        glBegin(GL_LINES); glVertex2f(630, 360); glVertex2f(650, 360); glVertex2f(640, 350); glVertex2f(640, 370); glEnd();
    }
    
    // HIT MARKER
    if (p->hit_feedback > 0) {
        if (p->hit_feedback == 20) glColor3f(1, 0, 1); // MAGENTA (Headshot)
        else glColor3f(0, 1, 1); // CYAN (Body)
        
        glBegin(GL_TRIANGLE_FAN);
        glVertex2f(640, 360); 
        for(int i=0; i<=20; i++) {
            float ang = (float)i / 20.0f * 6.28318f;
            glVertex2f(640 + cosf(ang)*15, 360 + sinf(ang)*15);
        }
        glEnd();
    }

    // --- BARS (Bottom Left) ---
    // Health (Red)
    glColor3f(0.2f, 0, 0); glRectf(50, 50, 250, 70);
    glColor3f(1.0f, 0, 0); glRectf(50, 50, 50 + (p->health * 2), 70);
    
    // Shield (Blue)
    glColor3f(0, 0, 0.2f); glRectf(50, 80, 250, 100);
    glColor3f(0.2f, 0.2f, 1.0f); glRectf(50, 80, 50 + (p->shield * 2), 100);

    // --- AMMO (Bottom Right) ---
    int w = p->current_weapon;
    int ammo = (w == WPN_KNIFE) ? 99 : p->ammo[w];
    glColor3f(1, 1, 0);
    for(int i=0; i<ammo; i++) {
        glRectf(1200 - (i*10), 50, 1205 - (i*10), 80);
    }
    
    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();

    float cam_h = render_p->crouching ? 2.5f : EYE_HEIGHT;
    glRotatef(-cam_pitch, 1, 0, 0);
    glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);

    draw_grid();
    draw_map();
    draw_projectiles();
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        if(local_state.players[i].active) {
            draw_player_3rd(&local_state.players[i]);
        }
    }

    draw_weapon_p(render_p);
    draw_hud(render_p);

int main(int argc, char* argv[]) {

    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    local_init_match(1);

    int game_state = STATE_MENU;
    int game_mode = MODE_BOTS;
    int running = 1;

    while (running) {
        SDL_Event e;
        while (SDL_PollEvent(&e)) {
            if (e.type == SDL_QUIT) running = 0;
            if (game_state == STATE_PLAYING && e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) {
                game_state = STATE_MENU;
                SDL_SetRelativeMouseMode(SDL_FALSE);
            }
        }

        const Uint8 *k = SDL_GetKeyboardState(NULL);

        if (game_state == STATE_MENU) {
            glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
            glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
            glLoadIdentity();
            
            float glitch_x = (rand() % 100 > 95) ? (float)(rand() % 10 - 5) : 0;
            draw_text_string("SHANKPIT", 500 + glitch_x, 400, 4.0f);
            draw_text_string("[B] BOTS   [D] DEV/DEMO", 450, 300, 1.2f);
            draw_text_string("[S] SERVER [N] NETWORK", 450, 260, 1.2f);

            if (k[SDL_SCANCODE_B]) { game_mode = MODE_BOTS; game_state = STATE_PLAYING; local_init_match(31); SDL_SetRelativeMouseMode(SDL_TRUE); }
            if (k[SDL_SCANCODE_D]) { game_mode = MODE_DEV;  game_state = STATE_PLAYING; local_init_match(1);  SDL_SetRelativeMouseMode(SDL_TRUE); }
            if (k[SDL_SCANCODE_S]) { game_mode = MODE_SERV; game_state = STATE_PLAYING; local_init_match(31); USE_NEURAL_NET = 1; SDL_SetRelativeMouseMode(SDL_TRUE); }
            if (k[SDL_SCANCODE_N]) { game_mode = MODE_NET;  game_state = STATE_PLAYING; SDL_SetRelativeMouseMode(SDL_TRUE); }
            
            SDL_GL_SwapWindow(win);
            continue; 
        }

        // --- GAMEPLAY STATE ---
        if (game_state == STATE_PLAYING) {
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd+=1; if(k[SDL_SCANCODE_S]) fwd-=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            
            int jump = k[SDL_SCANCODE_SPACE];
            int crouch = k[SDL_SCANCODE_LCTRL];
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int reload = k[SDL_SCANCODE_R];

            local_update(fwd, str, cam_yaw, cam_pitch, shoot, -1, jump, crouch, reload);
            draw_scene(&local_state.players[0]);
            SDL_GL_SwapWindow(win);
        }
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
