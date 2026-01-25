#define SDL_MAIN_HANDLED
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <winsock2.h>
#include <ws2tcpip.h>
#include "../../../packages/protocol/protocol.h"
#include "../../../packages/simulation/game_physics.h" 
#include "../../../packages/map/map.h" 
#pragma comment(lib, "ws2_32.lib")
#include <SDL2/SDL.h>
#include <SDL2/SDL_opengl.h>
#include <GL/glu.h>
#include "../../shank-fps/src/font.h"
#include "../../shank-fps/src/particles.h" 
#include "../../../packages/simulation/local_game.h" 

#define STATE_LOBBY 0
#define STATE_GAME_NET 1
#define STATE_GAME_LOCAL 2
int app_state = STATE_LOBBY;

SOCKET sock;
struct sockaddr_in server_addr;
long long last_packet_time = 0;
ServerState local_state; 
int my_player_id = 0;
int local_pid = 0;
char lobby_status[128] = "STATUS: IDLE"; 
char my_name[32] = "Guest"; 

float cam_yaw = 0; float cam_pitch = 0;
int zoom_held = 0; 

long long current_ms() { return (long long)SDL_GetTicks(); }

void net_init() {
    WSADATA wsa; WSAStartup(MAKEWORD(2,2), &wsa);
    sock = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    u_long mode = 1; ioctlsocket(sock, FIONBIO, &mode);
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(PORT);
    struct hostent *he = gethostbyname("s.farthq.com");
    if (he) server_addr.sin_addr = *((struct in_addr*)he->h_addr); else server_addr.sin_addr.s_addr = inet_addr("127.0.0.1");
}

void net_send_cmd(int cmd) {
    Packet pkt; pkt.type = PACKET_LOBBY_CMD; pkt.cmd_id = cmd;
    if (cmd == CMD_NAME) strncpy(pkt.data, my_name, 31);
    sendto(sock, (char*)&pkt, sizeof(int)*3 + (cmd==CMD_NAME?32:0), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
}

void net_send_input(float fwd, float strafe, int jump, int crouch, int shoot, int reload, int weapon, int zoom) {
    Packet pkt; pkt.type = PACKET_INPUT; pkt.owner_id = my_player_id;
    ClientInput in; in.fwd = fwd; in.strafe = strafe; in.yaw = cam_yaw; in.pitch = cam_pitch; in.jump = jump; in.crouch = crouch; in.shoot = shoot; in.reload = reload; in.weapon_req = weapon; in.zoom = zoom;
    memcpy(pkt.data, &in, sizeof(ClientInput));
    sendto(sock, (char*)&pkt, sizeof(int)*3 + sizeof(ClientInput), 0, (struct sockaddr*)&server_addr, sizeof(server_addr));
}

void net_update() {
    char buf[4096]; int len;
    while((len = recv(sock, buf, sizeof(buf), 0)) > 0) {
        Packet *pkt = (Packet*)buf;
        if (pkt->type == PACKET_STATE && len >= sizeof(int)*3 + sizeof(ServerState)) {
            memcpy(&local_state, pkt->data, sizeof(ServerState));
            my_player_id = pkt->owner_id;
            local_pid = my_player_id;
            last_packet_time = current_ms();
        }
        else if (pkt->type == PACKET_LOBBY_INFO) {
             ServerState info; memcpy(&info, pkt->data, sizeof(ServerState));
             sprintf(lobby_status, "%s", info.status_msg);
        }
    }
}

void draw_cube(float sx, float sy, float sz) {
    glPushMatrix(); glScalef(sx, sy, sz); glBegin(GL_QUADS);
    // Bevel effect (fake lighting)
    // Top
    glColor3f(1.0f, 1.0f, 1.0f); glVertex3f(-0.5,0.5,-0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5);
    // Front
    glColor3f(0.8f, 0.8f, 0.8f); glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
    // Back
    glColor3f(0.8f, 0.8f, 0.8f); glVertex3f(0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,0.5,-0.5); glVertex3f(0.5,0.5,-0.5);
    // Left
    glColor3f(0.6f, 0.6f, 0.6f); glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
    // Right
    glColor3f(0.6f, 0.6f, 0.6f); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
    // Bottom
    glColor3f(0.2f, 0.2f, 0.2f); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,0.5);
    glEnd(); glPopMatrix();
}

void draw_nameplate(Vec3 pos, char* name) {
    GLdouble model[16]; glGetDoublev(GL_MODELVIEW_MATRIX, model);
    GLdouble proj[16]; glGetDoublev(GL_PROJECTION_MATRIX, proj);
    GLint view[4]; glGetIntegerv(GL_VIEWPORT, view);
    GLdouble sx, sy, sz;
    if (gluProject(pos.x, pos.y + 2.2f, pos.z, model, proj, view, &sx, &sy, &sz) == GL_TRUE) {
        if (sz > 0.0f && sz < 1.0f) {
            glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 720, 0);
            glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity(); glDisable(GL_DEPTH_TEST);
            int text_w = strlen(name) * 15;
            // SHADOW
            draw_text(name, (int)sx - (text_w/2) + 2, (int)(720 - sy) + 2, 20.0f, 0, 0, 0); 
            // TEXT
            draw_text(name, (int)sx - (text_w/2), (int)(720 - sy), 20.0f, 1, 1, 0);
            glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
        }
    }
}

void draw_player(PlayerState *p, int tick) {
    glPushMatrix(); glTranslatef(p->pos.x, p->pos.y, p->pos.z); glRotatef(-p->yaw, 0, 1, 0);
    // Gold/Yellow Body (Like video "Shank")
    glColor3f(1.0f, 0.8f, 0.0f); glPushMatrix(); glTranslatef(0, 1.0f, 0); draw_cube(0.5f, 0.6f, 0.3f); glPopMatrix();
    // Head
    glColor3f(1.0f, 0.9f, 0.5f); glPushMatrix(); glTranslatef(0, 1.5f, 0); draw_cube(0.3f, 0.3f, 0.3f); glPopMatrix();
    glPopMatrix();
    draw_nameplate(p->pos, p->name);
}

void render_gun(PlayerState *me) {
    glLoadIdentity(); glClear(GL_DEPTH_BUFFER_BIT); if (me->current_weapon == 4 && zoom_held) return;
    float bob = sinf(local_state.server_tick * 0.2f) * 0.02f * (sqrtf(me->vel.x*me->vel.x + me->vel.z*me->vel.z) * 5.0f);
    glTranslatef(0.3f, -0.3f + bob, -0.5f); glRotatef(180.0f, 0, 1, 0);
    glColor3f(0.1f, 0.1f, 0.1f); glScalef(0.15f, 0.2f, 1.0f); draw_cube(1,1,1);
}

void render_game() {
    // VOID SKY (Dark Blue/Black)
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f); 
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    
    PlayerState *me = &local_state.players[my_player_id];
    float fov = (me->current_weapon == 4 && zoom_held) ? 20.0f : 70.0f;
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(fov, 1280.0f/720.0f, 0.1f, 1000.0f);
    glMatrixMode(GL_MODELVIEW); glLoadIdentity();
    glRotatef(-cam_pitch, 1.0f, 0.0f, 0.0f); glRotatef(-cam_yaw, 0.0f, 1.0f, 0.0f);
    glTranslatef(-me->pos.x, -(me->pos.y + 1.7f), -me->pos.z);
    
    // NEON CYAN GRID
    glLineWidth(2.0f);
    glBegin(GL_LINES); glColor3f(0.0f, 1.0f, 1.0f); // Cyan
    for(int i=-50; i<=50; i+=2) { glVertex3f(i,0,-50); glVertex3f(i,0,50); glVertex3f(-50,0,i); glVertex3f(50,0,i); } 
    glEnd();
    
    // MAP BLOCKS (Colored per type)
    for(int i=0; i<local_level.wall_count; i++) {
        Wall *w = &local_level.walls[i]; 
        glPushMatrix(); glTranslatef(w->x, w->y, w->z); glScalef(w->sx, w->sy, w->sz);
        if (w->id == 1) continue; // Skip floor block (already have grid)
        
        // Block Color
        if (w->id >= 10) glColor3f(0.8f, 0.2f, 0.8f); // Magenta blocks (Video style)
        else glColor3f(0.2f, 0.2f, 0.8f); // Blue Walls
        
        draw_cube(1,1,1); 
        glPopMatrix();
    }

    for(int i=0; i<MAX_CLIENTS; i++) {
        if (local_state.players[i].active && i != my_player_id) {
            draw_player(&local_state.players[i], local_state.server_tick);
        }
    }
    render_gun(me);
    
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 720, 0);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity(); glDisable(GL_DEPTH_TEST);
    
    char buf[128]; 
    if (app_state == STATE_LOBBY) {
        draw_text("SHANK PIT LOBBY v42", 50, 50, 40.0f, 1, 1, 0);
        draw_text("[C] CREATE  [S] SEARCH  [J] JOIN", 50, 130, 20.0f, 0, 1, 1);
        char name_ui[64]; sprintf(name_ui, "NAME: %s_", my_name);
        draw_text(name_ui, 50, 250, 30.0f, 1, 1, 1);
        draw_text(lobby_status, 50, 400, 30.0f, 1, 0.5, 0);
    } else {
        sprintf(buf, "HP %d  AMMO %d", me->health, me->ammo[me->current_weapon]); 
        draw_text(buf, 50, 650, 40.0f, 1, 1, 1);
        if (app_state == STATE_GAME_NET && current_ms() - last_packet_time > 3000) draw_text("CONNECTION LOST", 480, 360, 30.0f, 1, 0, 0);
    }
    float cx=640, cy=360;
    glBegin(GL_LINES); glColor3f(0,1,1); glVertex2f(cx-10,cy); glVertex2f(cx+10,cy); glVertex2f(cx,cy-10); glVertex2f(cx,cy+10); glEnd(); 
    
    glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANK PIT v42 (SLIPPERY)", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GLContext gl = SDL_GL_CreateContext(win);
    glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 720, 0); glMatrixMode(GL_MODELVIEW); glLoadIdentity();
    net_init(); SDL_StartTextInput(); particles_init(); last_packet_time = current_ms();
    int running = 1;
    
    local_init(); 

    while(running) {
        if (app_state != STATE_GAME_LOCAL) net_update();
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if (app_state == STATE_LOBBY) {
                if(e.type == SDL_TEXTINPUT) { if(strlen(my_name) < 30) strcat(my_name, e.text.text); net_send_cmd(CMD_NAME); }
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_BACKSPACE && strlen(my_name) > 0) { my_name[strlen(my_name)-1] = 0; net_send_cmd(CMD_NAME); }
                    if (e.key.keysym.sym == SDLK_c) net_send_cmd(CMD_CREATE);
                    if (e.key.keysym.sym == SDLK_s) net_send_cmd(CMD_SEARCH);
                    if (e.key.keysym.sym == SDLK_j) { SDL_SetRelativeMouseMode(SDL_TRUE); glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(70.0f, 1280.0f/720.0f, 0.1f, 1000.0f); glMatrixMode(GL_MODELVIEW); app_state = STATE_GAME_NET; }
                    if (e.key.keysym.sym == SDLK_d) { local_init(); app_state = STATE_GAME_LOCAL; SDL_SetRelativeMouseMode(SDL_TRUE); glEnable(GL_DEPTH_TEST); glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(70.0f, 1280.0f/720.0f, 0.1f, 1000.0f); glMatrixMode(GL_MODELVIEW); }
                }
            } else {
                if(e.type == SDL_KEYDOWN && e.key.keysym.sym == SDLK_ESCAPE) { app_state = STATE_LOBBY; SDL_SetRelativeMouseMode(SDL_FALSE); glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 720, 0); glMatrixMode(GL_MODELVIEW); glLoadIdentity(); glDisable(GL_DEPTH_TEST); }
                if(e.type == SDL_MOUSEMOTION) { cam_yaw -= e.motion.xrel * 0.1f; cam_pitch -= e.motion.yrel * 0.1f; if(cam_pitch > 89) cam_pitch = 89; if(cam_pitch < -89) cam_pitch = -89; }
            }
        }
        if (app_state != STATE_LOBBY) {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0; if(k[SDL_SCANCODE_W]) fwd++; if(k[SDL_SCANCODE_S]) fwd--; if(k[SDL_SCANCODE_D]) str++; if(k[SDL_SCANCODE_A]) str--;
            int shoot = (SDL_GetMouseState(NULL,NULL)&SDL_BUTTON(SDL_BUTTON_LEFT)); int zoom = (SDL_GetMouseState(NULL,NULL)&SDL_BUTTON(SDL_BUTTON_RIGHT)); zoom_held = zoom;
            int wpn=-1; if(k[SDL_SCANCODE_1]) wpn=0; if(k[SDL_SCANCODE_2]) wpn=1; if(k[SDL_SCANCODE_3]) wpn=2; if(k[SDL_SCANCODE_4]) wpn=3; if(k[SDL_SCANCODE_5]) wpn=4;
            if(app_state==STATE_GAME_NET) net_send_input(fwd, str, k[SDL_SCANCODE_SPACE], k[SDL_SCANCODE_LCTRL], shoot, k[SDL_SCANCODE_R], wpn, zoom);
            else { local_pid = 0; local_update(0.016f, fwd, str, cam_yaw, cam_pitch, shoot, wpn, k[SDL_SCANCODE_SPACE], k[SDL_SCANCODE_LCTRL], k[SDL_SCANCODE_R]); }
        }
        render_game();
        SDL_GL_SwapWindow(win); SDL_Delay(16);
    }
    return 0;
}
