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
#endif

#include <SDL2/SDL.h>
#include <SDL2/SDL_opengl.h>
#include <GL/glu.h>

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"
#include "../../../packages/simulation/local_game.h"

// --- APP STATES ---
#define STATE_LOBBY 0
#define STATE_GAME_NET 1
#define STATE_GAME_LOCAL 2
int app_state = STATE_LOBBY;

// --- GLOBALS ---
float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
int sock = -1;

void net_init() {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sock = socket(AF_INET, SOCK_DGRAM, 0);
}

// --- VISUALS ---
void draw_char(float x, float y, float size, char c) {
    glPushMatrix(); glTranslatef(x, y, 0); glScalef(size, size, 1);
    glBegin(GL_LINES);
    // Minimal Font
    if(c=='A'){ glVertex2f(0,0); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(1,0); glVertex2f(0,1); glVertex2f(1,1); }
    if(c=='D'){ glVertex2f(0,0); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,0); }
    if(c=='H'){ glVertex2f(0,0); glVertex2f(0,2); glVertex2f(1,0); glVertex2f(1,2); glVertex2f(0,1); glVertex2f(1,1); }
    if(c=='P'){ glVertex2f(0,0); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,1); }
    if(c=='K'){ glVertex2f(0,0); glVertex2f(0,2); glVertex2f(1,2); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,0); }
    if(c=='0'){ glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(0,0); }
    if(c=='1'){ glVertex2f(0.5,0); glVertex2f(0.5,2); }
    if(c=='2'){ glVertex2f(0,2); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='3'){ glVertex2f(0,2); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0,1); glVertex2f(1,1); }
    if(c=='4'){ glVertex2f(0,2); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,2); glVertex2f(1,0); }
    if(c=='5'){ glVertex2f(1,2); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); }
    if(c=='6'){ glVertex2f(1,2); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,1); }
    if(c=='/'){ glVertex2f(0,0); glVertex2f(1,2); }
    glEnd(); glPopMatrix();
}

void draw_text(float x, float y, float size, const char* str) {
    float cur = x;
    while(*str) { draw_char(cur, y, size, *str); cur += size * 1.5f; str++; }
}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    
    // Crosshair
    glLineWidth(2.0f);
    glColor3f(0, 1, 0);
    glBegin(GL_LINES); 
    glVertex2f(640-10, 360); glVertex2f(640+10, 360); 
    glVertex2f(640, 360-10); glVertex2f(640, 360+10); 
    glEnd();

    // Stats
    char buf[64];
    glColor3f(1, 0, 0);
    sprintf(buf, "HP %d", p->health);
    draw_text(50, 50, 30.0f, buf);

    glColor3f(1, 1, 0);
    int cur = p->ammo[p->current_weapon];
    int max = WPN_STATS[p->current_weapon].ammo_max;
    if (p->current_weapon == WPN_KNIFE) sprintf(buf, "KNIFE");
    else sprintf(buf, "%d/%d", cur, max);
    draw_text(1000, 50, 30.0f, buf);

    if (p->reload_timer > 0) {
        glColor3f(1, 0.5, 0);
        draw_text(550, 300, 20.0f, "RELOADING");
    }

    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPopMatrix(); 
    glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_grid() {
    glBegin(GL_LINES);
    glColor3f(0.0f, 1.0f, 1.0f);
    for(int i=-100; i<=100; i+=5) {
        glVertex3f(i, 0, -100); glVertex3f(i, 0, 100);
        glVertex3f(-100, 0, i); glVertex3f(100, 0, i);
    }
    glEnd();
}

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
}

void draw_box_mesh() {
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
}

void draw_weapon_p(PlayerState *p) {
    glPushMatrix();
    glLoadIdentity();
    
    // Recoil + Bob
    float kick = p->recoil_anim * 0.2f;
    float reload_dip = (p->reload_timer > 0) ? sinf(p->reload_timer * 0.2f) * 0.5f - 0.5f : 0.0f;
    glTranslatef(0.4f, -0.5f + kick + reload_dip, -1.2f + (kick * 0.5f));
    glRotatef(-p->recoil_anim * 10.0f, 1, 0, 0);
    
    // Base Color & Shape
    switch(p->current_weapon) {
        case WPN_KNIFE:   
            glColor3f(0.8f, 0.8f, 0.8f); 
            glScalef(0.05f, 0.05f, 0.4f); // Thin blade
            break;
        case WPN_MAGNUM:  
            glColor3f(0.6f, 0.6f, 0.6f); 
            glScalef(0.1f, 0.15f, 0.4f); // Thick barrel
            break;
        case WPN_AR:      
            glColor3f(0.2f, 0.2f, 0.2f); 
            glScalef(0.12f, 0.15f, 0.8f); // Long body
            break;
        case WPN_SHOTGUN: 
            glColor3f(0.4f, 0.2f, 0.1f); 
            glScalef(0.15f, 0.15f, 0.6f); // Short & Fat
            break;
        case WPN_SNIPER:  
            glColor3f(0.1f, 0.15f, 0.1f); 
            glScalef(0.08f, 0.1f, 1.2f); // Very Long
            break;
    }
    draw_box_mesh();
    
    // EXTRA DETAILS (Scope/Mag)
    if (p->current_weapon == WPN_SNIPER) {
        // Scope
        glPushMatrix();
        glTranslatef(0, 1.2f, -0.2f);
        glScalef(1.2f, 0.5f, 0.3f);
        glColor3f(0, 0, 0);
        draw_box_mesh();
        glPopMatrix();
    }
    if (p->current_weapon == WPN_AR) {
        // Mag
        glPushMatrix();
        glTranslatef(0, -1.0f, 0.2f);
        glScalef(0.8f, 1.5f, 0.3f);
        glColor3f(0.1f, 0.1f, 0.1f);
        draw_box_mesh();
        glPopMatrix();
    }

    glPopMatrix();
}

void draw_bot_model(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y, p->z);
    glRotatef(p->yaw, 0, 1, 0); 
    
    // Hurt Flash
    if (p->health < 100) glColor3f(1.0f, 0.0f, 0.0f);
    else glColor3f(0.0f, 1.0f, 0.0f);

    glScalef(1.0f, 4.0f, 1.0f);
    draw_box_mesh();
    glPopMatrix();
}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
    if (render_p->hit_feedback > 0) glClearColor(0.2f, 0.0f, 0.0f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();

    float cam_h = render_p->crouching ? 2.5f : 6.0f;
    glRotatef(-cam_pitch, 1, 0, 0);
    glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);

    draw_grid();
    draw_map();
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        if(state.players[i].active) {
            draw_bot_model(&state.players[i]);
        }
    }

    draw_weapon_p(render_p);
    draw_hud(render_p);
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT HYBRID", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    local_init();

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            
            if (app_state == STATE_LOBBY) {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_d) {
                        app_state = STATE_GAME_LOCAL;
                        local_init(); 
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                }
            } 
            else {
                if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) {
                    app_state = STATE_LOBBY;
                    SDL_SetRelativeMouseMode(SDL_FALSE);
                    glDisable(GL_DEPTH_TEST);
                    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
                    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
                }
                if(e.type == SDL_MOUSEMOTION) {
                    // MOUSE FIX: SUBTRACTION (Corrects inversion)
                    cam_yaw -= e.motion.xrel * 0.15f;
                    if(cam_yaw > 360) cam_yaw -= 360; 
                    if(cam_yaw < 0) cam_yaw += 360;
                    
                    cam_pitch -= e.motion.yrel * 0.15f;
                    if(cam_pitch > 89) cam_pitch = 89; 
                    if(cam_pitch < -89) cam_pitch = -89;
                }
            }
        }

        if (app_state == STATE_LOBBY) {
            glClearColor(0.1f, 0.1f, 0.2f, 1.0f);
            glClear(GL_COLOR_BUFFER_BIT);
            glLoadIdentity();
            glColor3f(1, 1, 0);
            // Draw 'D'
            glBegin(GL_LINES); 
            glVertex2f(600, 300); glVertex2f(600, 400); glVertex2f(600, 400); glVertex2f(650, 350);
            glVertex2f(650, 350); glVertex2f(600, 300);
            glEnd();
        } 
        else if (app_state == STATE_GAME_LOCAL) {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd+=1; if(k[SDL_SCANCODE_S]) fwd-=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            int jump = k[SDL_SCANCODE_SPACE];
            int crouch = k[SDL_SCANCODE_LCTRL];
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int reload = k[SDL_SCANCODE_R];
            int wpn = -1;
            if(k[SDL_SCANCODE_1]) wpn=0; if(k[SDL_SCANCODE_2]) wpn=1; if(k[SDL_SCANCODE_3]) wpn=2;
            if(k[SDL_SCANCODE_4]) wpn=3; if(k[SDL_SCANCODE_5]) wpn=4;

            local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn, jump, crouch, reload);
            draw_scene(&state.players[0]);
        }

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
