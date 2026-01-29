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

// --- GOLDEN MENU STATES ---
#define STATE_MAIN_MENU 0
#define STATE_B_MENU 1
#define STATE_PLAYING 2

int app_state = STATE_MAIN_MENU;
float cam_yaw = 0.0f;
float cam_pitch = 0.0f;

// Basic text renderer
void draw_string(const char* str, float x, float y, float size) {
    glLineWidth(2.0f); glBegin(GL_LINES);
    while(*str) {
        glVertex2f(x, y); glVertex2f(x+size, y+size);
        glVertex2f(x, y+size); glVertex2f(x+size, y);
        x += size * 1.5f; str++;
    }
    glEnd();
}

void draw_projectiles() {
    glPointSize(5.0f); glBegin(GL_POINTS);
    glColor3f(1.0f, 0.2f, 0.2f); // Red Storm Arrows
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if(local_state.projectiles[i].active) {
            glVertex3f(local_state.projectiles[i].x, local_state.projectiles[i].y, local_state.projectiles[i].z);
        }
    }
    glEnd();
}

void draw_scene() {
    glClearColor(0.02f, 0.02f, 0.05f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 8000.0);
    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
    
    PlayerState *p = &local_state.players[0];
    glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-p->x, -(p->y + EYE_HEIGHT), -p->z);
    
    // Draw Map (Placeholder Box)
    glColor3f(0.2f, 0.2f, 0.2f);
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        glPushMatrix(); glTranslatef(b.x, b.y, b.z); glScalef(b.w, b.h, b.d);
        // Simple Cube
        glBegin(GL_LINES);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5);
        glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5);
        glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,0.5,-0.5); glVertex3f(-0.5,-0.5,-0.5);
        glEnd();
        glPopMatrix();
    }
    
    // Draw Floor
    glBegin(GL_LINES); glColor3f(0, 1, 1);
    for(int i=-1000; i<=1000; i+=100) {
        glVertex3f(i, 0, -1000); glVertex3f(i, 0, 1000);
        glVertex3f(-1000, 0, i); glVertex3f(1000, 0, i);
    }
    glEnd();
    
    draw_projectiles();
    
    // HUD
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
    glDisable(GL_DEPTH_TEST);
    
    if (p->storm_charges > 0) {
        glColor3f(1.0f, 0.2f, 0.2f); // Storm Active Color
        char buf[32]; sprintf(buf, "STORM ARROWS: %d", p->storm_charges);
        draw_string(buf, 550, 400, 15);
    } else {
        glColor3f(0, 1, 0);
        if (p->current_weapon == WPN_SNIPER) draw_string("BOW READY", 580, 50, 10);
        else draw_string("WEAPON", 580, 50, 10);
    }
    glEnable(GL_DEPTH_TEST);
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT STORM", 100, 100, 1280, 720, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    SDL_GL_CreateContext(win);
    
    local_init_match(1, 0);
    
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_MOUSEMOTION && app_state == STATE_PLAYING) {
                cam_yaw += e.motion.xrel * 0.15f; // Note: Sign flip might be needed depending on coord system
                cam_pitch += e.motion.yrel * 0.15f;
            }
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) {
                    if(app_state == STATE_PLAYING) app_state = STATE_B_MENU;
                    else if(app_state == STATE_B_MENU) { app_state = STATE_MAIN_MENU; SDL_SetRelativeMouseMode(SDL_FALSE); }
                }
                if(app_state == STATE_MAIN_MENU && e.key.keysym.sym == SDLK_b) app_state = STATE_B_MENU;
                if(app_state == STATE_B_MENU && e.key.keysym.sym == SDLK_p) {
                    app_state = STATE_PLAYING;
                    local_init_match(1, 0);
                    SDL_SetRelativeMouseMode(SDL_TRUE);
                }
            }
        }
        
        if (app_state == STATE_PLAYING) {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd = (k[SDL_SCANCODE_S]?1:0) - (k[SDL_SCANCODE_W]?1:0); // Inverted Z
            float str = (k[SDL_SCANCODE_D]?1:0) - (k[SDL_SCANCODE_A]?1:0);
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int ability = k[SDL_SCANCODE_E];
            
            local_update(fwd, str, cam_yaw, cam_pitch, shoot, -1, k[SDL_SCANCODE_SPACE], 0, 0, ability, NULL, 0);
            draw_scene();
        } else {
            glClearColor(0,0,0,1); glClear(GL_COLOR_BUFFER_BIT);
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
            glMatrixMode(GL_MODELVIEW); glLoadIdentity();
            glColor3f(0,1,1);
            if(app_state == STATE_MAIN_MENU) draw_string("MAIN MENU", 500, 500, 20);
            else { glColor3f(1,0,0); draw_string("STAGING", 500, 500, 20); }
        }
        
        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    return 0;
}
