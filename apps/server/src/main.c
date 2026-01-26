
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <math.h>

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

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"
#include "../../../packages/simulation/local_game.h" // Re-using the sim logic

// --- SERVER STATE ---
int sock = -1;
struct sockaddr_in bind_addr;

// Track last processed sequence per client to reject duplicates
unsigned int client_last_seq[MAX_CLIENTS]; 


// --- HEADLESS TIMER (No SDL Required) ---
unsigned int get_server_time() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (unsigned int)(ts.tv_sec * 1000 + ts.tv_nsec / 1000000);
}

void server_net_init() {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    
    #ifdef _WIN32
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif

    bind_addr.sin_family = AF_INET;
    bind_addr.sin_port = htons(6969); // Standard ShankPort
    bind_addr.sin_addr.s_addr = INADDR_ANY;
    
    if (bind(sock, (struct sockaddr*)&bind_addr, sizeof(bind_addr)) < 0) {
        printf("❌ FAILED TO BIND PORT 6969\n");
    } else {
        printf("✅ SERVER LISTENING ON PORT 6969\n");
    }
}

// --- THE JUDGE LOGIC (From Test Suite) ---
void process_user_cmd(int client_id, UserCmd *cmd) {
    if (cmd->sequence <= client_last_seq[client_id]) return; // Old/Duplicate

    // 1. Sanitize (Phase 305-B)
    if (isnan(cmd->yaw) || isinf(cmd->yaw)) cmd->yaw = 0;
    if (isnan(cmd->pitch) || isinf(cmd->pitch)) cmd->pitch = 0;
    
    // 2. Execute Physics (The Truth)
    // We map the UserCmd back to the raw inputs local_update expects
    float fwd = cmd->fwd;
    float str = cmd->str;
    int shoot = (cmd->buttons & BTN_ATTACK);
    int jump = (cmd->buttons & BTN_JUMP);
    int crouch = (cmd->buttons & BTN_CROUCH);
    int reload = (cmd->buttons & BTN_RELOAD);
    
    // UPDATE THE SPECIFIC PLAYER
    // Note: In a real engine we'd pass client_id to local_update. 
    // For now, we hack it to update Player 0 or specific ID if local_game supports it.
    // Assuming local_game updates 'local_state.players[0]' by default, we need to adapt.
    // We will assume a helper 'server_update_player' exists or we do it manually.
    
    // Manual Update for transparency:
    PlayerState *p = &local_state.players[client_id];
    p->yaw = cmd->yaw;
    p->pitch = cmd->pitch;
    p->current_weapon = cmd->weapon_idx;
    
    // Use the physics engine (re-using client logic for now, but authoritative)
    // We temporarily override the "inputs" in the player struct if needed, 
    // or just call logic.
    // For this Alpha, we trust the movement logic is shared.
    
    // CRITICAL: LAG COMP HOOK (Req 5 - Rewind)
    // If shooting, we must rewind OTHERS before checking hits.
    // But local_update handles shooting internally. 
    // We will implement the rewind hook inside physics.h later.
    
    client_last_seq[client_id] = cmd->sequence;
}

void server_handle_packet(struct sockaddr_in *sender, char *buffer, int size) {
    if (size < sizeof(NetHeader)) return;
    
    NetHeader *head = (NetHeader*)buffer;
    
    // Simple Client ID assignment (Hash IP or just Slot 1 for now)
    int client_id = 1; // TODO: Real connection manager
    
    if (head->type == PACKET_USERCMD) {
        int cursor = sizeof(NetHeader);
        unsigned char count = *(unsigned char*)(buffer + cursor);
        cursor += 1;
        
        UserCmd *cmds = (UserCmd*)(buffer + cursor);
        
        // DE-STUFF: Execute Oldest -> Newest
        for (int i = count - 1; i >= 0; i--) {
            process_user_cmd(client_id, &cmds[i]);
        }
        
        // HEARTBEAT: Mark client as active/alive
        local_state.players[client_id].active = 1;
    }
}

int main() {
    server_net_init();
    local_init_match(1); // Init physics state
    
    printf("🛡️ SHANKPIT DEDICATED SERVER v0.2 (AUTHORITATIVE) 🛡️\n");
    
    int running = 1;
    unsigned int tick = 0;
    
    while(running) {
        // 1. NETWORK RECEIVE
        char buffer[1024];
        struct sockaddr_in sender;
        socklen_t slen = sizeof(sender);
        
        int len = recvfrom(sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        while (len > 0) {
            server_handle_packet(&sender, buffer, len);
            len = recvfrom(sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        }
        
        // 2. SIMULATION STEP
        // In a real server, we step physics here independent of inputs.
        // For now, inputs drive the sim steps.
        
        // 3. LAG COMP HISTORY (Req 5)
        // Every tick, store the state of all active players
        for(int i=0; i<MAX_CLIENTS; i++) {
            if (local_state.players[i].active) {
                phys_store_history(&local_state, i, get_server_time()); 
            }
        }
        
        // 4. BROADCAST SNAPSHOT (To be implemented)
        // We need to tell clients where everyone is.
        
        // 5. TICK RATE (60Hz)
        #ifdef _WIN32
        Sleep(16);
        #else
        usleep(16000);
        #endif
        tick++;
    }
    return 0;
}
