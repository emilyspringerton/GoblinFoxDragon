#include <stdio.h>
#include <SDL2/SDL.h>
#include <GL/gl.h>
#include <GL/glu.h>
#include <math.h>

// --- WEAPON CONSTANTS ---
#define WPN_KNIFE 0
#define WPN_MAGNUM 1
#define WPN_AR 2
#define WPN_SHOTGUN 3
#define WPN_SNIPER 4

int current_weapon = WPN_MAGNUM;
float cam_x = 0.0f, cam_y = 2.0f, cam_z = 10.0f;
float cam_yaw = 0.0f;

// --- RENDER: RED ROCK CANYON ---
void draw_canyon() {
    // Floor (Neon Grid Style)
    glBegin(GL_LINES);
    glColor3f(1.0f, 0.2f, 0.0f); // Neon Orange/Red
    for(int i=-50; i<=50; i+=2) {
        glVertex3f(i, 0, -50); glVertex3f(i, 0, 50);
        glVertex3f(-50, 0, i); glVertex3f(50, 0, i);
    }
    glEnd();

    // Boulders (The "Slippery" Obstacles)
    for(int i=-5; i<5; i++) {
        glPushMatrix();
        glTranslatef(i*8.0f, 0, sinf(i)*15.0f);
        glScalef(3.0f, (sinf(i*3.14)*2.0f) + 5.0f, 3.0f);
        
        glBegin(GL_TRIANGLES);
        glColor3f(0.8f, 0.3f, 0.1f); // Rust
        glVertex3f(0, 1, 0); glVertex3f(-1, 0, 1); glVertex3f(1, 0, 1);
        glVertex3f(0, 1, 0); glVertex3f(1, 0, 1); glVertex3f(1, 0, -1);
        glVertex3f(0, 1, 0); glVertex3f(1, 0, -1); glVertex3f(-1, 0, -1);
        glVertex3f(0, 1, 0); glVertex3f(-1, 0, -1); glVertex3f(-1, 0, 1);
        glEnd();
        
        glPopMatrix();
    }
}

// --- RENDER: WEAPON ARSENAL ---
void draw_weapon_model() {
    glPushMatrix();
    glLoadIdentity(); // Draw HUD-style relative to camera
    
    // Position weapon in hand
    glTranslatef(0.4f, -0.4f, -1.2f); // Bottom Right
    glRotatef(-5.0f, 1, 0, 0); // Slight tilt

    switch(current_weapon) {
        case WPN_KNIFE:   
            glColor3f(0.9f, 0.9f, 0.9f); // Silver
            glScalef(0.05f, 0.05f, 0.4f); 
            break;
        case WPN_MAGNUM:  
            glColor3f(0.2f, 0.2f, 0.2f); // Gunmetal
            glScalef(0.1f, 0.15f, 0.3f); 
            break;
        case WPN_AR:      
            glColor3f(0.1f, 0.4f, 0.1f); // OD Green
            glScalef(0.12f, 0.15f, 0.8f); 
            break;
        case WPN_SHOTGUN: 
            glColor3f(0.5f, 0.3f, 0.1f); // Wood Furniture
            glScalef(0.15f, 0.15f, 0.6f); 
            break;
        case WPN_SNIPER:  
            glColor3f(0.1f, 0.1f, 0.1f); // Matte Black
            glScalef(0.08f, 0.12f, 1.4f); 
            break;
    }

    // Draw Weapon Cube
    glBegin(GL_QUADS);
        // Top
        glVertex3f(-1, 1, 1); glVertex3f(1, 1, 1); glVertex3f(1, 1, -1); glVertex3f(-1, 1, -1);
        // Bottom
        glVertex3f(-1, -1, 1); glVertex3f(-1, -1, -1); glVertex3f(1, -1, -1); glVertex3f(1, -1, 1);
        // Front
        glVertex3f(-1, -1, 1); glVertex3f(1, -1, 1); glVertex3f(1, 1, 1); glVertex3f(-1, 1, 1);
        // Back
        glVertex3f(-1, -1, -1); glVertex3f(-1, 1, -1); glVertex3f(1, 1, -1); glVertex3f(1, -1, -1);
        // Left
        glVertex3f(-1, -1, -1); glVertex3f(-1, -1, 1); glVertex3f(-1, 1, 1); glVertex3f(-1, 1, -1);
        // Right
        glVertex3f(1, -1, 1); glVertex3f(1, -1, -1); glVertex3f(1, 1, -1); glVertex3f(1, 1, 1);
    glEnd();

    glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT // NEON UPDATE", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    SDL_SetRelativeMouseMode(SDL_TRUE);

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
                if(e.key.keysym.sym == SDLK_ESCAPE) running = 0;
            }
            if(e.type == SDL_MOUSEMOTION) {
                cam_yaw += e.motion.xrel * 0.1f;
            }
        }

        // Camera Logic (Orbit/Look)
        float rad = cam_yaw * 0.0174533f;
        float lookX = sinf(rad);
        float lookZ = -cosf(rad);

        glClearColor(0.05f, 0.05f, 0.1f, 1.0f); // Dark Neon Sky
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
        glLoadIdentity();
        
        gluLookAt(cam_x, cam_y, cam_z, cam_x + lookX, cam_y, cam_z + lookZ, 0, 1, 0);

        draw_canyon();
        draw_weapon_model();

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
