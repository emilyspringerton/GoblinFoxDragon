#ifndef TOWN_RENDER_H
#define TOWN_RENDER_H
#include "crisis_mock_state.h"
void town_render_world(const CrisisMockState *state);
void town_render_route_distances(float px, float pz, char *out, int out_sz);
#endif
