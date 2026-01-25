#ifndef PARTICLES_H
#define PARTICLES_H

#include "shank.h"

typedef struct {
    Vec3 pos;
    Vec3 vel;
    float life; // 1.0 to 0.0
    float r, g, b;
} Particle;

#define MAX_PARTICLES 128
Particle particles[MAX_PARTICLES];

void particles_init() {
    for(int i=0; i<MAX_PARTICLES; i++) particles[i].life = 0;
}

void spawn_particle(Vec3 pos, float r, float g, float b) {
    for(int i=0; i<MAX_PARTICLES; i++) {
        if(particles[i].life <= 0) {
            particles[i].pos = pos;
            particles[i].vel.x = ((rand()%100)/50.0f - 1.0f) * 2.0f;
            particles[i].vel.y = ((rand()%100)/50.0f) * 2.0f;
            particles[i].vel.z = ((rand()%100)/50.0f - 1.0f) * 2.0f;
            particles[i].life = 1.0f;
            particles[i].r = r; particles[i].g = g; particles[i].b = b;
            return;
        }
    }
}

void update_render_particles(float dt) {
    glBegin(GL_POINTS);
    for(int i=0; i<MAX_PARTICLES; i++) {
        if(particles[i].life > 0) {
            particles[i].pos.x += particles[i].vel.x * dt;
            particles[i].pos.y += particles[i].vel.y * dt;
            particles[i].pos.z += particles[i].vel.z * dt;
            particles[i].vel.y -= 9.8f * dt; // Gravity
            particles[i].life -= dt * 2.0f;
            
            glColor3f(particles[i].r, particles[i].g, particles[i].b);
            glVertex3f(particles[i].pos.x, particles[i].pos.y, particles[i].pos.z);
        }
    }
    glEnd();
}

#endif
