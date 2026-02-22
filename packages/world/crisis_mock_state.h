#ifndef CRISIS_MOCK_STATE_H
#define CRISIS_MOCK_STATE_H

typedef enum { MOCK_PHASE_OMENS = 0, MOCK_PHASE_BURROW, MOCK_PHASE_EMERGENCE, MOCK_PHASE_SPLIT_WAR, MOCK_PHASE_FINAL_WINDOW, MOCK_PHASE_RESOLUTION } MockPhase;
typedef struct {
    MockPhase phase;
    float ley_integrity;
    int time_to_next_sec;
    int gate_required;
    int gate_current;
    int gate_anchor_met;
    int gate_ritual_met;
    int gate_intercept_met;
    int overlay_on;
    int labels_on;
    int pressure_idx;
    int topdown_on;
    int noclip_on;
} CrisisMockState;

void crisis_mock_state_init(CrisisMockState *state);
void crisis_mock_state_handle_key(CrisisMockState *state, int keycode);
const char *crisis_mock_phase_name(MockPhase phase);
const char *crisis_mock_phase_prompt(MockPhase phase);

#endif
