#ifndef TOWN_RENDER_H
#define TOWN_RENDER_H

#include "crisis_mock_state.h"

#define TOWN_LABEL_TEXT_MAX 48
#define TOWN_LABEL_MAX 64

typedef enum {
    TOWN_LABEL_LANDMARK = 0,
    TOWN_LABEL_ROUTE = 1,
    TOWN_LABEL_GATE = 2
} TownLabelMode;

typedef struct {
    char text[TOWN_LABEL_TEXT_MAX];
    float x;
    float y;
    float z;
    TownLabelMode mode;
} TownMetaLabel;

void town_render_world(const CrisisMockState *state);
void town_render_route_distances(float px, float pz, char *out, int out_sz);
int town_render_collect_labels(const CrisisMockState *state,
                               float cam_x, float cam_y, float cam_z,
                               TownMetaLabel *out_labels, int max_labels);

#endif
