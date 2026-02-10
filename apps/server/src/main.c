// apps/server/src/main.c
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
#include "server_mode.h"
#include "server_state.h"

int sock = -1;
struct sockaddr_in bind_addr;
unsigned int client_last_seq[MAX_CLIENTS];

typedef struct {
    int enabled;
    FILE *file;
    int target_id;
    float cam_x;
    float cam_y;
    float cam_z;
    float cam_yaw;
    float cam_pitch;
    float cam_zoom;
} RecorderState;

static RecorderState recorder = {0};

#define RECORDER_SHAKE_POS 0.08f
#define RECORDER_SHAKE_ANGLE 0.35f
#define RECORDER_SMOOTH_POS 0.08f
#define RECORDER_SMOOTH_ANGLE 0.18f
#define RECORDER_NORTH_X 0.0f
#define RECORDER_NORTH_Y 6.5f
#define RECORDER_NORTH_Z -32.0f

unsigned int get_server_time() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (unsigned int)(ts.tv_sec * 1000 + ts.tv_nsec / 1000000);
}

static int recorder_pick_target() {
    if (recorder.target_id >= 0 && recorder.target_id < MAX_CLIENTS) {
        if (local_state.players[recorder.target_id].active) {
            return recorder.target_id;
        }
    }
    int best_id = -1;
    int best_kills = -1;
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (!local_state.players[i].active) continue;
        if (local_state.players[i].kills > best_kills) {
            best_kills = local_state.players[i].kills;
            best_id = i;
        }
    }
    return best_id;
}

static float recorder_compute_zoom(float dist) {
    if (dist > 120.0f) return 3.0f;
    if (dist > 70.0f) return 2.4f;
    if (dist > 35.0f) return 1.8f;
    return 1.2f;
}

static void recorder_init_file(const char *path) {
    if (!recorder.enabled) return;
    recorder.file = fopen(path, "w");
    if (!recorder.file) {
        printf("[REC] Failed to open recording file: %s\n", path);
        recorder.enabled = 0;
        return;
    }
    recorder.cam_x = RECORDER_NORTH_X;
    recorder.cam_y = RECORDER_NORTH_Y;
    recorder.cam_z = RECORDER_NORTH_Z;
    recorder.cam_yaw = 0.0f;
    recorder.cam_pitch = 0.0f;
    recorder.cam_zoom = 1.2f;
    fprintf(recorder.file, "; SHANKPIT Recorder v1 (Lisp-ASM)\n");
    fprintf(recorder.file, "(begin-recording :dt-ms 16 :north-start '(%.2f %.2f %.2f))\n",
            RECORDER_NORTH_X, RECORDER_NORTH_Y, RECORDER_NORTH_Z);
}

static void recorder_update_camera() {
    int target_id = recorder_pick_target();
    if (target_id < 0) return;
    PlayerState *target = &local_state.players[target_id];

    float desired_x = RECORDER_NORTH_X + target->x * 0.15f;
    float desired_y = RECORDER_NORTH_Y + target->y * 0.1f;
    float desired_z = RECORDER_NORTH_Z + target->z * 0.15f;

    recorder.cam_x += (desired_x - recorder.cam_x) * RECORDER_SMOOTH_POS;
    recorder.cam_y += (desired_y - recorder.cam_y) * RECORDER_SMOOTH_POS;
    recorder.cam_z += (desired_z - recorder.cam_z) * RECORDER_SMOOTH_POS;

    float dx = target->x - recorder.cam_x;
    float dy = (target->y + 2.0f) - recorder.cam_y;
    float dz = target->z - recorder.cam_z;
    float dist = sqrtf(dx * dx + dz * dz);
    float target_yaw = atan2f(dx, dz) * (180.0f / 3.14159f);
    float target_pitch = atan2f(dy, dist) * (180.0f / 3.14159f);

    recorder.cam_yaw += (target_yaw - recorder.cam_yaw) * RECORDER_SMOOTH_ANGLE;
    recorder.cam_pitch += (target_pitch - recorder.cam_pitch) * RECORDER_SMOOTH_ANGLE;

    float zoom = recorder_compute_zoom(dist);
    if (target->current_weapon == WPN_SNIPER) {
        zoom += 0.4f;
    }
    float shake_pos = RECORDER_SHAKE_POS / zoom;
    float shake_ang = RECORDER_SHAKE_ANGLE / zoom;

    recorder.cam_x += phys_rand_f() * shake_pos;
    recorder.cam_y += phys_rand_f() * shake_pos;
    recorder.cam_z += phys_rand_f() * shake_pos;
    recorder.cam_yaw += phys_rand_f() * shake_ang;
    recorder.cam_pitch += phys_rand_f() * shake_ang;
    recorder.cam_zoom = zoom;
}

static void recorder_write_frame(unsigned int tick, unsigned int now_ms) {
    if (!recorder.enabled || !recorder.file) return;
    recorder_update_camera();

    fprintf(recorder.file, "(frame :tick %u :time-ms %u\n", tick, now_ms);
    fprintf(recorder.file, "  (camera :x %.3f :y %.3f :z %.3f :yaw %.2f :pitch %.2f :zoom %.2f :mode \"handicam\")\n",
            recorder.cam_x, recorder.cam_y, recorder.cam_z,
            recorder.cam_yaw, recorder.cam_pitch, recorder.cam_zoom);

    for (int i = 0; i < MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active) continue;
        fprintf(recorder.file,
                "  (actor :id %d :x %.3f :y %.3f :z %.3f :vx %.3f :vy %.3f :vz %.3f :yaw %.2f :pitch %.2f :weapon %d :state %d)\n",
                i, p->x, p->y, p->z, p->vx, p->vy, p->vz, p->yaw, p->pitch, p->current_weapon, p->state);
    }
    fprintf(recorder.file, ")\n");
    if (tick % 60 == 0) {
        fflush(recorder.file);
    }
}

int parse_server_mode(int argc, char **argv) {
    int mode = MODE_DEATHMATCH;
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--tdm") == 0) {
            mode = MODE_TDM;
        } else if (strcmp(argv[i], "--deathmatch") == 0) {
            mode = MODE_DEATHMATCH;
        }
    }
    return mode;
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
    } else {
        printf("SERVER LISTENING ON PORT 6969\nWaiting...\n");
    }
}

void process_user_cmd(int client_id, UserCmd *cmd) {
    if (cmd->sequence <= client_last_seq[client_id]) return;
    printf("[CMD] client=%d seq=%u buttons=%u\n", client_id, cmd->sequence, cmd->buttons);
    PlayerState *p = &local_state.players[client_id];
    p->yaw = norm_yaw_deg(cmd->yaw);
    p->pitch = clamp_pitch_deg(cmd->pitch);
    p->in_fwd = cmd->fwd;
    p->in_strafe = cmd->str;
    p->in_jump = (cmd->buttons & BTN_JUMP);
    p->in_shoot = (cmd->buttons & BTN_ATTACK);
    p->crouching = (cmd->buttons & BTN_CROUCH);
    p->in_reload = (cmd->buttons & BTN_RELOAD);
    p->in_use = (cmd->buttons & BTN_USE);
    p->in_ability = (cmd->buttons & BTN_ABILITY_1);
    if (cmd->weapon_idx >= 0 && cmd->weapon_idx < MAX_WEAPONS) p->current_weapon = cmd->weapon_idx;
    client_last_seq[client_id] = cmd->sequence;
}

void server_handle_packet(struct sockaddr_in *sender, char *buffer, int size) {
    if (size < (int)sizeof(NetHeader)) return;
    NetHeader *head = (NetHeader*)buffer;
    int client_id = -1;

    // --- FIND EXISTING CLIENT BY SENDER (DO NOT RESET PLAYER STATE) ---
    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.client_meta[i].active &&
            memcmp(&local_state.clients[i].sin_addr, &sender->sin_addr, sizeof(struct in_addr)) == 0 &&
            local_state.clients[i].sin_port == sender->sin_port) {

            client_id = i;
            // Touch only liveness; never memset player here.
            local_state.client_meta[i].last_heard_ms = get_server_time();
            break;
        }
    }

    // --- NEW CONNECTION PATH ---
    if (client_id == -1) {
        if (head->type == PACKET_CONNECT) {
            char *ip_str = inet_ntoa(sender->sin_addr);
            for(int i=1; i<MAX_CLIENTS; i++) {
                if (!local_state.client_meta[i].active) {
                    client_id = i;

                    // Allocate + initialize player ONCE here.
                    memset(&local_state.players[i], 0, sizeof(PlayerState));
                    local_state.players[i].id = i;
                    local_state.players[i].scene_id = SCENE_GARAGE_OSAKA;
                    local_state.players[i].active = 1;

                    local_state.client_meta[i].active = 1;
                    local_state.client_meta[i].last_heard_ms = get_server_time();
                    local_state.clients[i] = *sender;

                    phys_respawn(&local_state.players[i], get_server_time());

                    printf("CLIENT %d CONNECTED (%s)\n", client_id, ip_str);

                    NetHeader h;
                    h.type = PACKET_WELCOME;
                    h.client_id = (unsigned char)client_id;
                    h.sequence = 0;
                    h.timestamp = get_server_time();
                    h.entity_count = 0;
                    h.scene_id = (unsigned char)local_state.players[i].scene_id;

                    sendto(sock, (char*)&h, sizeof(NetHeader), 0,
                           (struct sockaddr*)sender, sizeof(struct sockaddr_in));
                    break;
                }
            }
        }
    }

    // --- USER COMMANDS ---
    if (client_id != -1 && head->type == PACKET_USERCMD) {
        int cursor = (int)sizeof(NetHeader);
        if (size < cursor + 1) return;

        unsigned char count = *(unsigned char*)(buffer + cursor); cursor += 1;

        if (size >= cursor + (int)(count * sizeof(UserCmd))) {
            UserCmd *cmds = (UserCmd*)(buffer + cursor);

            // process oldest->newest to preserve chronological intent
            for (int i = (int)count - 1; i >= 0; i--) {
                process_user_cmd(client_id, &cmds[i]);
            }

            local_state.client_meta[client_id].last_heard_ms = get_server_time();
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
    head.scene_id = 0;

    unsigned char count = 0;
    for(int i=0; i<MAX_CLIENTS; i++) if (local_state.players[i].active) count++;
    head.entity_count = count;

    memcpy(buffer + cursor, &head, sizeof(NetHeader)); cursor += (int)sizeof(NetHeader);
    memcpy(buffer + cursor, &count, 1); cursor += 1;

    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (p->active) {
            NetPlayer np;
            np.id = (unsigned char)i;
            np.scene_id = (unsigned char)p->scene_id;
            np.x = p->x; np.y = p->y; np.z = p->z;
            np.yaw = norm_yaw_deg(p->yaw); np.pitch = clamp_pitch_deg(p->pitch);
            np.current_weapon = (unsigned char)p->current_weapon;
            np.state = (unsigned char)p->state;
            np.health = (unsigned char)p->health;
            np.shield = (unsigned char)p->shield;
            np.is_shooting = (unsigned char)p->is_shooting;
            np.crouching = (unsigned char)p->crouching;
            np.reward_feedback = p->accumulated_reward;
            np.ammo = (unsigned char)p->ammo[p->current_weapon];
            np.in_vehicle = (unsigned char)p->in_vehicle;
            np.hit_feedback = (unsigned char)p->hit_feedback;
            np.storm_charges = (unsigned char)p->storm_charges;

            p->accumulated_reward = 0;
            memcpy(buffer + cursor, &np, sizeof(NetPlayer)); cursor += (int)sizeof(NetPlayer);
        }
    }

    for(int i=1; i<MAX_CLIENTS; i++) {
        if (local_state.client_meta[i].active) {
            sendto(sock, buffer, cursor, 0,
                   (struct sockaddr*)&local_state.clients[i],
                   sizeof(struct sockaddr_in));
        }
    }
}

int main(int argc, char *argv[]) {
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--record") == 0) {
            recorder.enabled = 1;
        } else if (strcmp(argv[i], "--record-target") == 0 && i + 1 < argc) {
            recorder.target_id = atoi(argv[i + 1]);
            i++;
        } else if (strcmp(argv[i], "--record-file") == 0 && i + 1 < argc) {
            recorder.enabled = 1;
            recorder_init_file(argv[i + 1]);
            i++;
        }
    }

    if (recorder.enabled && !recorder.file) {
        recorder_init_file("shankpit_recording.lispasm");
    }

    server_net_init();
    int mode = parse_server_mode(argc, argv);
    local_init_match(1, mode);
    local_state.players[0].active = 0;
    local_state.players[0].health = 0;
    local_state.players[0].state = STATE_DEAD;
    printf("SERVER MODE: %s\n", mode == MODE_TDM ? "TEAM DEATHMATCH" : "DEATHMATCH");

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
        // TIMEOUT_SWEEP
        for (int i = 1; i < MAX_CLIENTS; i++) {
            if (local_state.client_meta[i].active &&
                now - local_state.client_meta[i].last_heard_ms > 5000) {
                server_disconnect(i, client_last_seq);
            }
        }

        int active_count = 0;

        for(int i=0; i<MAX_CLIENTS; i++) {
            PlayerState *p = &local_state.players[i];
            if (p->active) active_count++;

            if (p->state == STATE_DEAD) {
                if (local_state.game_mode != MODE_SURVIVAL && now > p->respawn_time) {
                    phys_respawn(p, now);
                }
            }

            if (p->active && p->state != STATE_DEAD) {
                phys_set_scene(p->scene_id);
                int use_pressed = p->in_use && !p->use_was_down;
                if (use_pressed && now >= p->portal_cooldown_until_ms &&
                    scene_portal_active(p->scene_id) && scene_portal_triggered(p)) {
                    int dest_scene = -1;
                    float sx = 0.0f, sy = 0.0f, sz = 0.0f;
                    if (portal_resolve_destination(p->scene_id, PORTAL_ID_GARAGE_EXIT, p->id,
                                                   &dest_scene, &sx, &sy, &sz)) {
                        int from_scene = p->scene_id;
                        p->scene_id = dest_scene;
                        phys_set_scene(p->scene_id);
                        p->x = sx; p->y = sy; p->z = sz;
                        p->vx = 0.0f; p->vy = 0.0f; p->vz = 0.0f;
                        p->in_vehicle = 0;
                        p->portal_cooldown_until_ms = now + 1000;
                        p->in_use = 0;
                        printf("PORTAL_TRAVEL client=%d from=%d to=%d\n", i, from_scene, dest_scene);
                    }
                } else if (use_pressed && p->vehicle_cooldown == 0) {
                    int in_garage = p->scene_id == SCENE_GARAGE_OSAKA;
                    if (in_garage && scene_near_vehicle_pad(p->scene_id, p->x, p->z, 6.0f, NULL)) {
                        p->in_vehicle = !p->in_vehicle;
                        p->vehicle_cooldown = 30;
                        printf("Client %d Toggle Vehicle: %d\n", i, p->in_vehicle);
                    } else if (!in_garage) {
                        p->in_vehicle = !p->in_vehicle;
                        p->vehicle_cooldown = 30;
                        printf("Client %d Toggle Vehicle: %d\n", i, p->in_vehicle);
                    }
                }
                p->use_was_down = p->in_use;
                if (p->vehicle_cooldown > 0) p->vehicle_cooldown--;

                float rad = p->yaw * 3.14159f / 180.0f;
                float wish_x = 0, wish_z = 0;
                float max_spd = MAX_SPEED;
                float acc = ACCEL;

                if (p->in_vehicle) {
                    wish_x = sinf(rad) * p->in_fwd;
                    wish_z = cosf(rad) * p->in_fwd;
                    max_spd = BUGGY_MAX_SPEED;
                    acc = BUGGY_ACCEL;
                } else {
                    wish_x = sinf(rad) * p->in_fwd + cosf(rad) * p->in_strafe;
                    wish_z = cosf(rad) * p->in_fwd - sinf(rad) * p->in_strafe;
                }

                float wish_speed = sqrtf(wish_x*wish_x + wish_z*wish_z);
                if (wish_speed > 1.0f) {
                    wish_speed = 1.0f;
                    wish_x/=wish_speed; wish_z/=wish_speed;
                }
                wish_speed *= max_spd;

                accelerate(p, wish_x, wish_z, wish_speed, acc);

                float g = p->in_vehicle ? BUGGY_GRAVITY : (p->in_jump ? GRAVITY_FLOAT : GRAVITY_DROP);
                p->vy -= g;
                if (p->in_jump && p->on_ground) {
                    p->y += 0.1f;
                    p->vy += JUMP_FORCE;
                }
            }

            update_entity(p, 0.016f, NULL, now);
        }

        update_projectiles(now);
        recorder_write_frame(tick, now);
        server_broadcast();

        if (tick % 300 == 0) printf("[STATUS] Tick: %u | Clients: %d\n", tick, active_count);

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
