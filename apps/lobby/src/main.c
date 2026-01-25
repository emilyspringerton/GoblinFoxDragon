#include <stdio.h>
#include <SDL2/SDL.h>
#include <GL/gl.h>
#include <GL/glu.h>
#include <math.h>
#include <string.h>

// Stable v95 Weapon Definitions
#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

int current_weapon = WPN_MAGNUM;

void draw_weapon_model() {
    glPushMatrix();
    glTranslatef(0.5f, -0.5f, -1.0f);
    switch(current_weapon) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.8f); glScalef(0.1f, 0.1f, 0.4f); break;
        case WPN_MAGNUM:  glColor3f(0.3f, 0.3f, 0.3f); glScalef(0.15f, 0.2f, 0.3f); break;
        case WPN_AR:      glColor3f(0.2f, 0.4f, 0.2f); glScalef(0.15f, 0.2f, 0.8f); break;
        case WPN_SHOTGUN: glColor3f(0.4f, 0.3f, 0.2f); glScalef(0.2f, 0.25f, 0.6f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.1f); glScalef(0.1f, 0.15f, 1.2f); break;
    }
    // Draw simple box for weapon
    glBegin(GL_QUADS);
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1);
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1);
    glEnd();
    glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT // STABLE v95", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    // SDL_SetRelativeMouseMode(SDL_TRUE); // Uncomment for FPS lock

    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluPerspective(70.0, 1280.0/720.0, 0.1, 1000.0);
    glMatrixMode(GL_MODELVIEW);
    glEnable(GL_DEPTH_TEST);

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_1) current_weapon = WPN_KNIFE;
                if(e.key.keysym.sym == SDLK_2) current_weapon = WPN_MAGNUM;
                if(e.key.keysym.sym == SDLK_3) current_weapon = WPN_AR;
                if(e.key.keysym.sym == SDLK_4) current_weapon = WPN_SHOTGUN;
                if(e.key.keysym.sym == SDLK_5) current_weapon = WPN_SNIPER;
            }
        }

        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
        glLoadIdentity();
        
        // Simple Camera Logic could go here
        
        draw_weapon_model();

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
