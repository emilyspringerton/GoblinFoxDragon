#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
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
    #include <netdb.h>
#endif

#include <SDL2/SDL.h>
#include <SDL2/SDL_opengl.h>
#include <GL/glu.h>

#include "../../../packages/common/protocol.h"
#include "../../../packages/common/physics.h"
#include "../../../packages/simulation/local_game.h"

// --- GAME STATE ---
#define STATE_LISTEN_SERVER 99 
int app_state = STATE_LISTEN_SERVER; // AUTO-START
int wpn_req = 1; // Start with Magnum

float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;

// --- NETWORKING ---
int sock = -1;
struct sockaddr_in server_addr;
int sv_sock = -1;
unsigned int sv_client_last_seq[MAX_CLIENTS];

// --- SERVER LOGIC (Local) ---
void sv_process_cmd(int client_id, UserCmd *cmd) {
    if (cmd->sequence <= sv_client_last_seq[client_id]) return;
    PlayerState *p = &local_state.players[client_id];
    
    // 1. Inputs
    if (!isnan(cmd->yaw)) p->yaw = cmd->yaw;
    if (!isnan(cmd->pitch)) p->pitch = cmd->pitch;
    p->current_weapon = cmd->weapon_idx;
    
    // 2. Physics (Quake-style)
    float rad = p->yaw * 0.0174533f;
    float wish_x = sinf(rad) * cmd->fwd + cosf(rad) * cmd->str;
    float wish_z = cosf(rad) * cmd->fwd - sinf(rad) * cmd->str;
    
    // Normalize wish direction
    float wish_speed = sqrtf(wish_x*wish_x + wish_z*wish_z);
    if (wish_speed > 1.0f) {
        wish_speed = 1.0f;
        wish_x /= wish_speed; wish_z /= wish_speed;
    }
    wish_speed *= MAX_SPEED;
    
    accelerate(p, wish_x, wish_z, wish_speed, ACCEL);
    
    // Jump
    if ((cmd->buttons & BTN_JUMP) && p->on_ground) {
        p->y += 0.1f; p->vy += JUMP_FORCE;
    }
    
    // Crouch
    p->crouching = (cmd->buttons & BTN_CROUCH);
    
    // Update Entity (Gravity, Collision, Move)
    update_entity(p, 0.016f, NULL, cmd->timestamp);
    
    sv_client_last_seq[client_id] = cmd->sequence;
}

void sv_tick() {
    // 1. Network Poll
    char buffer[1024];
    struct sockaddr_in sender;
    socklen_t slen = sizeof(sender);
    if (sv_sock != -1) {
        int len = recvfrom(sv_sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        while (len > 0) {
            if (len >= sizeof(NetHeader)) {
                NetHeader *head = (NetHeader*)buffer;
                if (head->type == PACKET_USERCMD) {
                    int cursor = sizeof(NetHeader);
                    unsigned char count = *(unsigned char*)(buffer + cursor); cursor++;
                    UserCmd *cmds = (UserCmd*)(buffer + cursor);
                    // Assume Client 0 for now
                    for (int i = count - 1; i >= 0; i--) sv_process_cmd(0, &cmds[i]);
                }
            }
            len = recvfrom(sv_sock, buffer, 1024, 0, (struct sockaddr*)&sender, &slen);
        }
    }
    
    // 2. Bots
    for(int i=1; i<MAX_CLIENTS; i++) {
        PlayerState *p = &local_state.players[i];
        if (!p->active) continue;
        float bfwd=0, byaw=p->yaw; int bbtns=0;
        bot_think(i, local_state.players, &bfwd, &byaw, &bbtns);
        p->yaw = byaw;
        
        // Bot Physics
        float brad = byaw * 0.0174533f;
        float bx = sinf(brad) * bfwd;
        float bz = cosf(brad) * bfwd;
        accelerate(p, bx, bz, MAX_SPEED, ACCEL);
        
        p->in_shoot = (bbtns & BTN_ATTACK);
        update_entity(p, 0.016f, NULL, 0);
    }
}

// --- CLIENT LOGIC ---
UserCmd client_create_cmd(float fwd, float str, float yaw, float pitch, int shoot, int jump, int crouch, int reload, int wpn_idx) {
    UserCmd cmd;
    memset(&cmd, 0, sizeof(UserCmd));
    static int seq = 0;
    cmd.sequence = ++seq;
    cmd.timestamp = SDL_GetTicks();
    cmd.yaw = yaw; cmd.pitch = pitch;
    cmd.fwd = fwd; cmd.str = str;
    if(shoot) cmd.buttons |= BTN_ATTACK;
    if(jump) cmd.buttons |= BTN_JUMP;
    if(crouch) cmd.buttons |= BTN_CROUCH;
    if(reload) cmd.buttons |= BTN_RELOAD;
    cmd.weapon_idx = wpn_idx;
    return cmd;
}

void net_send_cmd(UserCmd cmd) {
    if (sock == -1) return;
    char packet_data[1024];
    int cursor = 0;
    NetHeader head; head.type = PACKET_USERCMD; head.client_id = 1; 
    memcpy(packet_data + cursor, &head, sizeof(NetHeader)); cursor += sizeof(NetHeader);
    unsigned char count = 1;
    memcpy(packet_data + cursor, &count, 1); cursor += 1;
    memcpy(packet_data + cursor, &cmd, sizeof(UserCmd)); cursor += sizeof(UserCmd);
    sendto(sock, packet_data, cursor, 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
}

void net_init() {
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    #ifdef _WIN32
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    #else
    int flags = fcntl(sock, F_GETFL, 0); fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    #endif
    // Loopback for Listen Server
    server_addr.sin_family = AF_INET; server_addr.sin_port = htons(6969);
    server_addr.sin_addr.s_addr = inet_addr("127.0.0.1");
}

// --- RENDER ---
void draw_grid() {
    glBegin(GL_LINES); glColor3f(0.0f, 1.0f, 1.0f);
    for(int i=-100; i<=900; i+=20) {
        glVertex3f(i, 0, -100); glVertex3f(i, 0, 100);
        glVertex3f(-100, 0, i); glVertex3f(100, 0, i);
    }
    glEnd();
}

void draw_map() {
    // Using map_geo from physics.h
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        glPushMatrix();
        glTranslatef(b.x, b.y, b.z);
        glScalef(b.w, b.h, b.d);
        
        // Solid
        glBegin(GL_QUADS); glColor3f(0.3f, 0.3f, 0.35f); 
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5); // Front
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5); // Top
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5); // Back
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5); // Left
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5); // Right
        glEnd();
        
        // Wireframe
        glLineWidth(2.0f); glColor3f(1.0f, 0.0f, 1.0f); 
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        
        glPopMatrix();
    }
}

void draw_player_model(PlayerState *p) {
    if (!p->active || p->state == STATE_DEAD) return;
    
    glPushMatrix();
    glTranslatef(p->x, p->y + 1.9f, p->z); // Center of body
    glRotatef(-p->yaw, 0, 1, 0);
    
    // Body
    if(p->health <= 0) glColor3f(0.2, 0, 0); else glColor3f(1, 0, 0);
    glPushMatrix(); glScalef(0.6f, 1.8f, 0.6f); // Human Scale
    // (Simple box render)
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
    glPopMatrix();
    
    // Head
    glPushMatrix(); glTranslatef(0, 0.95f, 0);
    glColor3f(0, 1, 1);
    glScalef(0.4f, 0.4f, 0.4f);
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
    glPopMatrix();
    
    glPopMatrix();
}

void draw_hud(PlayerState *p) {
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    
    // Crosshair
    glColor3f(0, 1, 0); glLineWidth(2.0f);
    glBegin(GL_LINES); glVertex2f(630, 360); glVertex2f(650, 360); glVertex2f(640, 350); glVertex2f(640, 370); glEnd();
    
    // Bars
    glColor3f(0.2f, 0, 0); glRectf(50, 50, 250, 70);
    glColor3f(1.0f, 0, 0); glRectf(50, 50, 50 + (p->health * 2), 70);
    glColor3f(0, 0, 0.2f); glRectf(50, 80, 250, 100);
    glColor3f(0.2f, 0.2f, 1.0f); glRectf(50, 80, 50 + (p->shield * 2), 100);
    
    // Ammo
    glColor3f(1, 1, 0);
    int ammo = (p->current_weapon == WPN_KNIFE) ? 99 : p->ammo[p->current_weapon];
    for(int i=0; i<ammo; i++) glRectf(1200 - (i*10), 50, 1205 - (i*10), 80);
    
    // Debug Text (Primitive)
    // "POS X Y Z" drawn with lines
    
    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();
    
    // Camera
    float cam_h = render_p->crouching ? 2.5f : EYE_HEIGHT;
    glRotatef(-cam_pitch, 1, 0, 0);
    glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);
    
    draw_grid();
    draw_map();
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        draw_player_model(&local_state.players[i]);
    }
    
    draw_hud(render_p);
}

// --- MAIN ---
int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT [PHASE 432 RECOVERY]", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    
    // AUTO START LISTEN SERVER
    local_init_match(8, MODE_DEATHMATCH);
    
    // Bind
    #ifdef _WIN32
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    #endif
    sv_sock = socket(AF_INET, SOCK_DGRAM, 0);
    u_long mode = 1; ioctlsocket(sv_sock, FIONBIO, &mode);
    struct sockaddr_in bind_addr;
    bind_addr.sin_family = AF_INET; bind_addr.sin_port = htons(6969); bind_addr.sin_addr.s_addr = INADDR_ANY;
    bind(sv_sock, (struct sockaddr*)&bind_addr, sizeof(bind_addr));
    
    // Mouse
    SDL_SetRelativeMouseMode(SDL_TRUE);
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
    glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
    
    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) running = 0;
            
            if(e.type == SDL_MOUSEMOTION) {
                cam_yaw -= e.motion.xrel * 0.1f;
                if(cam_yaw > 360) cam_yaw -= 360; if(cam_yaw < 0) cam_yaw += 360;
                cam_pitch -= e.motion.yrel * 0.1f;
                if(cam_pitch > 89) cam_pitch = 89; if(cam_pitch < -89) cam_pitch = -89;
            }
        }
        
        const Uint8 *k = SDL_GetKeyboardState(NULL);
        float fwd=0, str=0;
        if(k[SDL_SCANCODE_W]) fwd-=1; if(k[SDL_SCANCODE_S]) fwd+=1;
        if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
        int jump = k[SDL_SCANCODE_SPACE];
        int crouch = k[SDL_SCANCODE_LCTRL];
        int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
        int reload = k[SDL_SCANCODE_R];
        
        if(k[SDL_SCANCODE_1]) wpn_req=0; if(k[SDL_SCANCODE_2]) wpn_req=1; 
        if(k[SDL_SCANCODE_3]) wpn_req=2; if(k[SDL_SCANCODE_4]) wpn_req=3; if(k[SDL_SCANCODE_5]) wpn_req=4;

        UserCmd cmd = client_create_cmd(fwd, str, cam_yaw, cam_pitch, shoot, jump, crouch, reload, wpn_req);
        net_send_cmd(cmd);
        sv_tick();
        
        draw_scene(&local_state.players[0]);
        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    
    SDL_Quit();
    return 0;
}
