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
int my_client_id = -1;
float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;
#define Z_FAR 8000.0f

// --- RENDERER FROM SPIRIT STICK (Restored Visuals) ---
void draw_char(char c, float x, float y, float s) {
    glLineWidth(2.0f); glBegin(GL_LINES);
    if(c>='0'&&c<='9'){ glVertex2f(x,y+s);glVertex2f(x+s,y+s);glVertex2f(x+s,y);glVertex2f(x,y);glVertex2f(x,y+s); }
    else if(c=='A'){glVertex2f(x,y);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y);glVertex2f(x,y+s/2);glVertex2f(x+s/2,y+s);glVertex2f(x+s/2,y+s);glVertex2f(x+s,y+s/2);}
    else if(c=='B'){glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s,y+s*0.75);glVertex2f(x+s,y+s*0.75);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x+s*0.8,y+s/2);glVertex2f(x+s,y+s/4);glVertex2f(x+s,y+s/4);glVertex2f(x+s*0.8,y);glVertex2f(x+s*0.8,y);glVertex2f(x,y);}
    else if(c=='D'){glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s*0.8,y+s);glVertex2f(x+s*0.8,y+s);glVertex2f(x,y);}
    else if(c=='J'){glVertex2f(x+s,y+s);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x,y);glVertex2f(x,y);glVertex2f(x,y+s/2);}
    else if(c=='O'){glVertex2f(x,y);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x+s,y+s);glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x,y);}
    else if(c=='I'){glVertex2f(x+s/2,y);glVertex2f(x+s/2,y+s);glVertex2f(x,y);glVertex2f(x+s,y);glVertex2f(x,y+s);glVertex2f(x+s,y+s);}
    else if(c=='N'){glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x+s,y+s);}
    else if(c=='S'){glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x,y);}
    else if(c=='E'){glVertex2f(x+s,y);glVertex2f(x,y);glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s,y+s);glVertex2f(x,y+s/2);glVertex2f(x+s*0.8,y+s/2);}
    else if(c=='R'){glVertex2f(x,y);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x+s,y+s);glVertex2f(x+s,y+s);glVertex2f(x+s,y+s/2);glVertex2f(x+s,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x,y+s/2);glVertex2f(x+s,y);}
    else if(c=='V'){glVertex2f(x,y+s);glVertex2f(x+s/2,y);glVertex2f(x+s/2,y);glVertex2f(x+s,y+s);}
    else if(c==' '){} 
    else { glVertex2f(x,y);glVertex2f(x+s,y);glVertex2f(x+s,y);glVertex2f(x+s,y+s);glVertex2f(x+s,y+s);glVertex2f(x,y+s);glVertex2f(x,y+s);glVertex2f(x,y); }
    glEnd();
}
void draw_string(const char* str, float x, float y, float size) { while(*str) { draw_char(*str, x, y, size); x += size * 1.5f; str++; } }

#define MAX_TRAILS 4096 
#define GRID_SIZE 50.0f
typedef struct { int cx, cz; float life; } Trail;
Trail trails[MAX_TRAILS];
int trail_head = 0;

void add_trail(int x, int z) {
    int prev = (trail_head - 1 + MAX_TRAILS) % MAX_TRAILS;
    if (trails[prev].cx == x && trails[prev].cz == z && trails[prev].life > 0.9f) return;
    trails[trail_head].cx = x; trails[trail_head].cz = z;
    trails[trail_head].life = 1.0f;
    trail_head = (trail_head + 1) % MAX_TRAILS;
}

void update_and_draw_trails() {
    glEnable(GL_BLEND); glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (p->active && p->on_ground) {
            int gx = (int)floorf(p->x / GRID_SIZE) * (int)GRID_SIZE + (int)(GRID_SIZE/2);
            int gz = (int)floorf(p->z / GRID_SIZE) * (int)GRID_SIZE + (int)(GRID_SIZE/2);
            add_trail(gx, gz);
        }
    }
    glLineWidth(2.0f);
    for(int i=0; i<MAX_TRAILS; i++) {
        if (trails[i].life > 0) {
            float s = (GRID_SIZE / 2.0f) - 4.0f;
            glColor4f(1.0f, 0.0f, 0.8f, trails[i].life);
            glBegin(GL_LINE_LOOP);
            glVertex3f(trails[i].cx - s, 0.2f, trails[i].cz - s);
            glVertex3f(trails[i].cx + s, 0.2f, trails[i].cz - s);
            glVertex3f(trails[i].cx + s, 0.2f, trails[i].cz + s);
            glVertex3f(trails[i].cx - s, 0.2f, trails[i].cz + s);
            glEnd();
            trails[i].life -= 0.02f;
        }
    }
    glDisable(GL_BLEND);
}

void draw_grid() {
    glLineWidth(1.0f); glBegin(GL_LINES); 
    glColor3f(0.0f, 1.0f, 1.0f);
    for(int i=-4000; i<=4000; i+=50) { 
        glVertex3f(i, 0.1f, -4000); glVertex3f(i, 0.1f, 4000);
        glVertex3f(-4000, 0.1f, i); glVertex3f(4000, 0.1f, i); 
    }
    glEnd();
}

void draw_map() {
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        float nr = 0.5f + 0.5f * sinf(b.x * 0.005f + b.y * 0.01f);
        float ng = 0.5f + 0.5f * sinf(b.z * 0.005f + 2.0f);
        float nb = 0.5f + 0.5f * sinf(b.x * 0.005f + 4.0f);
        if(nr > 0.8f) nr = 1.0f;
        if(ng > 0.8f) ng = 1.0f;
        if(nb > 0.8f) nb = 1.0f;

        glPushMatrix(); 
        glTranslatef(b.x, b.y, b.z); 
        glScalef(b.w, b.h, b.d);
        glBegin(GL_QUADS); 
        glColor3f(0.02f, 0.02f, 0.02f); 
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,-0.5);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
        glEnd();
        
        glLineWidth(2.0f); 
        glColor3f(nr, ng, nb);
        glBegin(GL_LINE_LOOP); glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5); glEnd();
        glBegin(GL_LINE_LOOP); glVertex3f(-0.5, -0.5, 0.5); glVertex3f(0.5, -0.5, 0.5); glVertex3f(0.5, -0.5, -0.5); glVertex3f(-0.5, -0.5, -0.5); glEnd();
        glBegin(GL_LINES);
        glVertex3f(-0.5, -0.5, 0.5); glVertex3f(-0.5, 0.5, 0.5);
        glVertex3f(0.5, -0.5, 0.5); glVertex3f(0.5, 0.5, 0.5);
        glVertex3f(0.5, -0.5, -0.5); glVertex3f(0.5, 0.5, -0.5);
        glVertex3f(-0.5, -0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        glPopMatrix();
    }
}

void draw_buggy_model() {
    glColor3f(0.2f, 0.2f, 0.2f);
    glPushMatrix(); glScalef(2.0f, 1.0f, 3.5f); 
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd(); 
    glLineWidth(2.0f); glColor3f(1.0f, 0.0f, 0.0f);
    glBegin(GL_LINES);
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1);
    glVertex3f(1,1,-1); glVertex3f(-1,1,-1); glVertex3f(-1,1,-1); glVertex3f(-1,1,1);
    glEnd();
    glPopMatrix();
    glColor3f(0.1f, 0.1f, 0.1f);
    float wx[] = {-2.2, 2.2, -2.2, 2.2}; float wz[] = {2.5, 2.5, -2.5, -2.5};
    for(int i=0; i<4; i++) {
        glPushMatrix(); glTranslatef(wx[i], -0.5f, wz[i]); glScalef(0.8f, 1.5f, 1.5f);
        glBegin(GL_QUADS); 
        glColor3f(0.1f, 0.1f, 0.1f);
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,-0.5,-0.5);
        glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,-0.5,0.5);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5); glVertex3f(-0.5,-0.5,-0.5);
        glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glEnd(); 
        glLineWidth(2.0f); glColor3f(0.0f, 1.0f, 1.0f); 
        glBegin(GL_LINE_LOOP); glVertex3f(-0.51, 0.5, 0.5); glVertex3f(-0.51, -0.5, 0.5); glVertex3f(-0.51, -0.5, -0.5); glVertex3f(-0.51, 0.5, -0.5); glEnd();
        glPopMatrix();
    }
}

void draw_gun_model(int weapon_id) {
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); glScalef(0.05f, 0.05f, 0.8f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); glScalef(0.15f, 0.2f, 0.5f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); glScalef(0.1f, 0.15f, 1.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); glScalef(0.25f, 0.15f, 0.8f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); glScalef(0.08f, 0.12f, 2.0f); break;
    }
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
    glLineWidth(1.0f); glColor3f(1.0f, 1.0f, 1.0f);
    glBegin(GL_LINES); glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,1); glVertex3f(-1,-1,1); glEnd();
}

void draw_weapon_p(PlayerState *p) {
    if (p->in_vehicle) return; 
    glPushMatrix();
    glLoadIdentity();
    float kick = p->recoil_anim * 0.2f;
    float reload_dip = (p->reload_timer > 0) ? sinf(p->reload_timer * 0.2f) * 0.5f - 0.5f : 0.0f;
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    float bob = sinf(SDL_GetTicks() * 0.015f) * speed * 0.15f; 
    float x_offset = (current_fov < 50.0f) ? 0.25f : 0.4f;
    glTranslatef(x_offset, -0.5f + kick + reload_dip + (bob * 0.5f), -1.2f + (kick * 0.5f) + bob);
    glRotatef(-p->recoil_anim * 10.0f, 1, 0, 0);
    draw_gun_model(p->current_weapon);
    glPopMatrix();
}

void draw_head(int weapon_id) {
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); break;
    }
    glBegin(GL_QUADS);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(-0.4, 0, 0.4);
    glVertex3f(-0.4, 0.8, -0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0.8, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(-0.4, 0, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(-0.4, 0, -0.4);
    glVertex3f(-0.4, 0.8, 0.4); glVertex3f(-0.4, 0, 0.4); glVertex3f(-0.4, 0, -0.4); glVertex3f(-0.4, 0.8, -0.4);
    glVertex3f(0.4, 0.8, 0.4); glVertex3f(0.4, 0, 0.4); glVertex3f(0.4, 0, -0.4); glVertex3f(0.4, 0.8, -0.4);
    glEnd();
}

void draw_player_3rd(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y + 2.0f, p->z);
    glRotatef(-p->yaw, 0, 1, 0);
    if (p->in_vehicle) {
        draw_buggy_model();
    } else {
        if(p->health <= 0) glColor3f(0.2, 0, 0); else glColor3f(1, 0, 0);
        glPushMatrix(); glScalef(0.97f, 2.91f, 0.97f); 
        glBegin(GL_QUADS);
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
        glEnd(); glPopMatrix();
        glPushMatrix(); glTranslatef(0, 1.54f, 0); draw_head(p->current_weapon); glPopMatrix();
        glPushMatrix(); glTranslatef(0.5f, 1.0f, 0.5f);
        glRotatef(p->pitch, 1, 0, 0);   
        glScalef(0.8f, 0.8f, 0.8f); draw_gun_model(p->current_weapon); glPopMatrix(); 
    }
    glPopMatrix();
}

void draw_circle(float x, float y, float r, int segments) {
    glBegin(GL_LINE_LOOP);
    for(int i=0; i<segments; i++) {
        float theta = 2.0f * 3.1415926f * (float)i / (float)segments;
        float cx = r * cosf(theta);
        float cy = r * sinf(theta);
        glVertex2f(x + cx, y + cy);
    }
    glEnd();
}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    glColor3f(0, 1, 0);
    if (current_fov < 50.0f) { glBegin(GL_LINES); glVertex2f(0, 360); glVertex2f(1280, 360); glVertex2f(640, 0); glVertex2f(640, 720); glEnd(); } 
    else { glLineWidth(2.0f); glBegin(GL_LINES); glVertex2f(630, 360); glVertex2f(650, 360); glVertex2f(640, 350); glVertex2f(640, 370); glEnd(); }
    
    if (p->hit_feedback > 0) {
        if (p->hit_feedback >= 25) glColor3f(1.0f, 0.0f, 0.0f); // RED
        else glColor3f(0.0f, 1.0f, 0.0f); // GREEN
        glLineWidth(2.0f);
        draw_circle(640, 360, 20.0f, 16);
        if (p->hit_feedback >= 25) draw_circle(640, 360, 28.0f, 16);
    }

    glColor3f(0.2f, 0, 0); glRectf(50, 50, 250, 70); glColor3f(1.0f, 0, 0);
    glRectf(50, 50, 50 + (p->health * 2), 70);
    glColor3f(0, 0, 0.2f); glRectf(50, 80, 250, 100); glColor3f(0.2f, 0.2f, 1.0f);
    glRectf(50, 80, 50 + (p->shield * 2), 100);
    if (p->in_vehicle) {
        glColor3f(0.0f, 1.0f, 0.0f);
        draw_string("BUGGY ONLINE", 50, 120, 12);
    }
    
    float raw_speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    char vel_buf[32]; sprintf(vel_buf, "VEL: %.2f", raw_speed);
    glColor3f(1.0f, 1.0f, 0.0f); draw_string(vel_buf, 1100, 50, 8); 

    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.02f, 0.02f, 0.05f, 1.0f); // DEEP SPACE BLUE BG
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT); glLoadIdentity();
    float cam_y = render_p->in_vehicle ? 8.0f : (render_p->crouching ? 2.5f : EYE_HEIGHT);
    float cam_z_off = render_p->in_vehicle ? 10.0f : 0.0f;
    
    float rad = -cam_yaw * 0.01745f;
    float cx = sinf(rad) * cam_z_off;
    float cz = cosf(rad) * cam_z_off;
    
    glRotatef(-cam_pitch, 1, 0, 0); glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-(render_p->x - cx), -(render_p->y + cam_y), -(render_p->z - cz));
    
    draw_grid(); 
    update_and_draw_trails();
    draw_map();
    if (render_p->in_vehicle) draw_player_3rd(render_p);
    for(int i=1; i<MAX_CLIENTS; i++) if(local_state.players[i].active && i != my_client_id) draw_player_3rd(&local_state.players[i]);
    draw_weapon_p(render_p); draw_hud(render_p);
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
    gluPerspective(current_fov, 1280.0/720.0, 0.1, Z_FAR);
    glMatrixMode(GL_MODELVIEW);
    glLoadIdentity();
    glEnable(GL_DEPTH_TEST);
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT [GOLDEN HYBRID]", 100, 100, 1280, 720, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    SDL_GL_CreateContext(win);
    SDL_GL_SetSwapInterval(1);

    local_init_match(1, 0);

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            
            // --- GOLDEN MENU LOGIC ---
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) {
                    if(app_state == STATE_PLAYING) {
                        app_state = STATE_B_MENU;
                        // Mouse remains captured
                    } else if (app_state == STATE_B_MENU) {
                        app_state = STATE_MAIN_MENU;
                        SDL_SetRelativeMouseMode(SDL_FALSE);
                    }
                }
                
                if(app_state == STATE_MAIN_MENU && e.key.keysym.sym == SDLK_b) {
                    app_state = STATE_B_MENU;
                }
                
                if(app_state == STATE_B_MENU && e.key.keysym.sym == SDLK_p) {
                    app_state = STATE_PLAYING;
                    local_init_match(8, 0);
                    SDL_SetRelativeMouseMode(SDL_TRUE);
                }
            }
            
            if(app_state == STATE_PLAYING && e.type == SDL_MOUSEMOTION) {
                float sens = (current_fov < 50.0f) ? 0.05f : 0.15f; 
                cam_yaw -= e.motion.xrel * sens;
                if(cam_yaw > 360) cam_yaw -= 360; if(cam_yaw < 0) cam_yaw += 360;
                cam_pitch -= e.motion.yrel * sens;
                if(cam_pitch > 89) cam_pitch = 89; if(cam_pitch < -89) cam_pitch = -89;
            }
        }

        // --- RENDER & UPDATE ---
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
            setup_3d();
            
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd-=1; if(k[SDL_SCANCODE_S]) fwd+=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            int jump = k[SDL_SCANCODE_SPACE]; int crouch = k[SDL_SCANCODE_LCTRL];
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int reload = k[SDL_SCANCODE_R];
            int use = k[SDL_SCANCODE_E];
            int wpn_req = local_state.players[0].current_weapon;
            if(k[SDL_SCANCODE_1]) wpn_req=0; if(k[SDL_SCANCODE_2]) wpn_req=1;
            if(k[SDL_SCANCODE_3]) wpn_req=2; if(k[SDL_SCANCODE_4]) wpn_req=3; if(k[SDL_SCANCODE_5]) wpn_req=4;

            // Update Physics
            local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn_req, jump, crouch, reload, NULL, 0);
            
            // Draw Scene (Using Spirit Stick Rendering)
            draw_scene(&local_state.players[0]);
        }

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    return 0;
}
