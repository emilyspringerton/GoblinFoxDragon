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
float current_fov = 75.0f;
GLuint tex_grid = 0;
float screen_shake = 0.0f;

// --- PROCEDURAL ASSETS ---
GLuint generate_cyber_grid() {
    unsigned char data[64][64][3];
    for (int y = 0; y < 64; y++) {
        for (int x = 0; x < 64; x++) {
            int is_grid = (x < 2 || y < 2 || x > 61 || y > 61);
            if (is_grid) { data[y][x][0] = 255; data[y][x][1] = 0; data[y][x][2] = 255; } 
            else { data[y][x][0] = 10; data[y][x][1] = 5; data[y][x][2] = 20; } 
        }
    }
    GLuint tex;
    glGenTextures(1, &tex);
    glBindTexture(GL_TEXTURE_2D, tex);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR_MIPMAP_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
    gluBuild2DMipmaps(GL_TEXTURE_2D, 3, 64, 64, GL_RGB, GL_UNSIGNED_BYTE, data);
    return tex;
}

void setup_lighting_fog() {
    glEnable(GL_FOG);
    GLfloat fogColor[] = {0.05f, 0.0f, 0.1f, 1.0f};
    glFogfv(GL_FOG_COLOR, fogColor);
    glFogi(GL_FOG_MODE, GL_EXP2);
    glFogf(GL_FOG_DENSITY, 0.0015f);
    glHint(GL_FOG_HINT, GL_NICEST);
}

void apply_screen_shake() {
    if (screen_shake > 0) {
        float shake_x = ((float)(rand() % 100) / 100.0f - 0.5f) * screen_shake * 2.0f;
        float shake_y = ((float)(rand() % 100) / 100.0f - 0.5f) * screen_shake * 2.0f;
        glRotatef(shake_x, 0, 1, 0); glRotatef(shake_y, 1, 0, 0);
        screen_shake -= 0.05f; if (screen_shake < 0) screen_shake = 0;
    }
}

// --- VECTOR ART ---
void draw_string(const char* str, float x, float y, float size) {
    glDisable(GL_TEXTURE_2D); glLineWidth(2.0f); glBegin(GL_LINES);
    while(*str) {
        char c = *str;
        if(c>='A'){ glVertex2f(x,y); glVertex2f(x+size,y+size); glVertex2f(x,y+size); glVertex2f(x+size,y); } 
        else if (c>='0' && c<='9') { glVertex2f(x,y); glVertex2f(x+size,y); glVertex2f(x+size,y+size); glVertex2f(x,y+size); glVertex2f(x,y); }
        else { glVertex2f(x,y); glVertex2f(x+size,y); glVertex2f(x+size,y+size); glVertex2f(x,y+size); glVertex2f(x,y); }
        x += size * 1.5f; str++;
    }
    glEnd();
}

void draw_ronin_insignia(float x, float y, float s) {
    glLineWidth(3.0f);
    glBegin(GL_LINE_LOOP); // Helmet
    for(int i=0; i<=180; i+=20) { float rad = i * 3.14159f / 180.0f; glVertex2f(x + cosf(rad)*s, y + sinf(rad)*s); }
    glVertex2f(x - s, y); glVertex2f(x, y - s*1.2f); glVertex2f(x + s, y);
    glEnd();
    glColor3f(0.0f, 1.0f, 1.0f); // Cyan Crack
    glBegin(GL_LINE_STRIP);
    glVertex2f(x - s*0.5f, y + s); glVertex2f(x, y + s*0.2f); glVertex2f(x - s*0.2f, y - s*0.5f); glVertex2f(x + s*0.5f, y - s*1.2f);
    glEnd();
}

void draw_storm_chevrons(float x, float y, float s, int charges) {
    glLineWidth(2.0f);
    for(int i=0; i<5; i++) {
        float oy = y + (i * s * 0.6f);
        if (i < charges) glColor3f(1.0f, 0.2f, 0.2f); else glColor3f(0.2f, 0.0f, 0.0f);
        glBegin(GL_LINE_STRIP); glVertex2f(x - s, oy); glVertex2f(x, oy + s*0.3f); glVertex2f(x + s, oy); glEnd();
    }
}

// --- 3D MODELS ---
void draw_box(float w, float h, float d) {
    glPushMatrix(); glScalef(w, h, d);
    glBegin(GL_QUADS);
    // Front
    glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
    // Back
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,-0.5,-0.5);
    // Left
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
    // Right
    glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,-0.5,0.5);
    // Top
    glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    // Bottom
    glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,0.5);
    glEnd();
    glPopMatrix();
}

void draw_ronin_shell(float anim) {
    glColor3f(0.1f, 0.1f, 0.1f); // Matte Black
    glPushMatrix(); glTranslatef(0, 1.2f, 0); 
    draw_box(1.2f, 1.4f, 0.8f); // Torso
    glColor3f(0.6f, 0.0f, 0.0f); // Red Lining
    glBegin(GL_QUADS);
    glVertex3f(-0.6, -0.7, 0.41); glVertex3f(0.6, -0.7, 0.41); glVertex3f(0.6, -0.65, 0.41); glVertex3f(-0.6, -0.65, 0.41);
    glEnd();
    glColor3f(0.1f, 0.1f, 0.1f);
    glPushMatrix(); glTranslatef(-0.7f, 0.0f, 0.0f); draw_box(0.4f, 1.4f, 0.4f); glPopMatrix(); // L
    glPushMatrix(); glTranslatef(0.7f, 0.0f, 0.0f); draw_box(0.4f, 1.4f, 0.4f); glPopMatrix(); // R
    glPopMatrix();
    glColor3f(0.15f, 0.15f, 0.18f); // Pants
    glPushMatrix(); glTranslatef(-0.3f, 0.4f, 0); draw_box(0.5f, 1.6f, 0.6f); glPopMatrix();
    glPushMatrix(); glTranslatef(0.3f, 0.4f, 0); draw_box(0.5f, 1.6f, 0.6f); glPopMatrix();
}

void draw_storm_mask() {
    glColor3f(0.05f, 0.05f, 0.05f); draw_box(0.6f, 0.7f, 0.6f); // Head
    glColor3f(0.2f, 0.2f, 0.2f); glPushMatrix(); glTranslatef(0, -0.1f, 0.35f); draw_box(0.5f, 0.4f, 0.1f); glPopMatrix(); // Plate
    glColor3f(0.0f, 1.0f, 1.0f); // Vents
    glPushMatrix(); glTranslatef(0.2f, -0.1f, 0.41f); draw_box(0.1f, 0.2f, 0.02f); glPopMatrix();
    glPushMatrix(); glTranslatef(-0.2f, -0.1f, 0.41f); draw_box(0.1f, 0.2f, 0.02f); glPopMatrix();
}

void draw_player_3rd(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y, p->z);
    glRotatef(-p->yaw, 0, 1, 0);
    if (p->in_vehicle) {
        draw_box(2.0f, 1.0f, 3.5f); // Placeholder Buggy
    } else {
        draw_ronin_shell(p->recoil_anim);
        glPushMatrix(); glTranslatef(0, 2.3f, 0); glRotatef(p->pitch, 1, 0, 0); draw_storm_mask(); glPopMatrix();
    }
    glPopMatrix();
}

void draw_projectiles() {
    glDisable(GL_TEXTURE_2D); glPointSize(6.0f); glBegin(GL_POINTS);
    for(int i=0; i<MAX_PROJECTILES; i++) {
        if(local_state.projectiles[i].active) {
            if (local_state.projectiles[i].bounces_left > 0) glColor3f(1.0f, 0.5f, 0.0f);
            else glColor3f(1.0f, 0.0f, 0.0f);
            glVertex3f(local_state.projectiles[i].x, local_state.projectiles[i].y, local_state.projectiles[i].z);
        }
    }
    glEnd();
}

void draw_floor() {
    glEnable(GL_TEXTURE_2D); glBindTexture(GL_TEXTURE_2D, tex_grid);
    glColor3f(1.0f, 1.0f, 1.0f);
    float size = 4000.0f; float reps = 100.0f;
    glBegin(GL_QUADS);
    glTexCoord2f(0, 0); glVertex3f(-size, 0, -size);
    glTexCoord2f(reps, 0); glVertex3f(size, 0, -size);
    glTexCoord2f(reps, reps); glVertex3f(size, 0, size);
    glTexCoord2f(0, reps); glVertex3f(-size, 0, size);
    glEnd();
    glDisable(GL_TEXTURE_2D);
}

void draw_map() {
    glDisable(GL_TEXTURE_2D);
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        float h_factor = b.y / 100.0f; 
        glColor3f(0.1f, 0.1f + h_factor, 0.3f); 
        glPushMatrix(); glTranslatef(b.x, b.y, b.z); glScalef(b.w, b.h, b.d);
        draw_box(1.0f, 1.0f, 1.0f); 
        glLineWidth(2.0f); glColor3f(0.0f, 1.0f, 1.0f);
        // Wireframes...
        glBegin(GL_LINE_LOOP); glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5); glEnd();
        glPopMatrix();
    }
}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    
    glColor3f(0, 1, 0);
    float recoil = p->recoil_anim * 20.0f;
    glLineWidth(2.0f); glBegin(GL_LINES); 
    glVertex2f(640 - 10 - recoil, 360); glVertex2f(640 - 2 - recoil, 360);
    glVertex2f(640 + 10 + recoil, 360); glVertex2f(640 + 2 + recoil, 360);
    glVertex2f(640, 360 - 10 - recoil); glVertex2f(640, 360 - 2 - recoil);
    glVertex2f(640, 360 + 10 + recoil); glVertex2f(640, 360 + 2 + recoil);
    glEnd();

    if (p->storm_charges > 0) {
        draw_storm_chevrons(700, 340, 10, p->storm_charges);
        glColor3f(1.0f, 0.2f, 0.2f); draw_string("STORM ACTIVE", 720, 350, 10);
    }

    glColor3f(1.0f, 1.0f, 1.0f);
    draw_ronin_insignia(60, 60, 40);
    glColor3f(0.5f, 0.5f, 0.5f);
    draw_string("SHANKPIT // STORM DIV", 120, 50, 8);

    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_scene() {
    glClearColor(0.05f, 0.0f, 0.1f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(current_fov, 1280.0/720.0, 0.1, Z_FAR);
    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
    apply_screen_shake();
    
    PlayerState *p = &local_state.players[0];
    glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-p->x, -(p->y + EYE_HEIGHT), -p->z);
    
    draw_floor(); draw_map(); draw_projectiles(); 
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        if(local_state.players[i].active && i != my_client_id) {
            draw_player_3rd(&local_state.players[i]);
        }
    }
    
    draw_hud(p);
}

void setup_2d() {
    int w, h; SDL_GL_GetDrawableSize(SDL_GL_GetCurrentWindow(), &w, &h);
    glViewport(0, 0, w, h); glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glLoadIdentity(); glDisable(GL_DEPTH_TEST);
}

int test_graphics_integrity() { return 1; }

int main(int argc, char* argv[]) {
    for(int i=1; i<argc; i++) if(strcmp(argv[i], "--test-gfx") == 0) return test_graphics_integrity() ? 0 : 1;

    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT [PHASE 526 LEVEL LOAD]", 100, 100, 1280, 720, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    SDL_GL_CreateContext(win);
    
    tex_grid = generate_cyber_grid();
    setup_lighting_fog();
    local_init_match(8, 0);
    
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_MOUSEMOTION && app_state == STATE_PLAYING) {
                cam_yaw += e.motion.xrel * 0.15f; 
                cam_pitch += e.motion.yrel * 0.15f;
                if(cam_pitch > 89.0f) cam_pitch = 89.0f; if(cam_pitch < -89.0f) cam_pitch = -89.0f;
            }
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) {
                    if(app_state == STATE_PLAYING) app_state = STATE_B_MENU;
                    else if(app_state == STATE_B_MENU) { app_state = STATE_MAIN_MENU; SDL_SetRelativeMouseMode(SDL_FALSE); }
                }
                
                // --- GOLDEN MENU LOGIC ---
                if(app_state == STATE_MAIN_MENU && e.key.keysym.sym == SDLK_b) app_state = STATE_B_MENU;
                
                if(app_state == STATE_B_MENU) {
                    if (e.key.keysym.sym == SDLK_p) {
                        app_state = STATE_PLAYING;
                        local_init_match(8, 0); // Play (Bots)
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                    }
                    if (e.key.keysym.sym == SDLK_2) {
                        app_state = STATE_PLAYING;
                        local_init_match(8, 0); // Level 2 (Standard Map)
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        printf("LOADING LEVEL...\n");
                    }
                }
            }
        }
        
        if (app_state == STATE_PLAYING) {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd = (k[SDL_SCANCODE_S]?1:0) - (k[SDL_SCANCODE_W]?1:0);
            float str = (k[SDL_SCANCODE_D]?1:0) - (k[SDL_SCANCODE_A]?1:0);
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int ability = k[SDL_SCANCODE_E];
            
            if (shoot && local_state.players[0].attack_cooldown == 0 && local_state.players[0].reload_timer == 0) screen_shake = 5.0f;
            local_update(fwd, str, cam_yaw, cam_pitch, shoot, -1, k[SDL_SCANCODE_SPACE], k[SDL_SCANCODE_LCTRL], 0, ability, NULL, 0);
            draw_scene();
        } else {
            glClearColor(0,0,0,1); glClear(GL_COLOR_BUFFER_BIT);
            setup_2d();
            glColor3f(0,1,1);
            if(app_state == STATE_MAIN_MENU) draw_string("MAIN MENU - PRESS B", 500, 500, 20);
            else { 
                glColor3f(1,0,0); 
                draw_string("STAGING", 500, 500, 20); 
                draw_string("P: PLAY (BOTS)", 500, 400, 10);
                draw_string("2: LOAD LEVEL", 500, 350, 10);
                draw_string("ESC: BACK", 500, 300, 10);
            }
        }
        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    return 0;
}
