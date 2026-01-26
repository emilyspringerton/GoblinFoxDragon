#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <time.h>

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #pragma comment(lib, "ws2_32.lib")
#else
    #include <sys/socket.h>
    #include <netinet/in.h>
    #include <arpa/inet.h>
    #include <unistd.h>
    #include <fcntl.h>
#endif

#include "../../packages/common/protocol.h"
#include "../../packages/common/physics.h"

// --- BOT STATE ---
int sock = -1;
struct sockaddr_in server_addr;
unsigned int bot_seq = 0;
float bot_x = 0, bot_y = 10, bot_z = 0;
float bot_yaw = 0;

// History for stuffing
#define HISTORY_SIZE 64
UserCmd history[HISTORY_SIZE];

void init_network(const char* ip, int port) {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    
    // Non-blocking
    #ifdef _WIN32
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif

    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(port);
    server_addr.sin_addr.s_addr = inet_addr(ip);
    
    printf("[BOT] Network Initialized. Target: %s:%d\n", ip, port);
}

// --- LOGIC: The "Brain" ---
// This mimics client_create_cmd but driven by Math instead of Keyboard
UserCmd bot_think(int tick) {
    UserCmd cmd;
    memset(&cmd, 0, sizeof(UserCmd));
    
    cmd.sequence = ++bot_seq;
    cmd.timestamp = tick * 16;
    cmd.msec = 16;
    
    // BEHAVIOR: Run in a circle
    bot_yaw += 2.0f;
    if (bot_yaw > 360) bot_yaw -= 360;
    
    cmd.yaw = bot_yaw;
    cmd.pitch = 0;
    
    // Always move forward
    cmd.fwd = 1.0f; 
    cmd.str = 0.0f;
    
    // Randomly Jump or Shoot
    if (tick % 60 == 0) cmd.buttons |= BTN_JUMP;
    if (tick % 120 == 0) cmd.buttons |= BTN_ATTACK;
    
    return cmd;
}

void bot_send(UserCmd cmd) {
    // 1. Store
    history[cmd.sequence % HISTORY_SIZE] = cmd;
    
    // 2. Pack (Stuffing Logic)
    char buffer[1024];
    int cursor = 0;
    
    NetHeader head;
    head.type = PACKET_USERCMD;
    head.client_id = 999; // ID for "Bot"
    memcpy(buffer + cursor, &head, sizeof(NetHeader)); cursor += sizeof(NetHeader);
    
    // Redundancy
    int count = 1;
    if (cmd.sequence > 1) count++;
    if (cmd.sequence > 2) count++;
    unsigned char c = (unsigned char)count;
    memcpy(buffer + cursor, &c, 1); cursor += 1;
    
    for(int i=0; i<count; i++) {
        int slot = (cmd.sequence - i) % HISTORY_SIZE;
        memcpy(buffer + cursor, &history[slot], sizeof(UserCmd));
        cursor += sizeof(UserCmd);
    }
    
    sendto(sock, buffer, cursor, 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
}

int main(int argc, char* argv[]) {
    const char* ip = "127.0.0.1";
    int port = 6969;
    
    if (argc > 1) ip = argv[1];
    
    init_network(ip, port);
    
    printf("[BOT] Starting Loop (60Hz)...\n");
    printf("[BOT] FORMAT: Seq | Yaw | Action\n");
    
    int tick = 0;
    while(1) {
        // 1. THINK
        UserCmd cmd = bot_think(tick);
        
        // 2. ACT (Send)
        bot_send(cmd);
        
        // 3. LOG (Chatty)
        if (tick % 60 == 0) { // Log once per second
            char action[32] = "RUN";
            if (cmd.buttons & BTN_JUMP) strcpy(action, "JUMP");
            if (cmd.buttons & BTN_ATTACK) strcpy(action, "SHOOT");
            
            printf("[BOT] Seq: %5d | Yaw: %6.1f | Act: %s | Packet Size: %lu\n", 
                   cmd.sequence, cmd.yaw, action, 
                   sizeof(NetHeader) + 1 + (3*sizeof(UserCmd))); // Approx size with stuffing
        }
        
        // 4. WAIT (Simulate 60Hz)
        // Simple sleep for now
        #ifdef _WIN32
        Sleep(16);
        #else
        usleep(16000);
        #endif
        
        tick++;
    }
    return 0;
}
