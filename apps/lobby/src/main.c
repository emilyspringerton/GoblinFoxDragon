#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <SDL2/SDL.h>
#include <SDL2/SDL_opengl.h>
#include <GL/glu.h>

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"
#include "../../../packages/simulation/local_game.h"

// APP STATES
#define STATE_MAIN_MENU 0
#define STATE_B_MENU 1
#define STATE_PLAYING 2

int app_state = STATE_MAIN_MENU;

// Simplified drawing from Phase 511
void draw_char(char c, float x, float y, float s) {
    glBegin(GL_LINES);
    glVertex2f(x, y); glVertex2f(x+s, y+s);
    glVertex2f(x, y+s); glVertex2f(x+s, y);
    glEnd();
}

void draw_string(const char* str, float x, float y, float size) {
    while(*str) { draw_char(*str, x, y, size); x += size * 1.5f; str++; }
}

void setup_2d() {
    int w, h;
    SDL_GL_GetDrawableSize(SDL_GL_GetCurrentWindow(), &w, &h);
    glViewport(0, 0, w, h);
    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW);
    glLoadIdentity();
    glDisable(GL_DEPTH_TEST);
}

void setup_3d() {
    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluPerspective(75.0, 1280.0/720.0, 0.1, 5000.0);
    glMatrixMode(GL_MODELVIEW);
    glLoadIdentity();
    glEnable(GL_DEPTH_TEST);
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT NAVIGATOR", 100, 100, 1280, 720, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    SDL_GL_CreateContext(win);
    SDL_GL_SetSwapInterval(1);

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN) {
                // ESCAPE LOGIC (Double Handshake)
                if(e.key.keysym.sym == SDLK_ESCAPE) {
                    if(app_state == STATE_PLAYING) {
                        app_state = STATE_B_MENU;
                        // Mouse stays locked as per requirement
                    } else if (app_state == STATE_B_MENU) {
                        app_state = STATE_MAIN_MENU;
                        SDL_SetRelativeMouseMode(SDL_FALSE); // Mouse released
                    }
                }

                // NAVIGATION
                if(app_state == STATE_MAIN_MENU && e.key.keysym.sym == SDLK_b) {
                    app_state = STATE_B_MENU;
                }
                
                if(app_state == STATE_B_MENU && e.key.keysym.sym == SDLK_p) {
                    app_state = STATE_PLAYING;
                    local_init_match(8, 0); // Init bots
                    SDL_SetRelativeMouseMode(SDL_TRUE);
                }
            }
        }

        // RENDER LOGIC
        glClearColor(0.01f, 0.01f, 0.02f, 1.0f);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

        if (app_state == STATE_MAIN_MENU) {
            setup_2d();
            glColor3f(0, 1, 1); // CYAN
            draw_string("SHANKPIT MAIN", 400, 500, 20);
            draw_string("PRESS B FOR STAGING", 400, 400, 10);
        } 
        else if (app_state == STATE_B_MENU) {
            setup_2d();
            glColor3f(1, 0, 0); // RED
            draw_string("STAGING AREA", 400, 500, 20);
            draw_string("HIT P TO PLAY", 400, 400, 15);
            draw_string("ESC TO BACK", 400, 300, 8);
        }
        else if (app_state == STATE_PLAYING) {
            // BASIC INFINITY LEVEL
            setup_3d();
            
            // Movement Update
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd-=1; if(k[SDL_SCANCODE_S]) fwd+=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            local_update(fwd, str, 0, 0, 0, 0, 0, 0, 0, NULL, 0);

            // Draw Infinite Grid
            glBegin(GL_LINES);
            glColor3f(0.2f, 0.2f, 0.5f);
            for(int i=-1000; i<=1000; i+=50) {
                glVertex3f(i, 0, -1000); glVertex3f(i, 0, 1000);
                glVertex3f(-1000, 0, i); glVertex3f(1000, 0, i);
            }
            glEnd();

            setup_2d();
            glColor3f(1, 1, 1);
            draw_string("LEVEL INFINITY", 50, 650, 10);
            draw_string("ESC TO STAGING", 50, 600, 8);
        }

        SDL_GL_SwapWindow(win);
    }
    return 0;
}
