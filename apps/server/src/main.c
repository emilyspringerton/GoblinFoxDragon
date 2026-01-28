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

void server_net_init() {
    setbuf(stdout, NULL);
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
    bind_addr.sin_port = htons(6969); 
    bind_addr.sin_addr.s_addr = INADDR_ANY;
    if (bind(sock, (struct sockaddr*)&bind_addr, sizeof(bind_addr)) < 0) {
        printf("FAILED TO BIND PORT 6969\n");
        exit(1);
    }
    printf("SERVER LISTENING ON PORT 6969\n");
}

void process_user_cmd(int client_id, UserCmd *cmd) {
    if (cmd->sequence <= client_last_seq[client_id]) return; 
    PlayerState *p = &local_state.players[client_id];
    p->yaw = cmd->yaw; p->pitch = cmd->pitch;
    p->in_fwd = cmd->fwd; p->in_strafe = cmd->str;
    p->in_jump = (cmd->buttons & BTN_JUMP);
    p->in_shoot = (cmd->buttons & BTN_ATTACK);
    p->crouching = (cmd->buttons & BTN_CROUCH);
    p->in_reload = (cmd->buttons & BTN_RELOAD);
    if (cmd->weapon_idx >= 0 && cmd->weapon_idx < MAX_WEAPONS) p->current_weapon = cmd->weapon_idx;
    client_last_seq[client_id] = cmd->sequence;
}

void server_handle_packet(struct sockaddr_in *sender, char *buffer, int size) {
    if (size < sizeof(NetHeader)) return;
    NetHeader *head = (NetHeader*)buffer;
    int client_id = -1;
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.client_active[i] && 
            memcmp(&local_state.clients[i].sin_addr, &sender->sin_addr, sizeof(struct in_addr)) == 0 &&
            local_state.clients[i].sin_port == sender->sin_port) {
            client_id = i; break;
        }
    }
    if (client_id == -1 && head->type == PACKET_CONNECT) {
        for(int i=1; i<MAX_CLIENTS; i++) {
            if (!local_state.client_active[i]) {
                client_id = i;
                local_state.client_active[i] = 1;
                local_state.clients[i] = *sender;
                local_state.players[i].active = 1;
                phys_respawn(&local_state.players[i], get_server_time());
                printf("CLIENT %d CONNECTED\n", client_id);
                NetHeader h = {PACKET_WELCOME, (unsigned char)client_id, 0, get_server_time(), 0};
                sendto(sock, (char*)&h, sizeof(NetHeader), 0, (struct sockaddr*)sender, sizeof(struct sockaddr_in));
                break;
            }
        }
    }
    if (client_id != -1 && head->type == PACKET_USERCMD) {
        int cursor = sizeof(NetHeader);
        unsigned char count = *(unsigned char*)(buffer + cursor); cursor++;
        if (size >= cursor + (count * sizeof(UserCmd))) {
            UserCmd *cmds = (UserCmd*)(buffer + cursor);
            for (int i = 0; i < count; i++) process_user_cmd(client_id, &cmds[i]);
        }
    }
}

void server_broadcast() {
    char buffer[4096]; int cursor = 0;
    NetHeader head = {PACKET_SNAPSHOT, 0, (unsigned short)local_state.server_tick, get_server_time(), 0};
    unsigned char count = 0;
    for(int i=0; i<MAX_CLIENTS; i++) if (local_state.players[i].active) count++;
    head.entity_count = count;
    memcpy(buffer + cursor, &head, sizeof(NetHeader)); cursor += sizeof(NetHeader);
    memcpy(buffer + cursor, &count, 1); cursor += 1;
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (p->active) {
            NetPlayer np = {(unsigned char)i, p->x, p->y, p->z, p->yaw, p->pitch, (unsigned char)p->current_weapon, (unsigned char)p->state, (unsigned char)p->health, (unsigned char)p->shield, (unsigned char)p->is_shooting, (unsigned char)p->crouching, 0, (unsigned char)p->ammo[p->current_weapon], (unsigned char)p->in_vehicle, (unsigned char)p->hit_feedback};
            p->hit_feedback = 0;
            memcpy(buffer + cursor, &np, sizeof(NetPlayer)); cursor += sizeof(NetPlayer);
        }
    }
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.client_active[i]) sendto(sock, buffer, cursor, 0, (struct sockaddr*)&local_state.clients[i], sizeof(struct sockaddr_in));
    }
}

int main() {
    server_net_init();
    local_init_match(1, 0); 
    printf("[STRESS-TEST] Spawning 26 bots...\n");
    for(int i = 1; i <= 26; i++) {
        local_state.client_active[i] = 1;
        local_state.players[i].active = 1;
        local_state.players[i].is_bot = 1;
        phys_respawn(&local_state.players[i], get_server_time());
    }
    while(1) {
        char buffer[2048]; struct sockaddr_in sender; socklen_t slen = sizeof(sender);
        int len = recvfrom(sock, buffer, 2048, 0, (struct sockaddr*)&sender, &slen);
        while (len > 0) {
            server_handle_packet(&sender, buffer, len);
            len = recvfrom(sock, buffer, 2048, 0, (struct sockaddr*)&sender, &slen);
        }
        unsigned int now = get_server_time();
        for(int i=0; i<MAX_CLIENTS; i++) {
            if (local_state.players[i].active) update_entity(&local_state.players[i], 0.016f, NULL, now);
        }
        server_broadcast();
        local_state.server_tick++;
        #ifdef _WIN32
        Sleep(16);
        #else
        usleep(16000);
        #endif
    }
    return 0;
}
