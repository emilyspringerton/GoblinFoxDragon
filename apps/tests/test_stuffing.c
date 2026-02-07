
#include <stdio.h>
#include <string.h>
#include "../../packages/common/protocol.h"

// --- MICRO TEST FRAMEWORK ---
int tests_run = 0;
int tests_passed = 0;
#define ASSERT(cond, msg) do { tests_run++; if (!(cond)) { printf("❌ %s\n", msg); return 0; } else { printf("✅ %s\n", msg); tests_passed++; } } while(0)

#define MAX_BACKUP_CMDS 2

// Mock Serialization Function (The Logic we want to verify)
int serialize_packet(char *buffer, NetHeader *head, UserCmd *cmds, int count) {
    int cursor = 0;
    
    // 1. Header
    memcpy(buffer + cursor, head, sizeof(NetHeader));
    cursor += sizeof(NetHeader);
    
    // 2. Count (How many commands are inside?)
    // We add a byte for count so the receiver knows how deep to read
    unsigned char cmd_count = (unsigned char)count;
    memcpy(buffer + cursor, &cmd_count, 1);
    cursor += 1;
    
    // 3. Commands (Newest to Oldest)
    for (int i = 0; i < count; i++) {
        memcpy(buffer + cursor, &cmds[i], sizeof(UserCmd));
        cursor += sizeof(UserCmd);
    }
    
    return cursor; // Total size
}

int test_packet_stuffing() {
    printf("--- Testing Redundant Packet Stuffing ---\n");
    
    // Setup Mock History
    UserCmd history[3];
    // Seq 102 (Current)
    history[0].sequence = 102; history[0].buttons = BTN_ATTACK;
    // Seq 101 (Backup 1)
    history[1].sequence = 101; history[1].buttons = BTN_JUMP;
    // Seq 100 (Backup 2)
    history[2].sequence = 100; history[2].buttons = 0;
    
    NetHeader head;
    head.type = PACKET_USERCMD;
    head.client_id = 99;
    head.scene_id = 0;
    
    char buffer[1400];
    int size = serialize_packet(buffer, &head, history, 3);
    
    // CHECK 1: Size Calculation
    int expected_size = sizeof(NetHeader) + 1 + (3 * sizeof(UserCmd));
    printf("Packet Size: %d bytes (Expected %d)\n", size, expected_size);
    ASSERT(size == expected_size, "Packet size matches sum of parts");
    ASSERT(size < 1400, "Packet is well within MTU");
    
    // CHECK 2: Deserialize / Verify Integrity
    int cursor = 0;
    NetHeader *out_head = (NetHeader*)(buffer + cursor);
    cursor += sizeof(NetHeader);
    
    unsigned char out_count = *(unsigned char*)(buffer + cursor);
    cursor += 1;
    
    ASSERT(out_head->type == PACKET_USERCMD, "Header Type Preserved");
    ASSERT(out_count == 3, "Command Count Preserved");
    
    UserCmd *out_cmd_current = (UserCmd*)(buffer + cursor);
    ASSERT(out_cmd_current->sequence == 102, "Primary Command (102) is first");
    ASSERT(out_cmd_current->buttons == BTN_ATTACK, "Primary payload intact");
    
    cursor += sizeof(UserCmd);
    UserCmd *out_cmd_backup = (UserCmd*)(buffer + cursor);
    ASSERT(out_cmd_backup->sequence == 101, "Backup Command (101) is second");
    ASSERT(out_cmd_backup->buttons == BTN_JUMP, "Backup payload intact");
    
    return 1;
}

int main() {
    printf("🛡️ PACKET STUFFING VERIFICATION 🛡️\n");
    if (test_packet_stuffing()) {
        printf("\nSUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
        return 0;
    }
    return 1;
}
