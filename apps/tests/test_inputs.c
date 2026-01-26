
#include <stdio.h>
#include <assert.h>
#include "../../packages/common/protocol.h"

// --- MICRO TEST FRAMEWORK ---
int tests_run = 0;
int tests_passed = 0;
#define ASSERT(cond, msg) do { tests_run++; if (!(cond)) { printf("❌ %s\n", msg); return 0; } else { printf("✅ %s\n", msg); tests_passed++; } } while(0)

// Helper to simulate the logic we will use in main.c
unsigned char pack_buttons(int attack, int jump, int crouch, int reload) {
    unsigned char buttons = 0;
    if (attack) buttons |= BTN_ATTACK;
    if (jump)   buttons |= BTN_JUMP;
    if (crouch) buttons |= BTN_CROUCH;
    if (reload) buttons |= BTN_RELOAD;
    return buttons;
}

int test_packing() {
    printf("--- Testing Input Bitmask Packing ---\n");

    // 1. Single Button
    unsigned char b = pack_buttons(1, 0, 0, 0);
    ASSERT(b == BTN_ATTACK, "Attack maps correctly");
    
    // 2. Multi-Button (The "Bunny Hop Shot")
    b = pack_buttons(1, 1, 0, 0);
    ASSERT(b == (BTN_ATTACK | BTN_JUMP), "Attack + Jump combines correctly");
    ASSERT(b & BTN_ATTACK, "Combo has Attack");
    ASSERT(b & BTN_JUMP, "Combo has Jump");
    ASSERT(!(b & BTN_CROUCH), "Combo does NOT have Crouch");

    // 3. Full House
    b = pack_buttons(1, 1, 1, 1);
    ASSERT(b == (BTN_ATTACK | BTN_JUMP | BTN_CROUCH | BTN_RELOAD), "All buttons active");

    return 1;
}

int main() {
    printf("🛡️ INPUT LOGIC VERIFICATION 🛡️\n");
    if (test_packing()) {
        printf("\nSUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
        return 0;
    }
    return 1;
}
