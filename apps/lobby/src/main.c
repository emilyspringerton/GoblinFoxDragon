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

#define STATE_MAIN_MENU 0
#define STATE_B_MENU 1
#define STATE_PLAYING 2

int app_state = STATE_MAIN_MENU;
float cam_yaw = 0.0f;
float cam_pitch = 0.0f;

void draw_string(const char* str, float x, float y, float size) {
    glBegin(GL_LINES);
    while(*str) {
        glVertex2f(x, y); glVertex2f(x+size, y+size);
        glVertex2f(x, y+size); glVertex2f(x+size, y);
        x += size * 1.5f; str++;
    }
    glEnd();
}

void draw_map() {
    for(int i=0; i<map_count; i++) {
        Box b = map_geo[i];
        glPushMatrix();
        glTranslatef(b.x, b.y, b.z);
        glScalef(b.w, b.h, b.d);
        glColor3f(0.1f, 0.1f, 0.1f);
        glBegin(GL_QUADS);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glEnd();
        glColor3f(0, 1, 1);
        glLineWidth(2.0f);
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        glPopMatrix();
    }
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT HYDRATED", 100, 100, 1280, 720, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    SDL_GL_CreateContext(win);

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) {
                    if(app_state == STATE_PLAYING) app_state = STATE_B_MENU;
                    else if (app_state == STATE_B_MENU) { app_state = STATE_MAIN_MENU; SDL_SetRelativeMouseMode(SDL_FALSE); }
                }
                if(app_state == STATE_MAIN_MENU && e.key.keysym.sym == SDLK_b) app_state = STATE_B_MENU;
                if(app_state == STATE_B_MENU && e.key.keysym.sym == SDLK_p) {
                    app_state = STATE_PLAYING;
                    local_init_match(8, 0);
                    SDL_SetRelativeMouseMode(SDL_TRUE);
                }
            }
            if(e.type == SDL_MOUSEMOTION && app_state == STATE_PLAYING) {
                cam_yaw -= e.motion.xrel * 0.15f;
                cam_pitch -= e.motion.yrel * 0.15f;
            }
        }

        glClearColor(0.0f, 0.0f, 0.02f, 1.0f);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

        if (app_state == STATE_PLAYING) {
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 8000.0);
            glMatrixMode(GL_MODELVIEW); glLoadIdentity();
            PlayerState *p = &local_state.players[0];
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd = (k[SDL_SCANCODE_S]?1:0) - (k[SDL_SCANCODE_W]?1:0);
            float str = (k[SDL_SCANCODE_D]?1:0) - (k[SDL_SCANCODE_A]?1:0);
            local_update(fwd, str, cam_yaw, cam_pitch, 0, 0, k[SDL_SCANCODE_SPACE], k[SDL_SCANCODE_LCTRL], 0, NULL, 0);
            glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0);
            glTranslatef(-p->x, -(p->y + EYE_HEIGHT), -p->z);
            draw_map();
        } else {
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
            glMatrixMode(GL_MODELVIEW); glLoadIdentity();
            if(app_state == STATE_MAIN_MENU) { glColor3f(0, 1, 1); draw_string("SHANKPIT MAIN", 400, 500, 20); }
            else { glColor3f(1, 0, 0); draw_string("STAGING: HIT P", 400, 400, 15); }
        }
        SDL_GL_SwapWindow(win);
    }
    return 0;
}
