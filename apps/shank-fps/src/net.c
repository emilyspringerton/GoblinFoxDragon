#define _WIN32_WINNT 0x0600 // Required for inet_pton
#include "net.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #pragma comment(lib, "ws2_32.lib")
#else
    #include <sys/socket.h>
    #include <netinet/in.h>
    #include <arpa/inet.h>
    #include <fcntl.h>
    #include <unistd.h>
#endif

int sockfd = -1;
struct sockaddr_in serv_addr;

void net_init(GameContext *game, const char *ip) {
#ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
#endif
    sockfd = socket(AF_INET, SOCK_DGRAM, 0);
#ifdef _WIN32
    u_long mode = 1; ioctlsocket(sockfd, FIONBIO, &mode);
#else
    int flags = fcntl(sockfd, F_GETFL, 0); fcntl(sockfd, F_SETFL, flags | O_NONBLOCK);
#endif
    memset(&serv_addr, 0, sizeof(serv_addr));
    serv_addr.sin_family = AF_INET; serv_addr.sin_port = htons(PORT);
    inet_pton(AF_INET, ip, &serv_addr.sin_addr);
    printf("NET: Socket created. Target: %s:%d\n", ip, PORT);
}

void net_send_input(GameContext *game) {
    if (sockfd < 0) return;
    ClientPacket pkt = {0};
    Player *p = &game->local_player;
    const Uint8 *state = SDL_GetKeyboardState(NULL);
    if (state[SDL_SCANCODE_W]) pkt.fwd += 1.0f;
    if (state[SDL_SCANCODE_S]) pkt.fwd -= 1.0f;
    if (state[SDL_SCANCODE_D]) pkt.strafe += 1.0f;
    if (state[SDL_SCANCODE_A]) pkt.strafe -= 1.0f;
    if (state[SDL_SCANCODE_G]) pkt.throw_grenade = 1;
    if (state[SDL_SCANCODE_E]) pkt.interact = 1;
    if (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT)) pkt.shoot = 1;
    pkt.yaw = p->yaw; pkt.pitch = p->pitch; pkt.weapon_req = p->current_weapon; 
    pkt.show_scores = state[SDL_SCANCODE_TAB];
    // Cast to const char*
    sendto(sockfd, (const char*)&pkt, sizeof(pkt), 0, (const struct sockaddr*)&serv_addr, sizeof(serv_addr));
}

void net_receive_state(GameContext *game) {
    if (sockfd < 0) return;
    ServerState state;
    struct sockaddr_in from; socklen_t len = sizeof(from);
    int n = recvfrom(sockfd, (char*)&state, sizeof(state), 0, (struct sockaddr*)&from, &len);
    if (n > 0) {
        int my_id = 0; 
        PlayerState *net_me = &state.players[my_id];
        game->local_player.pos = net_me->pos;
        game->local_player.vel = net_me->vel;
        game->local_player.health = net_me->health;
        game->local_player.ammo[net_me->current_weapon] = net_me->ammo[net_me->current_weapon];
        game->last_server_state = state; 
    }
}
