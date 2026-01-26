
#include <stdio.h>
#include <assert.h>
#include <string.h>
#include "../../packages/common/protocol.h"

// --- MICRO TEST FRAMEWORK ---
int tests_run = 0;
int tests_passed = 0;

#define ASSERT_EQ(a, b, msg) do { \
    tests_run++; \
    if ((a) != (b)) { \
        printf("❌ FAIL: %s (Expected %d, Got %d)\n", msg, (int)(a), (int)(b)); \
    } else { \
        printf("✅ PASS: %s\n", msg); \
        tests_passed++; \
    } \
} while(0)

#define ASSERT_TRUE(cond, msg) do { \
    tests_run++; \
    if (!(cond)) { \
        printf("❌ FAIL: %s\n", msg); \
    } else { \
        printf("✅ PASS: %s\n", msg); \
        tests_passed++; \
    } \
} while(0)

// --- TESTS ---

void test_usercmd_size() {
    printf("--- Testing Struct Alignment ---\n");
    // We want UserCmd to be reasonably compact.
    // 4(seq) + 4(ts) + 4(fwd) + 4(str) + 4(yaw) + 4(pitch) + 4(wpn) + 1(btn) + 2(msec) = ~31 bytes
    // With padding it might be 32 or 36.
    size_t size = sizeof(UserCmd);
    printf("UserCmd Size: %zu bytes\n", size);
    ASSERT_TRUE(size <= 48, "UserCmd should be under 48 bytes for efficiency");
}

void test_button_bits() {
    printf("--- Testing Button Bitmasks ---\n");
    UserCmd cmd;
    memset(&cmd, 0, sizeof(UserCmd));
    
    // Set Jump and Attack
    cmd.buttons |= BTN_JUMP;
    cmd.buttons |= BTN_ATTACK;
    
    ASSERT_EQ((cmd.buttons & BTN_JUMP), BTN_JUMP, "Jump bit set");
    ASSERT_EQ((cmd.buttons & BTN_ATTACK), BTN_ATTACK, "Attack bit set");
    ASSERT_EQ((cmd.buttons & BTN_CROUCH), 0, "Crouch bit NOT set");
    
    // Toggle Reload
    cmd.buttons |= BTN_RELOAD;
    ASSERT_EQ((cmd.buttons & BTN_RELOAD), BTN_RELOAD, "Reload bit set");
}

void test_sequence_monotonicity() {
    printf("--- Testing Sequence Logic ---\n");
    unsigned int last_seq = 100;
    unsigned int incoming = 101;
    
    ASSERT_TRUE(incoming > last_seq, "New packet is newer (101 > 100)");
    
    incoming = 99; // Out of order / Late
    ASSERT_TRUE(incoming < last_seq, "Old packet detected (99 < 100)");
}

int main() {
    printf("🛡️ SHANKPIT NETCODE REGRESSION SUITE 🛡️\n");
    test_usercmd_size();
    test_button_bits();
    test_sequence_monotonicity();
    
    printf("\n--------------------------------------\n");
    printf("SUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
