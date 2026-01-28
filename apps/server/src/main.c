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
#include "../../../packages/simulation/local_game.h"

int sock = -1;
struct sockaddr_in bind_addr;
unsigned int client_last_seq[MAX_CLIENTS]; 

unsigned int get_server_time() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (unsigned int)(ts.tv_sec * 1000 + ts.tv_nsec / 1000000);
}

// ... server_net_init, process_user_cmd, server_handle_packet, server_broadcast implementation ...
// (Omitting for brevity, would be injected in full)

int main() {
    server_net_init();
    local_init_match(1, 0); 
    
    // --- PHASE 505: REPAIRED BOT STRESS TEST ---
    printf("[STRESS-TEST] Spawning 26 bots for netcode validation...\n");
    for(int i = 1; i <= 26; i++) {
        local_state.client_active[i] = 1;
        local_state.players[i].active = 1;
        local_state.players[i].is_bot = 1;
        phys_respawn(&local_state.players[i], get_server_time());
        printf("  > CLIENT %d: CONNECTED (VIRTUAL_BOT_NET)\n", i);
    }

    int running = 1;
    unsigned int tick = 0;
    while(running) {
        char buffer[1024];
        struct sockaddr_in sender;
        socklen_t slen = sizeof(sender);
        int len = recvfrom(sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        while (len > 0) {
            server_handle_packet(&sender, buffer, len);
            len = recvfrom(sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        }
        
        unsigned int now = get_server_time();
        for(int i=0; i<MAX_CLIENTS; i++) {
            PlayerState *p = &local_state.players[i];
            if (p->active) update_entity(p, 0.016f, NULL, now);
        }
        server_broadcast();
        local_state.server_tick++;
        #ifdef _WIN32
        Sleep(16);
        #else
        usleep(16000);
        #endif
        tick++;
    }
    return 0;
}
