
#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <time.h>

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #pragma comment(lib, "ws2_32.lib")
    #define SLEEP(x) Sleep(x)
#else
    #include <sys/socket.h>
    #include <netinet/in.h>
    #include <arpa/inet.h>
    #include <unistd.h>
    #include <fcntl.h>
    #include <sys/ioctl.h>
    #define SOCKET int
    #define INVALID_SOCKET -1
    #define SOCKET_ERROR -1
    #define SLEEP(x) usleep((x)*1000)
#endif

#include "../../../packages/protocol/protocol.h"
#include "../../../packages/simulation/game_physics.h"
#include "../../../packages/map/map.h" 
#include "../../../packages/simulation/local_game.h"

ServerState local_state;
int local_pid = -1; // <-- FIXED: Defined here

long long current_time_ms() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (long long)(ts.tv_sec * 1000 + ts.tv_nsec / 1000000);
}

int main() {
    SOCKET sock = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(PORT);
    server_addr.sin_addr.s_addr = INADDR_ANY;
    int opt = 1; setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    if (bind(sock, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) return 1;
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);

    printf("🔥 SERVER v40 ONLINE PORT %d\n", PORT);
    local_init();
    strcpy(local_state.status_msg, "SERVER v40 READY");

    struct sockaddr_in clients[MAX_CLIENTS];
    int client_active[MAX_CLIENTS] = {0};
    long long last_seen[MAX_CLIENTS] = {0};

    while(1) {
        char buf[4096]; struct sockaddr_in from; 
        socklen_t fromlen = sizeof(from);
        long long now = current_time_ms();
        int len;
        
        while ((len = recvfrom(sock, buf, sizeof(buf), 0, (struct sockaddr*)&from, &fromlen)) > 0) {
            if (len < sizeof(int)*3) continue; 
            Packet *pkt = (Packet*)buf;
            
            // Identify Sender
            int sender_pid = -1;
            for(int i=0; i<MAX_CLIENTS; i++) {
                if (client_active[i] && clients[i].sin_addr.s_addr == from.sin_addr.s_addr && clients[i].sin_port == from.sin_port) {
                    sender_pid = i; break;
                }
            }

            if (pkt->type == PACKET_LOBBY_CMD) {
                if (pkt->cmd_id == CMD_SEARCH) {
                    Packet out; out.type = PACKET_LOBBY_INFO;
                    int count = 0; for(int i=0;i<MAX_CLIENTS;i++) if(client_active[i]) count++;
                    local_state.player_count = count;
                    sprintf(local_state.status_msg, "v40 | %d/%d PLAYERS", count, MAX_CLIENTS);
                    memcpy(out.data, &local_state, sizeof(ServerState));
                    sendto(sock, (char*)&out, sizeof(int)*3 + sizeof(ServerState), 0, (struct sockaddr*)&from, fromlen);
                }
                else if (pkt->cmd_id == CMD_CREATE) {
                    local_init();
                    memset(client_active, 0, sizeof(client_active));
                    strcpy(local_state.status_msg, "MATCH RESET (v40)");
                }
                else if (pkt->cmd_id == CMD_NAME) {
                    if (sender_pid != -1) {
                        strncpy(local_state.players[sender_pid].name, pkt->data, 31);
                        printf("Updated Name PID %d -> %s\n", sender_pid, local_state.players[sender_pid].name);
                    }
                }
                continue;
            }

            // Join
            if (sender_pid == -1 && pkt->type == PACKET_INPUT) {
                for(int i=0; i<MAX_CLIENTS; i++) {
                    if (i != 1 && !client_active[i] && !local_state.players[i].active) {
                        client_active[i] = 1; clients[i] = from; sender_pid = i;
                        local_state.players[i].active = 1;
                        local_state.players[i].health = 100;
                        local_state.players[i].pos.y = 5.0f;
                        sprintf(local_state.players[i].name, "Player %d", i); 
                        for(int j=0; j<MAX_WEAPONS; j++) local_state.players[i].ammo[j] = WPN_STATS[j].ammo_max;
                        printf("Join v40: PID %d\n", sender_pid);
                        break;
                    }
                }
            }

            // Input
            if (sender_pid != -1) {
                last_seen[sender_pid] = now;
                if (pkt->type == PACKET_INPUT) {
                    if (len >= sizeof(int)*3 + sizeof(ClientInput)) {
                        ClientInput in; memcpy(&in, pkt->data, sizeof(ClientInput));
                        local_pid = sender_pid; // Set Global for Update
                        local_update(0.016f, in.fwd, in.strafe, in.yaw, in.pitch, in.shoot, in.weapon_req, in.jump, in.crouch, in.reload);
                    }
                }
            }
        }

        for(int i=0; i<MAX_CLIENTS; i++) {
            if (client_active[i] && (now - last_seen[i] > 5000)) {
                printf("Timeout: PID %d\n", i); client_active[i] = 0; local_state.players[i].active = 0;
            }
        }
        
        if (!client_active[1]) {
             PlayerState *bot = &local_state.players[1]; bot->active = 1;
             strcpy(bot->name, "BOT ALPHA");
             bot->yaw += 2.0f;
             bot->pos.x = 5.0f + sinf(now*0.001f)*3.0f; bot->pos.z = 15.0f + cosf(now*0.001f)*3.0f;
             if ((now/500)%4==0) { bot->current_weapon = 2; bot->is_shooting = 5; } else { bot->is_shooting = 0; }
        }

        for(int i=0; i<MAX_CLIENTS; i++) {
            if (client_active[i]) {
                Packet out; out.type = PACKET_STATE; out.owner_id = i; 
                memcpy(out.data, &local_state, sizeof(local_state));
                sendto(sock, (char*)&out, sizeof(int)*3 + sizeof(ServerState), 0, (struct sockaddr*)&clients[i], sizeof(clients[i]));
            }
        }
        SLEEP(16);
    }
    return 0;
}
