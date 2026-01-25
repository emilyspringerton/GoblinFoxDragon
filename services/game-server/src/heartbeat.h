#ifndef HEARTBEAT_H
#define HEARTBEAT_H

#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <time.h>

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
#else
    #include <sys/socket.h>
    #include <netinet/in.h>
    #include <arpa/inet.h>
#endif

// Simple TCP sender
void send_heartbeat_to_master(const char* master_ip, int master_port, int my_port, const char* name, int players, int max) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) return;

    struct sockaddr_in serv_addr;
    memset(&serv_addr, 0, sizeof(serv_addr));
    serv_addr.sin_family = AF_INET;
    serv_addr.sin_port = htons(master_port);
    inet_pton(AF_INET, master_ip, &serv_addr.sin_addr);

    if (connect(sock, (struct sockaddr*)&serv_addr, sizeof(serv_addr)) < 0) {
        // Silently fail if master is down
        #ifdef _WIN32
        closesocket(sock);
        #else
        close(sock);
        #endif
        return;
    }

    char msg[256];
    // Protocol: HEARTBEAT|Port|Name|Players|Max
    sprintf(msg, "HEARTBEAT|%d|%s|%d|%d\n", my_port, name, players, max);
    send(sock, msg, strlen(msg), 0);

    #ifdef _WIN32
    closesocket(sock);
    #else
    close(sock);
    #endif
}

#endif
