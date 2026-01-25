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
char chat_buf[64] = {0};

// --- NETWORKING GLOBALS ---
int sock = -1;
struct sockaddr_in server_addr;

void net_init() {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    
    // Non-blocking mode
    #ifdef _WIN32
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif

    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(6969); // Port 6969
    
    // Resolve DNS manually if needed, for now hardcode or localhost
    // For cloud build, this would ideally be 's.farthq.com'
    // But simplified here for compilation safety
    server_addr.sin_addr.s_addr = inet_addr("127.0.0.1"); 
}

// --- RENDER HELPERS (Vector Font) ---
void draw_char(float x, float y, float size, char c) {
    glPushMatrix(); glTranslatef(x, y, 0); glScalef(size, size, 1);
    glBegin(GL_LINES);
    switch(c) {
        case 'A': glVertex2f(0,0); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(1,0); glVertex2f(0,1); glVertex2f(1,1); break;
        case 'D': glVertex2f(0,0); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,0); break;
        case '[': glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(1,2); break;
        case ']': glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(0,2); break;
        case '0': glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,2); glVertex2f(1,2); glVertex2f(0,2); glVertex2f(0,2); glVertex2f(0,0); break;
        case '1': glVertex2f(0.5,0); glVertex2f(0.5,2); break;
        // ... (Simplified set for menu)
    }
    glEnd(); glPopMatrix();
}

void draw_text_simple(float x, float y, float size, const char* str) {
    float cur = x;
    while(*str) { draw_char(cur, y, size, *str); cur += size * 1.5f; str++; }
}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();

    // Camera
    float cam_h = render_p->crouching ? 2.5f : 6.0f;
    glRotatef(-cam_pitch, 1, 0, 0);
    glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);

    // World
    draw_grid();
    draw_map();
    draw_weapon(); // Uses p.recoil_anim inside physics.h, need to update that to take args? 
                   // Ideally physics.h functions should take *p, but draw_weapon in client used global 'p'
                   // For now, we rely on the global 'p' in client from previous step, OR we update client to use local_p
}

// To fix the "draw_weapon" dependency, we redefine it here to take a PlayerState
void draw_weapon_p(PlayerState *p) {
    glPushMatrix();
    glLoadIdentity();
    float kick = p->recoil_anim * 0.2f;
    float reload_dip = (p->reload_timer > 0) ? sinf(p->reload_timer * 0.2f) * 0.5f - 0.5f : 0.0f;
    glTranslatef(0.4f, -0.5f + kick + reload_dip, -1.2f + (kick * 0.5f));
    glRotatef(-p->recoil_anim * 10.0f, 1, 0, 0);
    
    // Simple colored box
    switch(p->current_weapon) {
        case WPN_KNIFE: glColor3f(0.8f,0.8f,0.8f); break;
        case WPN_MAGNUM: glColor3f(0.6f,0.6f,0.6f); break;
        case WPN_AR: glColor3f(0.2f,0.2f,0.2f); break;
        case WPN_SHOTGUN: glColor3f(0.4f,0.2f,0.1f); break;
        case WPN_SNIPER: glColor3f(0.1f,0.1f,0.1f); break;
    }
    glScalef(0.15f, 0.2f, 0.8f);
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); // Top
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); // Bot
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); // Back
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); // Right
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); // Left
    glEnd();
    glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT HYBRID", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    
    // Init Local
    local_init();

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            
            // --- LOBBY INPUTS ---
            if (app_state == STATE_LOBBY) {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_a) {
                        app_state = STATE_GAME_NET;
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                    if (e.key.keysym.sym == SDLK_d) {
                        app_state = STATE_GAME_LOCAL;
                        local_init(); // Reset
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                }
            } 
            // --- GAME INPUTS ---
            else {
                if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) {
                    app_state = STATE_LOBBY;
                    SDL_SetRelativeMouseMode(SDL_FALSE);
                    glDisable(GL_DEPTH_TEST);
                    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
                    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
                }
                if(e.type == SDL_MOUSEMOTION) {
                    cam_yaw += e.motion.xrel * 0.15f;
                    // Normalize
                    if(cam_yaw > 360) cam_yaw -= 360; if(cam_yaw < 0) cam_yaw += 360;
                    cam_pitch += e.motion.yrel * 0.15f;
                    if(cam_pitch > 89) cam_pitch = 89; if(cam_pitch < -89) cam_pitch = -89;
                }
            }
        }

        // --- LOOP ---
        if (app_state == STATE_LOBBY) {
            glClearColor(0.1f, 0.1f, 0.2f, 1.0f);
            glClear(GL_COLOR_BUFFER_BIT);
            glLoadIdentity();
            glColor3f(1, 1, 0);
            // Draw crude menu text
            glBegin(GL_LINES); 
            // A
            glVertex2f(100, 300); glVertex2f(100, 400); glVertex2f(100, 400); glVertex2f(150, 400);
            glVertex2f(150, 400); glVertex2f(150, 300); glVertex2f(100, 350); glVertex2f(150, 350);
            // D
            glVertex2f(200, 300); glVertex2f(200, 400); glVertex2f(200, 400); glVertex2f(250, 350);
            glVertex2f(250, 350); glVertex2f(200, 300);
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

            // Run Local Sim
            local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn, jump, crouch, reload);

            // Render
            glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
            glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
            glLoadIdentity();
            float cam_h = local_p.crouching ? 2.5f : 6.0f;
            glRotatef(-cam_pitch, 1, 0, 0);
            glRotatef(-cam_yaw, 0, 1, 0);
            glTranslatef(-local_p.x, -(local_p.y + cam_h), -local_p.z);
            
            draw_grid();
            draw_map();
            draw_weapon_p(&local_p);
        }
        else if (app_state == STATE_GAME_NET) {
            // Placeholder for Net code (To be fully implemented)
            glClearColor(0, 0, 0, 1); glClear(GL_COLOR_BUFFER_BIT);
        }

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
