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

int app_state = 0; // STATE_LOBBY
int menu_selection = 0;

void draw_char(char c, float x, float y, float s) {
    glBegin(GL_LINES);
    glVertex2f(x, y); glVertex2f(x+s, y+s);
    glVertex2f(x, y+s); glVertex2f(x+s, y);
    glEnd();
}

void draw_string(const char* str, float x, float y, float size) {
    while(*str) { draw_char(*str, x, y, size); x += size * 1.5f; str++; }
}

void setup_2d_menu() {
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

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT V1", 100, 100, 1280, 720, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    SDL_GL_CreateContext(win);
    SDL_GL_SetSwapInterval(1);

    local_init_match(1, 0);

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) app_state = 0;
                if(e.key.keysym.sym == SDLK_b) { app_state = 1; local_init_match(8, 0); }
            }
        }

        // --- HARDENED RENDER PATH ---
        glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

        if (app_state == 0) {
            setup_2d_menu();
            glColor3f(0, 1, 1);
            draw_string("SHANKPIT V1", 400, 500, 20);
            draw_string("PRESS B FOR BATTLE", 400, 400, 10);
        } else {
            // Battle logic would go here
            setup_2d_menu();
            glColor3f(1, 0, 0);
            draw_string("IN BATTLE - ESC TO EXIT", 400, 500, 15);
        }

        SDL_GL_SwapWindow(win);
        SDL_Delay(1);
    }
    return 0;
}
