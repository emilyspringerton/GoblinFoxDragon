#include "town_render.h"
#include "town_map.h"
#include <SDL2/SDL_opengl.h>
#include <math.h>
#include <stdio.h>

static void draw_box(float x, float y, float z, float w, float h, float d, float r, float g, float b) {
    glPushMatrix();
    glTranslatef(x, y, z);
    glScalef(w, h, d);
    glColor3f(r, g, b);
    glBegin(GL_QUADS);
    glVertex3f(-0.5f,-0.5f,0.5f); glVertex3f(0.5f,-0.5f,0.5f); glVertex3f(0.5f,0.5f,0.5f); glVertex3f(-0.5f,0.5f,0.5f);
    glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(-0.5f,0.5f,-0.5f); glVertex3f(0.5f,0.5f,-0.5f); glVertex3f(0.5f,-0.5f,-0.5f);
    glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(-0.5f,-0.5f,0.5f); glVertex3f(-0.5f,0.5f,0.5f); glVertex3f(-0.5f,0.5f,-0.5f);
    glVertex3f(0.5f,-0.5f,-0.5f); glVertex3f(0.5f,0.5f,-0.5f); glVertex3f(0.5f,0.5f,0.5f); glVertex3f(0.5f,-0.5f,0.5f);
    glVertex3f(-0.5f,0.5f,0.5f); glVertex3f(0.5f,0.5f,0.5f); glVertex3f(0.5f,0.5f,-0.5f); glVertex3f(-0.5f,0.5f,-0.5f);
    glVertex3f(-0.5f,-0.5f,0.5f); glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(0.5f,-0.5f,-0.5f); glVertex3f(0.5f,-0.5f,0.5f);
    glEnd();
    glPopMatrix();
}

static void draw_ring(float x, float z, float y, float radius, float r, float g, float b) {
    glColor3f(r, g, b);
    glBegin(GL_LINE_LOOP);
    for (int i = 0; i < 32; i++) {
        float a = ((float)i / 32.0f) * 6.2831853f;
        glVertex3f(x + cosf(a) * radius, y, z + sinf(a) * radius);
    }
    glEnd();
}

void town_render_world(const CrisisMockState *state) {
    draw_box(50.0f, -1.5f, 50.0f, 120.0f, 3.0f, 120.0f, 0.04f, 0.04f, 0.05f);
    draw_box(50.0f, 2.0f, 50.0f, 100.0f, 4.0f, 2.0f, 0.12f, 0.12f, 0.14f);
    draw_box(50.0f, 2.0f, 50.0f, 2.0f, 4.0f, 100.0f, 0.12f, 0.12f, 0.14f);
    draw_box(50.0f, 2.0f, 100.0f, 100.0f, 4.0f, 2.0f, 0.20f, 0.20f, 0.22f);
    draw_box(50.0f, 2.0f, 0.0f, 100.0f, 4.0f, 2.0f, 0.20f, 0.20f, 0.22f);

    size_t bcount = 0;
    const TownBuilding *b = town_map_buildings(&bcount);
    for (size_t i = 0; i < bcount; i++) {
        float base = (b[i].id == BLD_AUCTION_HOUSE) ? 0.09f : 0.06f;
        draw_box(b[i].x, b[i].h * 0.5f, b[i].z, b[i].w, b[i].h, b[i].d, base, base, base + 0.02f);
    }

    size_t scount = 0;
    const CrisisSocket *s = town_map_sockets(&scount);
    for (size_t i = 0; i < scount; i++) {
        float r = 1.0f, g = 0.4f, bcol = 0.1f;
        if (s[i].id == SOCK_ANCHOR_AUCTION) { r = 0.1f; g = 1.0f; bcol = 1.0f; }
        else if (s[i].id == SOCK_RITUAL_TOWN_HALL) { r = 0.7f; g = 0.2f; bcol = 1.0f; }
        else if (s[i].id == SOCK_HEAD_A_DOCKS || s[i].id == SOCK_HEAD_B_MINES) { r = 1.0f; g = 0.1f; bcol = 0.1f; }
        draw_box(s[i].x, 4.0f, s[i].z, 1.2f, 8.0f, 1.2f, r, g, bcol);
        draw_ring(s[i].x, s[i].z, 0.2f, s[i].radius, r, g, bcol);
        if (state && state->pressure_idx > 0 && (int)(i % 3) == (state->pressure_idx - 1)) {
            draw_ring(s[i].x, s[i].z, 0.4f, s[i].radius + 1.3f, 1.0f, 1.0f, 1.0f);
        }
    }
}

void town_render_route_distances(float px, float pz, char *out, int out_sz) {
    size_t n = 0;
    const TownRoutePoint *pts = town_map_route_points(&n);
    int used = snprintf(out, out_sz, "Routes:");
    for (size_t i = 0; i < n && used < out_sz; i++) {
        float dx = pts[i].x - px;
        float dz = pts[i].z - pz;
        float d = sqrtf(dx * dx + dz * dz);
        used += snprintf(out + used, out_sz - used, " %s %.1fm", pts[i].label, d);
    }
}
