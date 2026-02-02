#include <stdio.h>
#include <string.h>

#ifndef _WIN32
#include <netinet/in.h>
#endif

#include "../server/src/server_state.h"

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

static void test_disconnect_clears_state(void) {
    printf("--- Testing Disconnect Clears State ---\n");
    memset(&local_state, 0, sizeof(local_state));

    unsigned int client_last_seq[MAX_CLIENTS];
    memset(client_last_seq, 0, sizeof(client_last_seq));

    int slot = 3;
    local_state.players[slot].active = 1;
    local_state.players[slot].id = slot;
    local_state.players[slot].health = 75;
    local_state.players[slot].x = 10.0f;
    local_state.client_meta[slot].active = 1;
    local_state.client_meta[slot].last_heard_ms = 12345;
    local_state.clients[slot].sin_port = 7777;
    local_state.clients[slot].sin_addr.s_addr = 0x01020304;
    client_last_seq[slot] = 42;

    server_disconnect(slot, client_last_seq);

    ASSERT_EQ(local_state.players[slot].active, 0, "Player active cleared");
    ASSERT_EQ(local_state.players[slot].id, 0, "Player id reset");
    ASSERT_EQ(local_state.players[slot].health, 0, "Player health reset");
    ASSERT_EQ(local_state.client_meta[slot].active, 0, "Client meta inactive");
    ASSERT_EQ(local_state.client_meta[slot].last_heard_ms, 0, "Client last heard reset");
    ASSERT_EQ(client_last_seq[slot], 0, "Client last seq reset");
    ASSERT_EQ(local_state.clients[slot].sin_port, 0, "Client port reset");
    ASSERT_EQ(local_state.clients[slot].sin_addr.s_addr, 0, "Client addr reset");
}

int main(void) {
    printf("🛡️ SHANKPIT SERVER STATE CHECK 🛡️\n");
    test_disconnect_clears_state();
    printf("\n--------------------------------------\n");
    printf("SUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
