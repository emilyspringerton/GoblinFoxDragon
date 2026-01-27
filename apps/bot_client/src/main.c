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
    #include <netdb.h>
#endif

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"

// --- STATE ---
int sock = -1;
struct sockaddr_in server_addr;
PlayerState my_state; // What the server says I am
PlayerState world_state[MAX_CLIENTS]; // What I see
BotGenome brain;
unsigned int cmd_seq = 0;
char brain_filename[64];

// --- UTILS ---
float rand_f() { return ((float)(rand()%1000)/1000.0f); }
float rand_w() { return ((float)(rand()%2000)/1000.0f) - 1.0f; }

static inline float angle_diff(float a, float b) {
    float d = a - b;
    while (d < -180) d += 360; while (d > 180) d -= 360;
    return d;
}

// --- BRAIN IO ---
void load_brain(const char* filename) {
    FILE *f = fopen(filename, "rb");
    if (f) {
        fread(&brain, sizeof(BotGenome), 1, f);
        fclose(f);
        printf("🧠 LOADED BRAIN v%d from %s\n", brain.version, filename);
    } else {
        printf("🧠 NEW BRAIN (Randomized)\n");
        brain.version = 1;
        brain.w_aggro = 0.5f + rand_w() * 0.5f;
        brain.w_strafe = rand_w();
        brain.w_jump = 0.05f + rand_f() * 0.1f;
        brain.w_slide = 0.01f + rand_f() * 0.05f;
        brain.w_turret = 10.0f + rand_f() * 10.0f;
        brain.w_repel = 1.0f + rand_f();
    }
    strcpy(brain_filename, filename);
}

void save_brain() {
    brain.version++;
    FILE *f = fopen(brain_filename, "wb");
    if (f) {
        fwrite(&brain, sizeof(BotGenome), 1, f);
        fclose(f);
        printf("💾 SAVED BRAIN v%d to %s\n", brain.version, brain_filename);
    }
}

// --- NETWORKING ---
void net_init(const char* host, int port) {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    #ifdef _WIN32
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif

    struct hostent *he = gethostbyname(host);
    if (!he) { printf("❌ Host lookup failed\n"); exit(1); }
    
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(port);
    memcpy(&server_addr.sin_addr, he->h_addr_list[0], he->h_length);
    
    // Connect Handshake
    char buf[32]; NetHeader *h = (NetHeader*)buf; h->type = PACKET_CONNECT;
    sendto(sock, buf, sizeof(NetHeader), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
}

// --- THINK ---
UserCmd bot_think() {
    UserCmd cmd; memset(&cmd, 0, sizeof(UserCmd));
    cmd.sequence = ++cmd_seq;
    
    if (my_state.state == STATE_DEAD) return cmd; // Do nothing if dead

    // Find Target
    int target_idx = -1;
    float min_dist = 9999.0f;
    
    for(int i=0; i<MAX_CLIENTS; i++) {
        if (!world_state[i].active || world_state[i].state == STATE_DEAD) continue;
        if (i == my_state.id) continue; // Don't target self (Need ID from server)
        
        float dx = world_state[i].x - my_state.x;
        float dz = world_state[i].z - my_state.z;
        float dist = sqrtf(dx*dx + dz*dz);
        
        // Skip if too far
        if (dist < 200.0f && dist < min_dist) {
            min_dist = dist; target_idx = i;
        }
    }

    if (target_idx != -1) {
        float dx = world_state[target_idx].x - my_state.x;
        float dz = world_state[target_idx].z - my_state.z;
        float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
        
        // Smooth Aim
        float diff = angle_diff(target_yaw, my_state.yaw);
        if (diff > brain.w_turret) diff = brain.w_turret;
        if (diff < -brain.w_turret) diff = -brain.w_turret;
        cmd.yaw = my_state.yaw + diff;
        
        cmd.buttons |= BTN_ATTACK;
        
        // Move
        if (min_dist > 15.0f) cmd.fwd = brain.w_aggro;
        else if (min_dist < 5.0f) cmd.fwd = -brain.w_aggro;
        
        // Strafe
        cmd.yaw += brain.w_strafe * 5.0f; 
        
        // Jump/Slide Random
        if ((rand()%1000) < (brain.w_jump * 1000.0f)) cmd.buttons |= BTN_JUMP;
        if ((rand()%1000) < (brain.w_slide * 1000.0f)) cmd.buttons |= BTN_CROUCH;
        
    } else {
        // Patrol
        cmd.yaw = my_state.yaw + 2.0f;
        cmd.fwd = 0.5f;
    }
    
    return cmd;
}

void process_packet(char *buf, int len) {
    NetHeader *h = (NetHeader*)buf;
    if (h->type == PACKET_SNAPSHOT) {
        int cursor = sizeof(NetHeader);
        unsigned char count = *(unsigned char*)(buf + cursor); cursor++;
        
        for(int i=0; i<count; i++) {
            NetPlayer *np = (NetPlayer*)(buf + cursor);
            cursor += sizeof(NetPlayer);
            
            // Update World View
            if (np->id > 0 && np->id < MAX_CLIENTS) {
                world_state[np->id].active = 1;
                world_state[np->id].x = np->x;
                world_state[np->id].z = np->z;
                world_state[np->id].state = np->state;
                
                // Heuristic: If this snapshot feels like 'ME', adopt it
                // (Real netcode has ID negotiation, hack for now: closest to 0,0?)
                // Actually, server needs to tell us our ID. 
                // Assumption: Single bot test -> I am the only connection usually.
                // For now, simple blind update: Update self logic is internal.
                
                // Hack: If accumulated reward > 0, we learned something!
                if (np->reward_feedback > 0) {
                    printf("🎯 REWARD! +%.1f\n", np->reward_feedback);
                    my_state.accumulated_reward += np->reward_feedback;
                    
                    // REINFORCEMENT LEARNING STEP (Simple Hill Climbing)
                    // If reward is good, reinforce current behavior weights?
                    // For now, we just track score to save on exit.
                }
            }
        }
    }
}

int main(int argc, char* argv[]) {
    srand(time(NULL));
    char *host = "127.0.0.1";
    char *bfile = "bot_v1.bin";
    
    for(int i=1; i<argc; i++) {
        if(strcmp(argv[i], "--host")==0) host = argv[++i];
        if(strcmp(argv[i], "--brain")==0) bfile = argv[++i];
    }
    
    load_brain(bfile);
    net_init(host, 6969);
    
    printf("🤖 SHANKBOT ACTIVE. Target: %s. Brain: %s\n", host, bfile);
    
    int running = 1;
    while(running) {
        // 1. RECV
        char buf[4096];
        struct sockaddr_in sender; socklen_t slen=sizeof(sender);
        int len = recvfrom(sock, buf, 4096, 0, (struct sockaddr*)&sender, &slen);
        while(len > 0) {
            process_packet(buf, len);
            len = recvfrom(sock, buf, 4096, 0, (struct sockaddr*)&sender, &slen);
        }
        
        // 2. THINK & SEND
        UserCmd cmd = bot_think();
        
        // Pack & Send
        char pbuf[256];
        NetHeader h; h.type = PACKET_USERCMD; h.entity_count=1;
        memcpy(pbuf, &h, sizeof(NetHeader));
        unsigned char c=1; memcpy(pbuf+sizeof(NetHeader), &c, 1);
        memcpy(pbuf+sizeof(NetHeader)+1, &cmd, sizeof(UserCmd));
        sendto(sock, pbuf, sizeof(NetHeader)+1+sizeof(UserCmd), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
        
        // 3. SLEEP (60Hz)
        #ifdef _WIN32
        Sleep(16);
        #else
        usleep(16000);
        #endif
        
        // AUTO SAVE if doing well
        if (my_state.accumulated_reward > 5000.0f) {
            save_brain();
            my_state.accumulated_reward = 0; // Reset threshold
        }
    }
    return 0;
}
