#include "ui_bridge.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #pragma comment(lib, "ws2_32.lib")
#else
    #include <sys/socket.h>
    #include <netinet/in.h>
    #include <arpa/inet.h>
    #include <netdb.h>
    #include <unistd.h>
#endif

#define UI_BRIDGE_BUF 16384

static char ui_bridge_host[128] = "127.0.0.1";
static int ui_bridge_port = 17777;

static int ui_bridge_socket_init = 0;

static void ui_bridge_net_init() {
    if (ui_bridge_socket_init) return;
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2, 2), &wsa);
    #endif
    ui_bridge_socket_init = 1;
}

int ui_bridge_init(const char *host, int port) {
    ui_bridge_net_init();
    if (host && host[0]) {
        strncpy(ui_bridge_host, host, sizeof(ui_bridge_host) - 1);
        ui_bridge_host[sizeof(ui_bridge_host) - 1] = '\0';
    }
    if (port > 0) ui_bridge_port = port;
    return 1;
}

static int ui_bridge_connect() {
    ui_bridge_net_init();
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) return -1;

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(ui_bridge_port);

    struct hostent *he = gethostbyname(ui_bridge_host);
    if (he) {
        memcpy(&addr.sin_addr, he->h_addr_list[0], he->h_length);
    } else {
        addr.sin_addr.s_addr = inet_addr(ui_bridge_host);
    }

    if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        #ifdef _WIN32
        closesocket(sock);
        #else
        close(sock);
        #endif
        return -1;
    }
    return sock;
}

static int ui_bridge_send_request(const char *request, char *out_buf, size_t out_size) {
    int sock = ui_bridge_connect();
    if (sock < 0) return 0;

    size_t req_len = strlen(request);
    if (send(sock, request, (int)req_len, 0) < 0) {
        #ifdef _WIN32
        closesocket(sock);
        #else
        close(sock);
        #endif
        return 0;
    }

    int total = 0;
    int n = 0;
    while ((n = recv(sock, out_buf + total, (int)(out_size - 1 - total), 0)) > 0) {
        total += n;
        if (total >= (int)out_size - 1) break;
    }
    out_buf[total] = '\0';

    #ifdef _WIN32
    closesocket(sock);
    #else
    close(sock);
    #endif

    return total > 0;
}

static const char *ui_bridge_body_start(const char *resp) {
    const char *body = strstr(resp, "\r\n\r\n");
    if (!body) return NULL;
    return body + 4;
}

static int ui_bridge_extract_string(const char *src, const char *key, char *out, size_t out_size) {
    char needle[64];
    snprintf(needle, sizeof(needle), "\"%s\"", key);
    const char *pos = strstr(src, needle);
    if (!pos) return 0;
    pos = strstr(pos, ":");
    if (!pos) return 0;
    pos++;
    while (*pos == ' ' || *pos == '\t' || *pos == '\n') pos++;
    if (*pos != '"') return 0;
    pos++;
    const char *end = strchr(pos, '"');
    if (!end) return 0;
    size_t len = (size_t)(end - pos);
    if (len >= out_size) len = out_size - 1;
    memcpy(out, pos, len);
    out[len] = '\0';
    return 1;
}

static int ui_bridge_extract_bool(const char *src, const char *key, int *out_val) {
    char needle[64];
    snprintf(needle, sizeof(needle), "\"%s\"", key);
    const char *pos = strstr(src, needle);
    if (!pos) return 0;
    pos = strstr(pos, ":");
    if (!pos) return 0;
    pos++;
    while (*pos == ' ' || *pos == '\t' || *pos == '\n') pos++;
    if (strncmp(pos, "true", 4) == 0) {
        *out_val = 1;
        return 1;
    }
    if (strncmp(pos, "false", 5) == 0) {
        *out_val = 0;
        return 1;
    }
    return 0;
}

static int ui_bridge_parse_menu(const char *body, UiState *state) {
    const char *entries = strstr(body, "\"entries\"");
    if (!entries) return 0;
    const char *array_start = strchr(entries, '[');
    if (!array_start) return 0;
    const char *cursor = array_start + 1;
    int count = 0;

    while (*cursor && *cursor != ']' && count < UI_BRIDGE_MAX_ENTRIES) {
        const char *obj_start = strchr(cursor, '{');
        if (!obj_start) break;
        const char *obj_end = strchr(obj_start, '}');
        if (!obj_end) break;

        UiMenuEntry *entry = &state->entries[count];
        memset(entry, 0, sizeof(UiMenuEntry));
        ui_bridge_extract_string(obj_start, "id", entry->id, sizeof(entry->id));
        ui_bridge_extract_string(obj_start, "label", entry->label, sizeof(entry->label));
        ui_bridge_extract_string(obj_start, "kind", entry->kind, sizeof(entry->kind));
        entry->enabled = 1;
        ui_bridge_extract_bool(obj_start, "enabled", &entry->enabled);

        if (entry->id[0] != '\0' && entry->label[0] != '\0') {
            count++;
        }
        cursor = obj_end + 1;
    }

    state->entry_count = count;
    return count > 0;
}

static void ui_bridge_state_reset(UiState *state) {
    memset(state, 0, sizeof(UiState));
    strncpy(state->protocol_version, "v0", sizeof(state->protocol_version) - 1);
}

int ui_bridge_fetch_state(UiState *out_state) {
    if (!out_state) return 0;
    ui_bridge_state_reset(out_state);

    char request[256];
    snprintf(request, sizeof(request),
             "GET /api/v0/ui/state HTTP/1.1\r\nHost: %s:%d\r\nConnection: close\r\n\r\n",
             ui_bridge_host, ui_bridge_port);

    char response[UI_BRIDGE_BUF];
    if (!ui_bridge_send_request(request, response, sizeof(response))) return 0;

    const char *body = ui_bridge_body_start(response);
    if (!body) return 0;

    ui_bridge_extract_string(body, "protocolVersion", out_state->protocol_version, sizeof(out_state->protocol_version));
    ui_bridge_extract_bool(body, "isOpen", &out_state->menu_open);
    ui_bridge_extract_string(body, "activeSceneId", out_state->active_scene_id, sizeof(out_state->active_scene_id));
    ui_bridge_extract_string(body, "activeModeId", out_state->active_mode_id, sizeof(out_state->active_mode_id));

    return ui_bridge_parse_menu(body, out_state);
}

int ui_bridge_send_intent_activate(const char *entry_id, UiState *out_state) {
    if (!entry_id || !entry_id[0]) return 0;
    long long ts = (long long)time(NULL) * 1000;
    char body[512];
    snprintf(body, sizeof(body),
             "{\"protocolVersion\":\"v0\",\"intentId\":\"intent-%lld\",\"type\":\"intent.activate\",\"ts\":%lld,\"payload\":{\"entryId\":\"%s\"}}",
             ts, ts, entry_id);

    char request[1024];
    snprintf(request, sizeof(request),
             "POST /api/v0/ui/intent HTTP/1.1\r\nHost: %s:%d\r\nContent-Type: application/json\r\nContent-Length: %zu\r\nConnection: close\r\n\r\n%s",
             ui_bridge_host, ui_bridge_port, strlen(body), body);

    char response[UI_BRIDGE_BUF];
    if (!ui_bridge_send_request(request, response, sizeof(response))) return 0;

    if (out_state) {
        return ui_bridge_fetch_state(out_state);
    }
    return 1;
}
