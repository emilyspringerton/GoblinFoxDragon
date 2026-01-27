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

// --- SERVER STATE ---
int sock = -1;
struct sockaddr_in bind_addr;
unsigned int client_last_seq[MAX_CLIENTS]; 

unsigned int get_server_time() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (unsigned int)(ts.tv_sec * 1000 + ts.tv_nsec / 1000000);
}

void server_net_init() {
    // DISABLE BUFFERING
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
        printf("FAILED TO BIND PORT 6969 (Is another server running?)\n");
        exit(1);
    } else {
        printf("SERVER LISTENING ON PORT 6969\n");
        printf("Waiting for connections...\n");
    }
}

void process_user_cmd(int client_id, UserCmd *cmd) {
    if (cmd->sequence <= client_last_seq[client_id]) return; 
    
    PlayerState *p = &local_state.players[client_id];
    p->yaw = cmd->yaw;
    p->pitch = cmd->pitch;
    p->in_fwd = cmd->fwd;
    p->in_strafe = cmd->str;
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
    
    // 1. Identify Client
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.client_active[i] && 
            memcmp(&local_state.clients[i].sin_addr, &sender->sin_addr, sizeof(struct in_addr)) == 0 &&
            local_state.clients[i].sin_port == sender->sin_port) {
            client_id = i;
            break;
        }
    }
    
    // 2. New Connection
    if (client_id == -1) {
        char *ip_str = inet_ntoa(sender->sin_addr);
        printf("PACKET FROM NEW IP: %s:%d (Type: %d)\n", ip_str, ntohs(sender->sin_port), head->type);

        for(int i=1; i<MAX_CLIENTS; i++) {
            if (!local_state.client_active[i]) {
                client_id = i;
                local_state.client_active[i] = 1;
                local_state.clients[i] = *sender;
                local_state.players[i].active = 1;
                
                phys_respawn(&local_state.players[i], get_server_time());
                for(int w=0; w<MAX_WEAPONS; w++) local_state.players[i].ammo[w] = WPN_STATS[w].ammo_max;

                printf("CLIENT %d ASSIGNED to %s\n", client_id, ip_str);
                break;
            }
        }
    }
    
    // 3. Process Payload
    if (client_id != -1 && head->type == PACKET_USERCMD) {
        int cursor = sizeof(NetHeader);
        unsigned char count = *(unsigned char*)(buffer + cursor); cursor += 1;
        
        if (size >= cursor + (count * sizeof(UserCmd))) {
            UserCmd *cmds = (UserCmd*)(buffer + cursor);
            for (int i = count - 1; i >= 0; i--) {
                process_user_cmd(client_id, &cmds[i]);
            }
            local_state.players[client_id].active = 1; 
        }
    }
}

void server_broadcast() {
    char buffer[4096];
    int cursor = 0;
    
    NetHeader head; 
    head.type = PACKET_SNAPSHOT; 
    head.client_id = 0; 
    head.sequence = local_state.server_tick;
    head.timestamp = get_server_time();
    
    unsigned char count = 0;
    for(int i=0; i<MAX_CLIENTS; i++) if (local_state.players[i].active) count++;
    head.entity_count = count;
    
    memcpy(buffer + cursor, &head, sizeof(NetHeader)); cursor += sizeof(NetHeader);
    memcpy(buffer + cursor, &count, 1); cursor += 1;
    
    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (p->active) {
            NetPlayer np;
            np.id = (unsigned char)i;
            np.x = p->x; np.y = p->y; np.z = p->z;
            np.yaw = p->yaw; np.pitch = p->pitch;
            np.current_weapon = (unsigned char)p->current_weapon;
            np.state = (unsigned char)p->state;
            np.health = (unsigned char)p->health;
            np.shield = (unsigned char)p->shield;
            np.is_shooting = (unsigned char)p->is_shooting;
            np.crouching = (unsigned char)p->crouching;
            np.reward_feedback = p->accumulated_reward;
            np.ammo = (unsigned char)p->ammo[p->current_weapon];
            p->accumulated_reward = 0; 
            memcpy(buffer + cursor, &np, sizeof(NetPlayer)); cursor += sizeof(NetPlayer);
        }
    }
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.client_active[i]) {
            sendto(sock, buffer, cursor, 0, (struct sockaddr*)&local_state.clients[i], sizeof(struct sockaddr_in));
        }
    }
}

int main() {
    server_net_init();
    local_init_match(1, 0); 
    
    int running = 1;
    unsigned int tick = 0;
    
    while(running) {
        char buffer[1024];
        struct sockaddr_in sender; socklen_t slen = sizeof(sender);
        int len = recvfrom(sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        while (len > 0) {
            server_handle_packet(&sender, buffer, len);
            len = recvfrom(sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        }
        
        unsigned int now = get_server_time();
        int active_count = 0;
        
        for(int i=0; i<MAX_CLIENTS; i++) {
            PlayerState *p = &local_state.players[i];
            if (p->active) active_count++;
            
            if (p->state == STATE_DEAD) {
               if (local_state.game_mode != MODE_SURVIVAL && now > p->respawn_time) {
                   phys_respawn(p, now);
                   printf("Client %d Respawned\n", i);
               }
            }
            update_entity(p, 0.016f, NULL, now);
        }
        
        server_broadcast();
        
        if (tick % 300 == 0) {
            printf("[STATUS] Tick: %d | Active Entities: %d\n", tick, active_count);
        }
        
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
