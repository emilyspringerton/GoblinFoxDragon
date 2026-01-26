
#include <stdio.h>
#include <math.h>
#include <string.h>

// --- MICRO TEST FRAMEWORK ---
int tests_run = 0;
int tests_passed = 0;
#define ASSERT(cond, msg) do { tests_run++; if (!(cond)) { printf("❌ %s\n", msg); return 0; } else { printf("✅ %s\n", msg); tests_passed++; } } while(0)
#define ASSERT_NEAR(a, b, tol, msg) do { tests_run++; if (fabs((a)-(b)) > (tol)) { printf("❌ %s (Exp %.2f, Got %.2f)\n", msg, (float)(b), (float)(a)); return 0; } else { printf("✅ %s\n", msg); tests_passed++; } } while(0)

// --- MOCK LAG STRUCTURES ---
typedef struct {
    int active;
    unsigned int timestamp;
    float x, y, z;
} LagRecord;

#define MAX_HISTORY 64

// The Interpolation Logic (The "Magic")
int resolve_rewind_pos(LagRecord *history, unsigned int target_time, float *out_x) {
    // 1. Find the two records surrounding target_time
    LagRecord *prev = NULL;
    LagRecord *next = NULL;
    
    // Scan backwards (assuming history is circular or sorted new->old, 
    // but for this mock we'll assume linear sorted OLD -> NEW for simplicity of test logic)
    for (int i=0; i<MAX_HISTORY; i++) {
        if (!history[i].active) continue;
        
        if (history[i].timestamp <= target_time) {
            prev = &history[i];
        }
        if (history[i].timestamp >= target_time) {
            next = &history[i];
            break; // Found the upper bound
        }
    }
    
    if (!prev || !next) return 0; // Can't rewind (Out of bounds)
    
    // 2. Exact Match
    if (prev == next) {
        *out_x = prev->x;
        return 1;
    }
    
    // 3. Interpolate (LERP)
    // Fraction = (Target - Prev) / (Next - Prev)
    float total_dt = (float)(next->timestamp - prev->timestamp);
    float dt = (float)(target_time - prev->timestamp);
    float frac = dt / total_dt;
    
    *out_x = prev->x + (next->x - prev->x) * frac;
    return 1;
}

int test_linear_interpolation() {
    printf("--- Testing Linear Interpolation (LERP) ---\n");
    
    LagRecord history[MAX_HISTORY];
    memset(history, 0, sizeof(history));
    
    // Setup: Player moving linearly along X
    // T=100, X=10
    history[0].active = 1; history[0].timestamp = 100; history[0].x = 10.0f;
    // T=200, X=20
    history[1].active = 1; history[1].timestamp = 200; history[1].x = 20.0f;
    
    float res_x = 0;
    
    // CASE 1: Exact Match (T=100)
    if (resolve_rewind_pos(history, 100, &res_x)) {
        ASSERT_NEAR(res_x, 10.0f, 0.01f, "Rewind to exact frame T=100");
    } else return 0;

    // CASE 2: Dead Center (T=150) -> Should be X=15
    if (resolve_rewind_pos(history, 150, &res_x)) {
        ASSERT_NEAR(res_x, 15.0f, 0.01f, "Rewind to interpolated T=150 (Midpoint)");
    } else return 0;
    
    // CASE 3: 25% (T=125) -> Should be X=12.5
    if (resolve_rewind_pos(history, 125, &res_x)) {
        ASSERT_NEAR(res_x, 12.5f, 0.01f, "Rewind to T=125 (Quarter)");
    } else return 0;
    
    return 1;
}

int test_out_of_bounds() {
    printf("--- Testing Out of Bounds (History Limits) ---\n");
    LagRecord history[MAX_HISTORY];
    memset(history, 0, sizeof(history));
    
    history[0].active = 1; history[0].timestamp = 1000;
    history[1].active = 1; history[1].timestamp = 1100;
    
    float dummy;
    // Request T=500 (Too old)
    ASSERT(resolve_rewind_pos(history, 500, &dummy) == 0, "Reject request older than history");
    
    // Request T=1200 (Future? Should clamp to newest usually, but strictly fails match here)
    // In real implementation we might clamp, but for strict interpolation it fails.
    ASSERT(resolve_rewind_pos(history, 1200, &dummy) == 0, "Reject request newer than history");
    
    return 1;
}

int main() {
    printf("🛡️ LAG COMPENSATION VERIFICATION 🛡️\n");
    test_linear_interpolation();
    test_out_of_bounds();
    
    printf("\nSUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
