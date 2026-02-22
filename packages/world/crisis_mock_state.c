#include "crisis_mock_state.h"
#include <SDL2/SDL.h>

void crisis_mock_state_init(CrisisMockState *s) {
    s->phase = MOCK_PHASE_OMENS;
    s->ley_integrity = 100.0f;
    s->time_to_next_sec = 180;
    s->gate_required = 3;
    s->gate_current = 0;
    s->gate_anchor_met = 0;
    s->gate_ritual_met = 0;
    s->gate_intercept_met = 0;
    s->overlay_on = 1;
    s->labels_on = 1;
    s->pressure_idx = 0;
    s->topdown_on = 0;
    s->noclip_on = 0;
}

const char *crisis_mock_phase_name(MockPhase p) {
    static const char *names[] = {"OMENS", "BURROW", "EMERGENCE", "SPLIT WAR", "FINAL WINDOW", "RESOLUTION"};
    return names[(int)p < 0 || (int)p > 5 ? 0 : (int)p];
}

const char *crisis_mock_phase_prompt(MockPhase p) {
    switch (p) {
        case MOCK_PHASE_OMENS: return "Scout reports and prep town routes.";
        case MOCK_PHASE_BURROW: return "Harden anchor lines and stage intercept teams.";
        case MOCK_PHASE_EMERGENCE: return "Contest ritual points and hold central spine.";
        case MOCK_PHASE_SPLIT_WAR: return "Split force to docks/mines heads now.";
        case MOCK_PHASE_FINAL_WINDOW: return "Seal final gate. Burn priority targets.";
        default: return "Crisis stabilized. Reset and review routes.";
    }
}

static float clampf(float v, float lo, float hi) { return v < lo ? lo : (v > hi ? hi : v); }

void crisis_mock_state_handle_key(CrisisMockState *s, int key) {
    if (key >= SDLK_1 && key <= SDLK_6) s->phase = (MockPhase)(key - SDLK_1);
    else if (key == SDLK_LEFTBRACKET) s->ley_integrity = clampf(s->ley_integrity - 5.0f, 0.0f, 100.0f);
    else if (key == SDLK_RIGHTBRACKET) s->ley_integrity = clampf(s->ley_integrity + 5.0f, 0.0f, 100.0f);
    else if (key == SDLK_MINUS) s->gate_current = s->gate_current > 0 ? s->gate_current - 1 : 0;
    else if (key == SDLK_EQUALS) s->gate_current = s->gate_current < 9 ? s->gate_current + 1 : 9;
    else if (key == SDLK_F1) s->overlay_on = !s->overlay_on;
    else if (key == SDLK_F2) s->labels_on = !s->labels_on;
    else if (key == SDLK_F3) s->pressure_idx = (s->pressure_idx + 1) % 3;
    else if (key == SDLK_z) s->gate_anchor_met = !s->gate_anchor_met;
    else if (key == SDLK_x) s->gate_ritual_met = !s->gate_ritual_met;
    else if (key == SDLK_c) s->gate_intercept_met = !s->gate_intercept_met;
    else if (key == SDLK_m) s->topdown_on = !s->topdown_on;
    else if (key == SDLK_n) s->noclip_on = !s->noclip_on;
}
