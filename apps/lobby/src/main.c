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

#define STATE_LOBBY 0
#define STATE_GAME_NET 1
#define STATE_GAME_LOCAL 2
int app_state = STATE_LOBBY;

float cam_yaw = 0.0f;
float cam_pitch = 0.0f;
float current_fov = 75.0f;
char chat_buf[64] = {0};

int sock = -1;
struct sockaddr_in server_addr;

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

    struct hostent *he = gethostbyname("s.farthq.com");
    if (he) {
        server_addr.sin_family = AF_INET;
        server_addr.sin_port = htons(5314); 
        memcpy(&server_addr.sin_addr, he->h_addr_list[0], he->h_length);
    } else {
        server_addr.sin_family = AF_INET;
        server_addr.sin_port = htons(6969);
        server_addr.sin_addr.s_addr = inet_addr("127.0.0.1");
    }
}

// --- HELPER: DRAW LETTER VECTORS ---
// Draws a letter at (x,y) with size (w,h)
void draw_char(char c, float x, float y, float w, float h) {
    glBegin(GL_LINES);
    switch(c) {
        case 'S': 
            glVertex2f(x+w, y); glVertex2f(x, y); 
            glVertex2f(x, y); glVertex2f(x, y+h/2);
            glVertex2f(x, y+h/2); glVertex2f(x+w, y+h/2);
            glVertex2f(x+w, y+h/2); glVertex2f(x+w, y+h);
            glVertex2f(x+w, y+h); glVertex2f(x, y+h);
            break;
        case 'H':
            glVertex2f(x, y); glVertex2f(x, y+h);
            glVertex2f(x+w, y); glVertex2f(x+w, y+h);
            glVertex2f(x, y+h/2); glVertex2f(x+w, y+h/2);
            break;
        case 'A':
            glVertex2f(x, y+h); glVertex2f(x+w/2, y);
            glVertex2f(x+w/2, y); glVertex2f(x+w, y+h);
            glVertex2f(x+w/4, y+h/2); glVertex2f(x+w*0.75f, y+h/2);
            break;
        case 'N':
            glVertex2f(x, y+h); glVertex2f(x, y);
            glVertex2f(x, y); glVertex2f(x+w, y+h);
            glVertex2f(x+w, y+h); glVertex2f(x+w, y);
            break;
        case 'K':
            glVertex2f(x, y); glVertex2f(x, y+h);
            glVertex2f(x+w, y); glVertex2f(x, y+h/2);
            glVertex2f(x, y+h/2); glVertex2f(x+w, y+h);
            break;
        case 'P':
            glVertex2f(x, y+h); glVertex2f(x, y);
            glVertex2f(x, y); glVertex2f(x+w, y);
            glVertex2f(x+w, y); glVertex2f(x+w, y+h/2);
            glVertex2f(x+w, y+h/2); glVertex2f(x, y+h/2);
            break;
        case 'I':
            glVertex2f(x+w/2, y); glVertex2f(x+w/2, y+h);
            glVertex2f(x, y); glVertex2f(x+w, y);
            glVertex2f(x, y+h); glVertex2f(x+w, y+h);
            break;
        case 'T':
            glVertex2f(x+w/2, y); glVertex2f(x+w/2, y+h);
            glVertex2f(x, y); glVertex2f(x+w, y);
            break;
        case 'D': // Menu D
            glVertex2f(x, y); glVertex2f(x, y+h);
            glVertex2f(x, y); glVertex2f(x+w/2, y);
            glVertex2f(x+w/2, y); glVertex2f(x+w, y+h/2);
            glVertex2f(x+w, y+h/2); glVertex2f(x+w/2, y+h);
            glVertex2f(x+w/2, y+h); glVertex2f(x, y+h);
            break;
        case 'B': // Menu B
            glVertex2f(x, y); glVertex2f(x, y+h);
            glVertex2f(x, y); glVertex2f(x+w*0.8, y);
            glVertex2f(x+w*0.8, y); glVertex2f(x+w, y+h/4);
            glVertex2f(x+w, y+h/4); glVertex2f(x+w*0.8, y+h/2);
            glVertex2f(x+w*0.8, y+h/2); glVertex2f(x, y+h/2);
            glVertex2f(x+w*0.8, y+h/2); glVertex2f(x+w, y+h*0.75);
            glVertex2f(x+w, y+h*0.75); glVertex2f(x+w*0.8, y+h);
            glVertex2f(x+w*0.8, y+h); glVertex2f(x, y+h);
            break;
        case 'J': // Menu J
            glVertex2f(x+w, y); glVertex2f(x+w, y+h*0.8);
            glVertex2f(x+w, y+h*0.8); glVertex2f(x+w/2, y+h);
            glVertex2f(x+w/2, y+h); glVertex2f(x, y+h*0.8);
            glVertex2f(x, y+h*0.8); glVertex2f(x, y+h*0.6);
            break; 
        case 'v': // small v
            glVertex2f(x, y); glVertex2f(x+w/2, y+h);
            glVertex2f(x+w/2, y+h); glVertex2f(x+w, y);
            break;
    }
    glEnd();
}

void draw_text_string(const char* s, float x, float y, float size) {
    float cursor = x;
    float w = size * 0.6f;
    float h = size;
    float gap = size * 0.2f;
    
    while(*s) {
        draw_char(*s, cursor, y, w, h);
        cursor += w + gap;
        s++;
    }
}

// --- TITLE SCREEN RENDERER ---
void draw_lobby_screen() {
    glLoadIdentity();
    glClearColor(0.05f, 0.05f, 0.05f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT);
    
    float time = SDL_GetTicks() * 0.005f;
    
    // 1. TITLE "SHANKPIT" (CMYK Glitch)
    float tx = 200, ty = 100, tsize = 100;
    
    glLineWidth(3.0f);
    
    // Cyan Layer (Shift Left)
    float jx = sinf(time) * 4.0f;
    float jy = cosf(time * 1.3f) * 2.0f;
    glColor3f(0, 1, 1);
    draw_text_string("SHANKPIT", tx + jx, ty + jy, tsize);
    
    // Magenta Layer (Shift Right)
    glColor3f(1, 0, 1);
    draw_text_string("SHANKPIT", tx - jx, ty - jy, tsize);
    
    // Yellow Layer (Center + Glow)
    glColor3f(1, 1, 0); // Yellow
    draw_text_string("SHANKPIT", tx, ty, tsize);
    
    // 2. VERSION
    glLineWidth(1.0f);
    glColor3f(0.5f, 0.5f, 0.5f);
    draw_text_string("v0.225", 900, 180, 20);
    
    // 3. MENU OPTIONS (Grid Layout)
    float mx = 300, my = 350, msize = 40, mgap = 60;
    
    // [D] 1 Bot
    glColor3f(0, 1, 0); // Green
    draw_char('D', mx, my, msize, msize);
    glColor3f(1, 1, 1);
    draw_text_string("DUEL 1v1", mx + 60, my + 10, 20);
    
    // [B] Horde
    glColor3f(1, 0.5f, 0); // Orange
    draw_char('B', mx, my + mgap, msize, msize);
    glColor3f(1, 1, 1);
    draw_text_string("HORDE 32", mx + 60, my + mgap + 10, 20);
    
    // [S] Neural
    float flash = (sinf(time*5) + 1.0f) * 0.5f;
    glColor3f(0, flash, flash); // Cyan Flash
    draw_char('S', mx, my + mgap*2, msize, msize);
    glColor3f(1, 1, 1);
    draw_text_string("NEURAL S", mx + 60, my + mgap*2 + 10, 20);
    
    // [N] Net
    glColor3f(0.5f, 0.5f, 1.0f); // Blue
    draw_char('N', mx, my + mgap*3, msize, msize);
    glColor3f(1, 1, 1);
    draw_text_string("NET PLAY", mx + 60, my + mgap*3 + 10, 20);
}

void draw_grid() {
    glBegin(GL_LINES);
    glColor3f(0.0f, 1.0f, 1.0f);
    for(int i=-100; i<=100; i+=5) {
        glVertex3f(i, 0, -100); glVertex3f(i, 0, 100);
        glVertex3f(-100, 0, i); glVertex3f(100, 0, i);
    }
    glEnd();
}

void draw_map() {
    for(int i=1; i<map_count; i++) {
        Box b = map_geo[i];
        glPushMatrix();
        glTranslatef(b.x, b.y, b.z);
        glScalef(b.w, b.h, b.d);
        glBegin(GL_QUADS);
        glColor3f(0.2f, 0.2f, 0.2f); 
        glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
        glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
        glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
        glEnd();
        glLineWidth(2.0f);
        glColor3f(1.0f, 0.0f, 1.0f); 
        glBegin(GL_LINE_LOOP);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        glPopMatrix();
    }
}

void draw_gun_model(int weapon_id) {
    switch(weapon_id) {
        case WPN_KNIFE:   glColor3f(0.8f, 0.8f, 0.9f); glScalef(0.05f, 0.05f, 0.8f); break;
        case WPN_MAGNUM:  glColor3f(0.4f, 0.4f, 0.4f); glScalef(0.15f, 0.2f, 0.5f); break;
        case WPN_AR:      glColor3f(0.2f, 0.3f, 0.2f); glScalef(0.1f, 0.15f, 1.2f); break;
        case WPN_SHOTGUN: glColor3f(0.5f, 0.3f, 0.2f); glScalef(0.25f, 0.15f, 0.8f); break;
        case WPN_SNIPER:  glColor3f(0.1f, 0.1f, 0.15f); glScalef(0.08f, 0.12f, 2.0f); break;
        default: glColor3f(1,0,1); glScalef(0.1f, 0.1f, 0.5f);
    }
    glBegin(GL_QUADS); 
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); 
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); 
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); 
    glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1); 
    glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); glVertex3f(-1,-1,-1); 
    glEnd();
}

void draw_weapon_p(PlayerState *p) {
    glPushMatrix();
    glLoadIdentity();
    float kick = p->recoil_anim * 0.2f;
    float reload_dip = (p->reload_timer > 0) ? sinf(p->reload_timer * 0.2f) * 0.5f - 0.5f : 0.0f;
    float speed = sqrtf(p->vx*p->vx + p->vz*p->vz);
    float bob = sinf(SDL_GetTicks() * 0.015f) * speed * 0.15f; 

    float x_offset = (current_fov < 50.0f) ? 0.25f : 0.4f;
    glTranslatef(x_offset, -0.5f + kick + reload_dip + (bob * 0.5f), -1.2f + (kick * 0.5f) + bob);
    glRotatef(-p->recoil_anim * 10.0f, 1, 0, 0);
    draw_gun_model(p->current_weapon);
    glPopMatrix();
}

void draw_player_3rd(PlayerState *p) {
    glPushMatrix();
    glTranslatef(p->x, p->y + 2.0f, p->z);
    glRotatef(-p->yaw, 0, 1, 0); 
    
    if(p->health <= 0) glColor3f(0.2, 0, 0); 
    else glColor3f(1, 0, 0); 
    
    glPushMatrix();
    glScalef(1.2f, 3.8f, 1.2f);
    glBegin(GL_QUADS);
    glVertex3f(-0.5,-0.5,0.5); glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(-0.5,0.5,0.5);
    glVertex3f(-0.5,0.5,0.5); glVertex3f(0.5,0.5,0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5,0.5); glVertex3f(-0.5,0.5,0.5); glVertex3f(-0.5,0.5,-0.5);
    glVertex3f(0.5,-0.5,0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5,0.5,-0.5); glVertex3f(0.5,0.5,0.5);
    glEnd();
    glPopMatrix();
    
    glPushMatrix();
    glTranslatef(0.5f, 1.0f, 0.5f);
    glRotatef(p->pitch, 1, 0, 0);   
    glScalef(0.8f, 0.8f, 0.8f);
    draw_gun_model(p->current_weapon);
    glPopMatrix();
    glPopMatrix();
}

void draw_projectiles() {}

void draw_scene(PlayerState *render_p) {
    glClearColor(0.05f, 0.05f, 0.1f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();

    float cam_h = render_p->crouching ? 2.5f : EYE_HEIGHT;
    glRotatef(-cam_pitch, 1, 0, 0);
    glRotatef(-cam_yaw, 0, 1, 0);
    glTranslatef(-render_p->x, -(render_p->y + cam_h), -render_p->z);

    draw_grid();
    draw_map();
    draw_projectiles();
    
    for(int i=1; i<MAX_CLIENTS; i++) {
        if(local_state.players[i].active) {
            draw_player_3rd(&local_state.players[i]);
        }
    }

    draw_weapon_p(render_p);
    
    glDisable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    
    glColor3f(0, 1, 0);
    if (current_fov < 50.0f) {
        glBegin(GL_LINES); glVertex2f(0, 360); glVertex2f(1280, 360); glVertex2f(640, 0); glVertex2f(640, 720); glEnd();
    } else {
        glLineWidth(2.0f);
        glBegin(GL_LINES); glVertex2f(630, 360); glVertex2f(650, 360); glVertex2f(640, 350); glVertex2f(640, 370); glEnd();
    }
    
    if (render_p->hit_feedback > 0) {
        glColor3f(0, 1, 1);
        glBegin(GL_TRIANGLE_FAN);
        glVertex2f(50, 50); 
        for(int i=0; i<=20; i++) {
            float ang = (float)i / 20.0f * 6.28318f;
            glVertex2f(50 + cosf(ang)*20, 50 + sinf(ang)*20);
        }
        glEnd();
    }
    
    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPopMatrix(); glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT [v0.225]", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    net_init();
    local_init_match(1);

    int running = 1;
    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            
            if (app_state == STATE_LOBBY) {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_d) {
                        app_state = STATE_GAME_LOCAL;
                        USE_NEURAL_NET = 0;
                        local_init_match(1);
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                    if (e.key.keysym.sym == SDLK_b) {
                        app_state = STATE_GAME_LOCAL;
                        USE_NEURAL_NET = 0; 
                        local_init_match(31); 
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                    if (e.key.keysym.sym == SDLK_s) { 
                        app_state = STATE_GAME_LOCAL;
                        USE_NEURAL_NET = 1; 
                        local_init_match(31);
                        printf("MODE S ACTIVATED.\n");
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                    if (e.key.keysym.sym == SDLK_n) {
                        app_state = STATE_GAME_NET;
                        SDL_SetRelativeMouseMode(SDL_TRUE);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0);
                        glMatrixMode(GL_MODELVIEW); glEnable(GL_DEPTH_TEST);
                    }
                }
            } 
            else {
                if(e.type == SDL_KEYDOWN) {
                    if (e.key.keysym.sym == SDLK_ESCAPE) {
                        app_state = STATE_LOBBY;
                        SDL_SetRelativeMouseMode(SDL_FALSE);
                        glDisable(GL_DEPTH_TEST);
                        glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
                        glMatrixMode(GL_MODELVIEW); glLoadIdentity();
                    }
                }
                if(e.type == SDL_MOUSEMOTION) {
                    float sens = (current_fov < 50.0f) ? 0.05f : 0.15f; 
                    cam_yaw -= e.motion.xrel * sens;
                    if(cam_yaw > 360) cam_yaw -= 360; if(cam_yaw < 0) cam_yaw += 360;
                    cam_pitch -= e.motion.yrel * sens;
                    if(cam_pitch > 89) cam_pitch = 89; if(cam_pitch < -89) cam_pitch = -89;
                }
            }
        }

        if (app_state == STATE_LOBBY) {
            // Use new HUD function
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); gluOrtho2D(0, 1280, 0, 720);
            glMatrixMode(GL_MODELVIEW); glLoadIdentity();
            draw_lobby_screen();
        } 
        else if (app_state == STATE_GAME_LOCAL) {
            const Uint8 *k = SDL_GetKeyboardState(NULL);
            float fwd=0, str=0;
            if(k[SDL_SCANCODE_W]) fwd+=1; if(k[SDL_SCANCODE_S]) fwd-=1;
            if(k[SDL_SCANCODE_D]) str+=1; if(k[SDL_SCANCODE_A]) str-=1;
            int jump = k[SDL_SCANCODE_SPACE];
            int crouch = k[SDL_SCANCODE_LCTRL];
            int shoot = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_LEFT));
            int rmb = (SDL_GetMouseState(NULL, NULL) & SDL_BUTTON(SDL_BUTTON_RIGHT));
            int reload = k[SDL_SCANCODE_R];
            int wpn = -1;
            if(k[SDL_SCANCODE_1]) wpn=0; if(k[SDL_SCANCODE_2]) wpn=1; if(k[SDL_SCANCODE_3]) wpn=2;
            if(k[SDL_SCANCODE_4]) wpn=3; if(k[SDL_SCANCODE_5]) wpn=4;

            float target_fov = (rmb && local_state.players[0].current_weapon == WPN_SNIPER) ? 20.0f : 75.0f;
            current_fov += (target_fov - current_fov) * 0.2f;
            glMatrixMode(GL_PROJECTION); glLoadIdentity(); 
            gluPerspective(current_fov, 1280.0/720.0, 0.1, 1000.0);
            glMatrixMode(GL_MODELVIEW);

            local_update(fwd, str, cam_yaw, cam_pitch, shoot, wpn, jump, crouch, reload);
            draw_scene(&local_state.players[0]);
        }
        else if (app_state == STATE_GAME_NET) {
            draw_scene(&local_state.players[0]); 
        }

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
