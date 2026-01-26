
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "../../packages/common/protocol.h"

// --- MICRO TEST FRAMEWORK ---
int tests_run = 0;
int tests_passed = 0;
#define ASSERT(cond, msg) do { tests_run++; if (!(cond)) { printf("❌ %s\n", msg); return 0; } else { printf("✅ %s\n", msg); tests_passed++; } } while(0)

// --- MOCK CLIENT LOGIC (The "Sender") ---
// This represents Requirement 2: client_create_cmd
void client_pack_cmd(UserCmd *cmd, int seq, float yaw, int fire) {
    memset(cmd, 0, sizeof(UserCmd));
    cmd->sequence = seq;
    cmd->timestamp = 123456; // Mock time
    cmd->yaw = yaw;
    if (fire) cmd->buttons |= BTN_ATTACK;
    cmd->msec = 16;
}

// --- MOCK NETWORK (The "Wire") ---
// This represents Requirement 3: Serialization
typedef struct {
    char data[1400];
    int size;
} VirtualPacket;

void net_transmit(VirtualPacket *pkt, NetHeader *head, UserCmd *cmd) {
    pkt->size = 0;
    // 1. Write Header
    memcpy(pkt->data, head, sizeof(NetHeader));
    pkt->size += sizeof(NetHeader);
    
    // 2. Write Body (UserCmd)
    if (cmd) {
        memcpy(pkt->data + pkt->size, cmd, sizeof(UserCmd));
        pkt->size += sizeof(UserCmd);
    }
}

// --- MOCK SERVER LOGIC (The "Receiver") ---
// This represents Requirement 4: Server Validation
int server_receive(VirtualPacket *pkt) {
    if (pkt->size < sizeof(NetHeader)) return 0; // Junk packet
    
    NetHeader *head = (NetHeader*)pkt->data;
    
    if (head->type == PACKET_USERCMD) {
        if (pkt->size != sizeof(NetHeader) + sizeof(UserCmd)) return -1; // Wrong size
        
        UserCmd *cmd = (UserCmd*)(pkt->data + sizeof(NetHeader));
        
        // Validate Content
        printf("   [Server] Recv Seq: %d, Yaw: %.2f, Buttons: %d\n", cmd->sequence, cmd->yaw, cmd->buttons);
        
        if (cmd->sequence != 101) return -2; // Wrong seq
        if (cmd->yaw != 90.0f) return -3;   // Data corruption
        if (!(cmd->buttons & BTN_ATTACK)) return -4; // Lost button press
        
        return 1; // Success
    }
    return 0;
}

// --- THE INTEGRATION TEST ---
int test_full_loop() {
    printf("--- Testing Full Client->Net->Server Loop ---\n");
    
    // 1. Client creates intent
    UserCmd client_cmd;
    client_pack_cmd(&client_cmd, 101, 90.0f, 1); // Seq 101, Facing East, Shooting
    
    // 2. Network Transmission
    NetHeader head;
    head.type = PACKET_USERCMD;
    head.client_id = 1;
    
    VirtualPacket wire;
    net_transmit(&wire, &head, &client_cmd);
    
    ASSERT(wire.size == sizeof(NetHeader) + sizeof(UserCmd), "Packet size matches struct sum");
    
    // 3. Server Processing
    int result = server_receive(&wire);
    
    ASSERT(result == 1, "Server validated packet content (Seq, Yaw, Buttons)");
    return 1;
}

int main() {
    printf("🛡️ LIVE FIRE INTEGRATION TEST 🛡️\n");
    if (test_full_loop()) {
        printf("\nSUMMARY: %d/%d Tests Passed.\n", tests_passed, tests_run);
        return 0;
    }
    return 1;
}
