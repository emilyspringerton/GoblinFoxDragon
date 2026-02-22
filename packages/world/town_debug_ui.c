#include "town_debug_ui.h"
#include <stdio.h>
#include <SDL2/SDL_opengl.h>
#include "../ui/turtle_text.h"
#include "crisis_mock_state.h"

static void draw_string2(const char *str, float x, float y, float size) {
    TurtlePen pen = turtle_pen_create(x, y, size);
    turtle_draw_text(&pen, str);
}

void town_debug_ui_draw(const CrisisMockState *s, const char *routes_line) {
    if (!s || !s->overlay_on) return;
    char line[256];
    glColor3f(1.0f, 0.5f, 0.2f);
    draw_string2("WORLD CRISIS: THE SUNDERWORM", 30, 690, 7);
    glColor3f(0.8f, 0.9f, 1.0f);
    snprintf(line, sizeof(line), "Phase: %s", crisis_mock_phase_name(s->phase));
    draw_string2(line, 30, 665, 5);
    snprintf(line, sizeof(line), "Ley Integrity: %.0f%%", s->ley_integrity);
    draw_string2(line, 30, 645, 5);
    snprintf(line, sizeof(line), "Escalation ETA: %ds", s->time_to_next_sec);
    draw_string2(line, 30, 625, 5);
    snprintf(line, sizeof(line), "Gate: %d/%d   A:%s R:%s I:%s", s->gate_current, s->gate_required,
             s->gate_anchor_met ? "Y" : "N", s->gate_ritual_met ? "Y" : "N", s->gate_intercept_met ? "Y" : "N");
    draw_string2(line, 30, 605, 5);
    draw_string2(crisis_mock_phase_prompt(s->phase), 30, 585, 4);
    if (routes_line && routes_line[0]) {
        glColor3f(0.9f, 0.8f, 0.2f);
        draw_string2(routes_line, 30, 560, 4);
    }
}
