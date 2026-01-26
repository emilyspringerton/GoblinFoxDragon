
#include <stdio.h>
#include <string.h>
#include "../../packages/common/protocol.h"

// --- MICRO TEST FRAMEWORK ---
int tests_run = 0;
int tests_passed = 0;
#define ASSERT(cond, msg) do { tests_run++; if (!(cond)) { printf("❌ %s\n", msg); return 0; } else { printf("✅ %s\n", msg); tests_passed++; } } while(0)

// --- MOCK SERVER ---
int server_last_seq = 99; // Server has processed up to 99
int processed_log[10];    // Log of what sequences we ran
int log_count = 0;

void mock_process_cmd(UserCmd *cmd) {
    if (cmd->sequence <= server_last_seq) return; // Discard duplicates
    
    // Execute!
    processed_log[log_count++] = cmd->sequence;
    server_last_seq = cmd->sequence;
}

// The Logic We Want to Test (The De-Stuffer)
void server_handle_packet(char *buffer, int size) {
    int cursor = 0;
    
    // 1. Skip Header (Already validated in real life)
    cursor += sizeof(NetHeader);
    
    // 2. Read Count
    unsigned char count = *(unsigned char*)(buffer + cursor);
    cursor += 1;
    
    // 3. De-Stuff (The Buffer is ordered Newest -> Oldest)
    // BUT we must execute Oldest -> Newest.
    // So we need to peek at them or iterate backwards.
    
    UserCmd *cmds_in_packet = (UserCmd*)(buffer + cursor);
    
    // Iterate from the BACK (Oldest) to the FRONT (Newest)
    for (int i = count - 1; i >= 0; i--) {
        mock_process_cmd(&cmds_in_packet[i]);
    }
}

int test_gap_fill() {
    printf("--- Testing Packet Loss Recovery (Gap Fill) ---\n");
    
    // Reset
    server_last_seq = 99;
    log_count = 0;
    
    // Create a Stuffed Packet (Contains 102, 101, 100)
    // Server has 99. It missed 100 and 101.
    // This packet should trigger 100 -> 101 -> 102.
    
    char buffer[1024];
    int cursor = 0;
    
    NetHeader head; head.type = PACKET_USERCMD;
    memcpy(buffer + cursor, &head, sizeof(NetHeader)); cursor += sizeof(NetHeader);
    
    unsigned char count = 3;
    memcpy(buffer + cursor, &count, 1); cursor += 1;
    
    UserCmd cmds[3];
    cmds[0].sequence = 102; // Newest
    cmds[1].sequence = 101;
    cmds[2].sequence = 100; // Oldest
    
    memcpy(buffer + cursor, cmds, sizeof(UserCmd) * 3);
    
    // RUN THE LOGIC
    server_handle_packet(buffer, cursor + (sizeof(UserCmd)*3));
    
    // VERIFY
    ASSERT(log_count == 3, "Executed 3 commands");
    ASSERT(processed_log[0] == 100, "First was 100");
    ASSERT(processed_log[1] == 101, "Second was 101");
    ASSERT(processed_log[2] == 102, "Third was 102");
    ASSERT(server_last_seq == 102, "Server state updated to 102");
    
    return 1;
}

int test_duplicate_rejection() {
    printf("--- Testing Duplicate Rejection ---\n");
    
    // Reset: Server is already at 102
    server_last_seq = 102;
    log_count = 0;
    
    // Receive the SAME packet (102, 101, 100)
    // Should execute NOTHING.
    
    char buffer[1024];
    // ... (Same setup as above, simplified for brevity)
    int cursor = sizeof(NetHeader);
    unsigned char count = 3;
    memcpy(buffer + cursor, &count, 1); cursor += 1;
    UserCmd cmds[3];
    cmds[0].sequence = 102; cmds[1].sequence = 101; cmds[2].sequence = 100;
    memcpy(buffer + cursor, cmds, sizeof(UserCmd) * 3);
    
    server_handle_packet(buffer, 0); // Size doesn't matter for this logic mockup
    
    ASSERT(log_count == 0, "Zero commands executed");
    ASSERT(server_last_seq == 102, "Server state unchanged");
    
    return 1;
}

int main() {
    printf("🛡️ SERVER LOGIC VERIFICATION 🛡️\n");
    test_gap_fill();
    test_duplicate_rejection();
    
    printf("\nSUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
