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

char SERVER_HOST[64] = "s.farthq.com";
int SERVER_PORT = 6969;
int app_state = STATE_LOBBY;
int wpn_req = 1; 
int my_client_id = -1;
float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;
#define Z_FAR 8000.0f

#define MENU_MAIN 0
#define MENU_LAN_BROWSER 1
int menu_state = MENU_MAIN;
int menu_selection = 0;

void draw_char(char c, float x, float y, float s) {
    glLineWidth(1.5f); glBegin(GL_LINES);
    if(c >= '0' && c <= '9'){ glVertex2f(x,y); glVertex2f(x+s,y); glVertex2f(x+s,y); glVertex2f(x+s,y+s); glVertex2f(x+s,y+s); glVertex2f(x,y+s); glVertex2f(x,y+s); glVertex2f(x,y); }
    else if(c == 'A'){ glVertex2f(x,y); glVertex2f(x+s/2,y+s); glVertex2f(x+s/2,y+s); glVertex2f(x+s,y); glVertex2f(x+s/4,y+s/2); glVertex2f(x+3*s/4,y+s/2); }
    else if(c == 'B'){ glVertex2f(x,y); glVertex2f(x,y+s); glVertex2f(x,y+s); glVertex2f(x+s,y+s*0.75); glVertex2f(x+s,y+s*0.75); glVertex2f(x,y+s/2); glVertex2f(x,y+s/2); glVertex2f(x+s,y+s*0.25); glVertex2f(x+s,y+s*0.25); glVertex2f(x,y); }
    else if(c == 'D'){ glVertex2f(x,y); glVertex2f(x,y+s); glVertex2f(x,y+s); glVertex2f(x+s,y+s/2); glVertex2f(x+s,y+s/2); glVertex2f(x,y); }
    else if(c == 'J'){ glVertex2f(x,y+s/4); glVertex2f(x+s/2,y); glVertex2f(x+s/2,y); glVertex2f(x+s,y+s); }
    else if(c == 'N'){ glVertex2f(x,y); glVertex2f(x,y+s); glVertex2f(x,y+s); glVertex2f(x+s,y); glVertex2f(x+s,y); glVertex2f(x+s,y+s); }
    else if(c == 'S'){ glVertex2f(x+s,y+s); glVertex2f(x,y+s); glVertex2f(x,y+s); glVertex2f(x,y+s/2); glVertex2f(x,y+s/2); glVertex2f(x+s,y+s/2); glVertex2f(x+s,y+s/2); glVertex2f(x+s,y); glVertex2f(x+s,y); glVertex2f(x,y); }
    else if(c == 'E'){ glVertex2f(x+s,y); glVertex2f(x,y); glVertex2f(x,y); glVertex2f(x,y+s); glVertex2f(x,y+s); glVertex2f(x+s,y+s); glVertex2f(x,y+s/2); glVertex2f(x+s*0.7f,y+s/2); } // FIXED ARGUMENT
    else if(c == 'T'){ glVertex2f(x+s/2,y); glVertex2f(x+s/2,y+s); glVertex2f(x,y+s); glVertex2f(x+s,y+s); }
    else if(c == 'U'){ glVertex2f(x,y+s); glVertex2f(x,y); glVertex2f(x,y); glVertex2f(x+s,y); glVertex2f(x+s,y); glVertex2f(x+s,y+s); }
    else if(c == ':'){ glVertex2f(x+s/2,y+s*0.3); glVertex2f(x+s/2,y+s*0.35); glVertex2f(x+s/2,y+s*0.65); glVertex2f(x+s/2,y+s*0.7); }
    else { glVertex2f(x,y); glVertex2f(x+s,y+s); glVertex2f(x,y+s); glVertex2f(x+s,y); }
    glEnd();
}

void draw_string(const char* str, float x, float y, float size) { while(*str) { draw_char(*str, x, y, size); x += size * 1.5f; str++; } }

void draw_menu() {
    glClearColor(0.02f, 0.02f, 0.05f, 1.0f); glClear(GL_COLOR_BUFFER_BIT);
    glLoadIdentity(); glColor3f(0, 1, 1);
    draw_string("SHANKPIT NAVIGATOR", 400, 600, 15);
    if (menu_state == MENU_MAIN) {
        const char* opts[] = {"BOT BATTLE", "MICRO DEMO", "FIND LAN GAMES", "JOIN GLOBAL"};
        for(int i=0; i<4; i++) {
            if (i == menu_selection) glColor3f(1, 1, 0); else glColor3f(0, 0.5f, 0.5f);
            draw_string(opts[i], 400, 450 - (i * 50), 10);
        }
    }
}

// ... Additional Rendering Helpers (draw_grid, draw_map, etc.) would be here ...

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if (app_state == STATE_LOBBY) {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_UP) menu_selection = (menu_selection - 1 + 4) % 4;
                    if (e.key.keysym.sym == SDLK_DOWN) menu_selection = (menu_selection + 1) % 4;
                    if (e.key.keysym.sym == SDLK_RETURN) {
                        if (menu_selection == 0) { app_state = STATE_GAME_LOCAL; local_init_match(12, MODE_DEATHMATCH); }
                        if (menu_selection == 1) { app_state = STATE_GAME_LOCAL; local_init_match(8, MODE_DEATHMATCH); }
                        if (menu_selection == 3) { app_state = STATE_GAME_NET; }
                    }
                }
            } else if (e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) {
                app_state = STATE_LOBBY;
            }
        }
        if (app_state == STATE_LOBBY) {
            draw_menu();
        } else {
            // ACOG Logic
            float target_fov = 75.0f;
            int rmb = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_RIGHT));
            if (rmb) {
                if (local_state.players[0].current_weapon == WPN_SNIPER) target_fov = 20.0f;
                else if (local_state.players[0].current_weapon == WPN_AR) target_fov = 45.0f;
            }
            current_fov += (target_fov - current_fov) * 0.2f;
            // Scene drawing code goes here...
        }
        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    return 0;
}
