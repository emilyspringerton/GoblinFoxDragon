#include "town_render.h"
#include "town_map.h"
#include <SDL2/SDL_opengl.h>
#include <math.h>
#include <stdio.h>
#include <string.h>

static const float town_label_landmark_clearance = 2.4f;
static const float town_label_route_clearance = 1.6f;
static const float town_label_gate_clearance = 1.2f;
static const float town_label_height_max = 34.0f;
static const float town_label_horizontal_nudge = 1.1f;

static float clampf(float v, float lo, float hi) {
    if (v < lo) return lo;
    if (v > hi) return hi;
    return v;
}

static float hash01(unsigned int x) {
    x ^= x >> 16;
    x *= 0x7feb352dU;
    x ^= x >> 15;
    x *= 0x846ca68bU;
    x ^= x >> 16;
    return (float)(x & 1023U) / 1023.0f;
}

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

static void draw_box_outline(float x, float y, float z, float w, float h, float d, float r, float g, float b, float line_w) {
    glPushMatrix();
    glTranslatef(x, y, z);
    glScalef(w, h, d);
    glColor3f(r, g, b);
    glLineWidth(line_w);

    glBegin(GL_LINE_LOOP);
    glVertex3f(-0.5f, -0.5f,  0.5f); glVertex3f( 0.5f, -0.5f,  0.5f); glVertex3f( 0.5f,  0.5f,  0.5f); glVertex3f(-0.5f,  0.5f,  0.5f);
    glEnd();
    glBegin(GL_LINE_LOOP);
    glVertex3f(-0.5f, -0.5f, -0.5f); glVertex3f( 0.5f, -0.5f, -0.5f); glVertex3f( 0.5f,  0.5f, -0.5f); glVertex3f(-0.5f,  0.5f, -0.5f);
    glEnd();
    glBegin(GL_LINES);
    glVertex3f(-0.5f,-0.5f,-0.5f); glVertex3f(-0.5f,-0.5f, 0.5f);
    glVertex3f( 0.5f,-0.5f,-0.5f); glVertex3f( 0.5f,-0.5f, 0.5f);
    glVertex3f(-0.5f, 0.5f,-0.5f); glVertex3f(-0.5f, 0.5f, 0.5f);
    glVertex3f( 0.5f, 0.5f,-0.5f); glVertex3f( 0.5f, 0.5f, 0.5f);
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

static void draw_mock_door(const TownBuilding *b) {
    float dx = (b->x >= 150.0f) ? 1.0f : -1.0f;
    float dz = 0.0f;
    if (fabsf(b->x - 150.0f) < 50.0f) {
        dx = 0.0f;
        dz = (b->z >= 150.0f) ? -1.0f : 1.0f;
    }

    float half_w = b->w * 0.5f;
    float half_d = b->d * 0.5f;
    float px = b->x + (dx * half_w + dz * 0.0f);
    float pz = b->z + (dz * half_d + dx * 0.0f);
    float door_w = fmaxf(2.2f, fminf(b->w, b->d) * 0.22f);
    float door_h = fmaxf(3.5f, b->h * 0.34f);

    draw_box(px - dx * 0.2f, door_h * 0.5f, pz - dz * 0.2f, (dx != 0.0f ? 0.5f : door_w), door_h, (dz != 0.0f ? 0.5f : door_w), 0.03f, 0.03f, 0.04f);
    draw_box_outline(px, door_h * 0.5f, pz, (dx != 0.0f ? 0.9f : door_w + 0.9f), door_h + 0.6f, (dz != 0.0f ? 0.9f : door_w + 0.9f), 0.98f, 0.72f, 0.20f, 1.6f);
    draw_box(px + dx * 0.35f, 0.18f, pz + dz * 0.35f, (dx != 0.0f ? 1.0f : door_w + 1.2f), 0.25f, (dz != 0.0f ? 1.0f : door_w + 1.2f), 0.20f, 0.95f, 0.95f);
}

static void draw_mock_gate(float x, float z, float opening_w, float post_h, float r, float g, float b) {
    float post_w = 1.2f;
    draw_box(x - opening_w * 0.5f, post_h * 0.5f, z, post_w, post_h, post_w, 0.08f, 0.08f, 0.09f);
    draw_box(x + opening_w * 0.5f, post_h * 0.5f, z, post_w, post_h, post_w, 0.08f, 0.08f, 0.09f);
    draw_box(x, post_h + 0.9f, z, opening_w + 2.2f, 1.0f, 1.4f, 0.05f, 0.05f, 0.06f);

    draw_box_outline(x - opening_w * 0.5f, post_h * 0.5f, z, post_w, post_h, post_w, r, g, b, 1.8f);
    draw_box_outline(x + opening_w * 0.5f, post_h * 0.5f, z, post_w, post_h, post_w, r, g, b, 1.8f);
    draw_box_outline(x, post_h + 0.9f, z, opening_w + 2.2f, 1.0f, 1.4f, r, g, b, 2.0f);
}

static float town_label_roof_height(float x, float z) {
    size_t n = 0;
    const TownBuilding *b = town_map_buildings(&n);
    float best = 6.0f;
    float best_d2 = 1e9f;
    for (size_t i = 0; i < n; i++) {
        float dx = b[i].x - x;
        float dz = b[i].z - z;
        float d2 = dx * dx + dz * dz;
        if (d2 < best_d2) {
            best_d2 = d2;
            best = b[i].h;
        }
    }
    return best;
}

int town_render_collect_labels(const CrisisMockState *state,
                               float cam_x, float cam_y, float cam_z,
                               TownMetaLabel *out_labels, int max_labels) {
    (void)state;
    if (!out_labels || max_labels <= 0) return 0;

    int out_count = 0;
    size_t bcount = 0;
    const TownBuilding *b = town_map_buildings(&bcount);
    for (size_t i = 0; i < bcount && out_count < max_labels; i++) {
        TownMetaLabel *lbl = &out_labels[out_count++];
        snprintf(lbl->text, sizeof(lbl->text), "%s", b[i].label);
        lbl->x = b[i].x + ((b[i].x > 150.0f) ? -0.8f : 0.8f);
        lbl->z = b[i].z + ((b[i].z > 150.0f) ? -0.6f : 0.6f);
        lbl->y = clampf(b[i].h + town_label_landmark_clearance, 3.0f, town_label_height_max);
        lbl->mode = TOWN_LABEL_LANDMARK;
    }

    size_t rcount = 0;
    const TownRoutePoint *r = town_map_route_points(&rcount);
    for (size_t i = 0; i < rcount && out_count < max_labels; i++) {
        TownMetaLabel *lbl = &out_labels[out_count++];
        snprintf(lbl->text, sizeof(lbl->text), "Route %s", r[i].label);
        lbl->x = r[i].x;
        lbl->z = r[i].z;
        float near_roof = town_label_roof_height(r[i].x, r[i].z);
        lbl->y = clampf(near_roof + town_label_route_clearance, 2.0f, town_label_height_max - 2.5f);
        lbl->mode = TOWN_LABEL_ROUTE;
    }

    const float gates[][2] = {
        {270.0f, 258.0f}, // docks
        {36.0f, 60.0f},   // secret gate
        {270.0f, 54.0f},  // mines
        {24.0f, 174.0f}   // residences / west edge
    };
    const char *gate_names[] = {"Gate Docks", "Gate Secret", "Gate Mines", "Gate Residences"};
    for (int i = 0; i < 4 && out_count < max_labels; i++) {
        TownMetaLabel *lbl = &out_labels[out_count++];
        snprintf(lbl->text, sizeof(lbl->text), "%s", gate_names[i]);
        lbl->x = gates[i][0];
        lbl->z = gates[i][1];
        float near_roof = town_label_roof_height(lbl->x, lbl->z);
        lbl->y = clampf(near_roof + town_label_gate_clearance, 2.0f, town_label_height_max - 4.0f);
        lbl->mode = TOWN_LABEL_GATE;
    }

    for (int i = 0; i < out_count; i++) {
        float dx = cam_x - out_labels[i].x;
        float dz = cam_z - out_labels[i].z;
        float len = sqrtf(dx * dx + dz * dz);
        if (len > 0.001f) {
            out_labels[i].x += (dx / len) * town_label_horizontal_nudge;
            out_labels[i].z += (dz / len) * town_label_horizontal_nudge;
        }
        if (cam_y < out_labels[i].y + 1.0f) out_labels[i].y = cam_y + 1.0f;
    }

    return out_count;
}

void town_render_world(const CrisisMockState *state) {
    static int log_once = 0;
    if (!log_once) {
        log_once = 1;
        printf("[TOWN] labels tuned: landmark_clearance=%.2f route_clearance=%.2f gate_clearance=%.2f max=%.2f\n",
               town_label_landmark_clearance, town_label_route_clearance, town_label_gate_clearance, town_label_height_max);
        printf("[TOWN] outline pass=on\n");
    }

    draw_box(150.0f, -4.5f, 150.0f, 360.0f, 9.0f, 360.0f, 0.04f, 0.04f, 0.05f);
    draw_box(150.0f, 6.0f, 150.0f, 300.0f, 12.0f, 6.0f, 0.12f, 0.12f, 0.14f);
    draw_box(150.0f, 6.0f, 150.0f, 6.0f, 12.0f, 300.0f, 0.12f, 0.12f, 0.14f);
    draw_box(150.0f, 6.0f, 300.0f, 300.0f, 12.0f, 6.0f, 0.20f, 0.20f, 0.22f);
    draw_box(150.0f, 6.0f, 0.0f, 300.0f, 12.0f, 6.0f, 0.20f, 0.20f, 0.22f);

    draw_box_outline(150.0f, -4.5f, 150.0f, 360.0f, 9.0f, 360.0f, 0.18f, 0.90f, 0.92f, 1.5f);
    draw_box_outline(150.0f, 6.0f, 150.0f, 300.0f, 12.0f, 6.0f, 0.98f, 0.78f, 0.25f, 1.4f);
    draw_box_outline(150.0f, 6.0f, 150.0f, 6.0f, 12.0f, 300.0f, 0.98f, 0.78f, 0.25f, 1.4f);

    size_t bcount = 0;
    const TownBuilding *b = town_map_buildings(&bcount);
    int door_count = 0;
    for (size_t i = 0; i < bcount; i++) {
        float base = (b[i].id == BLD_AUCTION_HOUSE) ? 0.09f : 0.06f;
        float j = (hash01((unsigned int)b[i].id + 17U) - 0.5f) * 0.045f;
        float district_x = (b[i].x < 150.0f) ? 0.012f : -0.006f;
        float district_z = (b[i].z > 150.0f) ? 0.010f : -0.004f;
        float r = clampf(base + j + district_x, 0.04f, 0.17f);
        float g = clampf(base + j * 0.7f + district_z, 0.04f, 0.17f);
        float bl = clampf(base + 0.02f + j * 0.6f + district_x * 0.5f, 0.05f, 0.20f);
        draw_box(b[i].x, b[i].h * 0.5f, b[i].z, b[i].w, b[i].h, b[i].d, r, g, bl);

        float or = (b[i].id == BLD_AUCTION_HOUSE || b[i].id == BLD_TOWN_HALL) ? 1.0f : 0.85f;
        float og = (b[i].id == BLD_AUCTION_HOUSE || b[i].id == BLD_TOWN_HALL) ? 0.85f : 0.94f;
        float ob = (b[i].id == BLD_AUCTION_HOUSE || b[i].id == BLD_TOWN_HALL) ? 0.28f : 0.38f;
        draw_box_outline(b[i].x, b[i].h * 0.5f, b[i].z, b[i].w, b[i].h, b[i].d, or, og, ob, 1.6f);

        draw_mock_door(&b[i]);
        door_count++;
    }

    draw_box_outline(150.0f, 12.0f, 156.0f, 108.0f, 2.0f, 7.0f, 1.0f, 0.95f, 0.35f, 2.1f);

    size_t scount = 0;
    const CrisisSocket *s = town_map_sockets(&scount);
    for (size_t i = 0; i < scount; i++) {
        float r = 1.0f, g = 0.4f, bcol = 0.1f;
        if (s[i].id == SOCK_ANCHOR_AUCTION) { r = 0.1f; g = 1.0f; bcol = 1.0f; }
        else if (s[i].id == SOCK_RITUAL_TOWN_HALL) { r = 0.7f; g = 0.2f; bcol = 1.0f; }
        else if (s[i].id == SOCK_HEAD_A_DOCKS || s[i].id == SOCK_HEAD_B_MINES) { r = 1.0f; g = 0.1f; bcol = 0.1f; }
        draw_box(s[i].x, 12.0f, s[i].z, 3.6f, 24.0f, 3.6f, r, g, bcol);
        draw_box_outline(s[i].x, 12.0f, s[i].z, 3.9f, 24.6f, 3.9f, 0.95f, 1.0f, 1.0f, 1.3f);
        draw_ring(s[i].x, s[i].z, 0.2f, s[i].radius, r, g, bcol);
        if (state && state->pressure_idx > 0 && (int)(i % 3) == (state->pressure_idx - 1)) {
            draw_ring(s[i].x, s[i].z, 1.2f, s[i].radius + 3.9f, 1.0f, 1.0f, 1.0f);
        }
    }

    draw_mock_gate(270.0f, 258.0f, 20.0f, 18.0f, 0.95f, 0.72f, 0.25f); // Docks route
    draw_mock_gate(36.0f, 60.0f, 16.0f, 16.0f, 0.25f, 0.95f, 0.95f);   // Secret gate
    draw_mock_gate(270.0f, 54.0f, 18.0f, 18.0f, 0.98f, 0.55f, 0.20f);   // Mines exit
    draw_mock_gate(24.0f, 174.0f, 16.0f, 16.0f, 0.35f, 0.88f, 0.95f);   // Residences route

    if (log_once == 1) {
        log_once = 2;
        printf("[TOWN] mock_doors placed=%d\n", door_count);
        printf("[TOWN] mock_gates placed=%d\n", 4);
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
