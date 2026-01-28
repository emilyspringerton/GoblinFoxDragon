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
int my_client_id = -1;
float current_fov = 75.0f;
#define Z_FAR 8000.0f

void draw_char(char c, float x, float y, float s) {
    glLineWidth(2.0f); glBegin(GL_LINES); 
    if(c>='0'&&c<='9'){ glVertex2f(x,y+s);glVertex2f(x+s,y+s);glVertex2f(x+s,y);glVertex2f(x,y);glVertex2f(x,y+s); }
    else if(c=='A'){glVertex2f(x,y);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y);glVertex2f(x,y+s/2);glVertex2f(x+s/2,y+s);glVertex2f(x+s/2,y+s);glVertex2f(x+s,y+s/2);}
    else if(c=='B'){glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s,y+s*0.75);glVertex2f(x+s,y+s*0.75);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x+s,y+s/4);glVertex2f(x+s,y+s/4);glVertex2f(x+s*0.8,y);glVertex2f(x+s*0.8,y);glVertex2f(x,y);}
    else if(c=='D'){glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x,y);}
    else if(c=='J'){glVertex2f(x+s,y+s);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x,y);glVertex2f(x,y);glVertex2f(x,y+s/2);}
    else if(c=='S'){glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x,y);}
    else { glVertex2f(x,y);glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x+s,y); }
    glEnd();
}
void draw_string(const char* str, float x, float y, float size) { while(*str) { draw_char(*str, x, y, size); x += size * 1.5f; str++; } }

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT V1", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    
    local_init_match(1, 0); 
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if (app_state == STATE_LOBBY && e.type == SDL_KEYDOWN) {
                if (e.key.keysym.sym == SDLK_b) { app_state = STATE_GAME_LOCAL; local_init_match(8, MODE_DEATHMATCH); }
                if (e.key.keysym.sym == SDLK_d) { app_state = STATE_GAME_LOCAL; local_init_match(1, MODE_DEATHMATCH); }
                if (e.key.keysym.sym == SDLK_j) { app_state = STATE_GAME_NET; /* net_connect call here */ }
            }
            if (e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) app_state = STATE_LOBBY;
        }

        if (app_state == STATE_LOBBY) {
            glClearColor(0.02f, 0.02f, 0.05f, 1.0f); glClear(GL_COLOR_BUFFER_BIT);
            glLoadIdentity(); glColor3f(0, 1, 1);
            draw_string("SHANKPIT", 400, 500, 20);
            draw_string("B: BATTLE", 400, 350, 10);
            draw_string("D: DEMO", 400, 400, 10);
            SDL_GL_SwapWindow(win);
        } else {
            // Scene rendering logic for Battle/Demo goes here
            glClearColor(0.02f, 0.02f, 0.05f, 1.0f); glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
            // draw_scene(&local_state.players[0]);
            SDL_GL_SwapWindow(win);
        }
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
