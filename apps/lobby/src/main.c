#include <stdio.h>
#include <SDL2/SDL.h>
#include <GL/gl.h>
#include <GL/glu.h>
#include <math.h>
#include "protocol.h"
#include "physics.h"

// Init Player High in the air
PlayerState p = {0, 10.0f, 0, 0, 0, 0, 0, 0, 1, 100, 0};

void draw_grid() {
    glBegin(GL_LINES);
    glColor3f(0.0f, 1.0f, 1.0f); // Cyan
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
        glColor3f(1.0f, 0.0f, 1.0f); // Magenta
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        
        glPopMatrix();
    }
}

void draw_weapon() {
    glPushMatrix();
    glLoadIdentity();
    
    float kick = p.recoil_anim * 0.2f;
    glTranslatef(0.4f, -0.5f + kick, -1.2f + (kick * 0.5f));
    glRotatef(-p.recoil_anim * 10.0f, 1, 0, 0);

    switch(p.current_weapon) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.8f); glScalef(0.1f, 0.1f, 0.5f); break;
        case WPN_MAGNUM:  glColor3f(0.6f, 0.6f, 0.6f); glScalef(0.15f, 0.2f, 0.4f); break;
        case WPN_AR:      glColor3f(0.1f, 0.1f, 0.1f); glScalef(0.15f, 0.2f, 1.0f); break;
        case WPN_SHOTGUN: glColor3f(0.4f, 0.2f, 0.1f); glScalef(0.2f, 0.2f, 0.8f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.2f, 0.1f); glScalef(0.1f, 0.15f, 1.5f); break;
    }

    glBegin(GL_QUADS);
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1);
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1);
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1);
    glVertex3f(-1,-1,-1); glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1);
    glVertex3f(1,-1,1); glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1);
    glEnd();
    
    glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT // CALIBRATED", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    SDL_SetRelativeMouseMode(SDL_TRUE);
    
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
    glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);

    int running = 1;
    int space_held = 0;

    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) running = 0;
                if(e.key.keysym.sym >= SDLK_1 && e.key.keysym.sym <= SDLK_5) 
                    p.current_weapon = e.key.keysym.sym - SDLK_0;
                if(e.key.keysym.sym == SDLK_SPACE) space_held = 1;
                if(e.key.keysym.sym == SDLK_LCTRL) p.crouching = 1;
            }
            if(e.type == SDL_KEYUP) {
                if(e.key.keysym.sym == SDLK_SPACE) space_held = 0;
                if(e.key.keysym.sym == SDLK_LCTRL) p.crouching = 0;
            }
            if(e.type == SDL_MOUSEMOTION) {
                // SENSITIVITY
                p.yaw += e.motion.xrel * 0.15f;
                p.pitch += e.motion.yrel * 0.15f;
                
                // NORMALIZE ANGLES
                if (p.yaw >= 360.0f) p.yaw -= 360.0f;
                if (p.yaw < 0.0f) p.yaw += 360.0f;

                if(p.pitch > 89.0f) p.pitch = 89.0f;
                if(p.pitch < -89.0f) p.pitch = -89.0f;
            }
            if(e.type == SDL_MOUSEBUTTONDOWN) p.recoil_anim = 1.0f;
        }

        // --- PHYSICS STEP ---
        apply_friction(&p);

        const Uint8 *k = SDL_GetKeyboardState(NULL);
        
        // --- VECTOR FIX: REMOVE NEGATIVE SIGN ---
        // Positive Yaw = Right Rotation.
        // sin/cos logic:
        // yaw=0  -> fwd=(0,-1) (-Z)
        // yaw=90 -> fwd=(1, 0) (+X)
        float rad = p.yaw * 0.0174533f; // NO NEGATIVE SIGN HERE
        
        float fwd_x = sinf(rad);
        float fwd_z = -cosf(rad);
        float right_x = cosf(rad);
        float right_z = sinf(rad);
        
        float wish_x = 0; float wish_z = 0;
        
        if(k[SDL_SCANCODE_W]) { wish_x += fwd_x; wish_z += fwd_z; }
        if(k[SDL_SCANCODE_S]) { wish_x -= fwd_x; wish_z -= fwd_z; }
        if(k[SDL_SCANCODE_D]) { wish_x += right_x; wish_z += right_z; }
        if(k[SDL_SCANCODE_A]) { wish_x -= right_x; wish_z -= right_z; }

        float wish_len = sqrtf(wish_x*wish_x + wish_z*wish_z);
        if (wish_len > 0) { wish_x /= wish_len; wish_z /= wish_len; }

        float target_speed = p.crouching ? MAX_SPEED * 0.5f : MAX_SPEED;
        
        if (p.on_ground) {
            accelerate(&p, wish_x, wish_z, target_speed, ACCEL);
            if (space_held) { p.vy = JUMP_FORCE; p.on_ground = 0; }
        } else {
            accelerate(&p, wish_x, wish_z, target_speed * MAX_AIR_SPEED, ACCEL);
        }

        p.vy -= GRAVITY;
        p.x += p.vx; p.z += p.vz; p.y += p.vy;

        // Respawn Safety
        if (p.pos.y < -50.0f) { p.pos.x = 0; p.pos.y = 10.0f; p.pos.z = 0; p.vx=0; p.vy=0; p.vz=0; }

        resolve_collision(&p);
        
        if (p.recoil_anim > 0) p.recoil_anim -= 0.1f;

        // --- RENDER STEP ---
        glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
        glLoadIdentity();

        float cam_h = p.crouching ? 2.5f : 6.0f;
        glRotatef(-p.pitch, 1, 0, 0);
        glRotatef(-p.yaw, 0, 1, 0); // Visuals match Physics now
        glTranslatef(-p.x, -(p.y + cam_h), -p.z);

        draw_grid();
        draw_map();
        draw_weapon();

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
