#define SDL_MAIN_HANDLED
#include <stdio.h>
#include "shank.h"
#include "input.h"
#include "net.h"
#include "font.h"
#include "particles.h"
#include <GL/glu.h>
#include "../../../packages/map/map.h"
#include "input.c"
#include "net.c"
#include "../../../packages/map/map.c"

GameContext game;
GameMap level;
int show_scoreboard = 0;
int show_help = 0;
char kill_feed[5][64];
int kill_timer = 0;

void add_kill_msg(const char* msg) {
    for(int i=0; i<4; i++) strcpy(kill_feed[i], kill_feed[i+1]);
    strcpy(kill_feed[4], msg); kill_timer = 300;
}

void init_opengl() {
    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glLoadIdentity();
    gluPerspective(FOV, (float)WINDOW_WIDTH / WINDOW_HEIGHT, 0.1f, 1000.0f);
    glMatrixMode(GL_MODELVIEW); glPointSize(4.0f);
}

void render_hud() {
    glMatrixMode(GL_PROJECTION); glPushMatrix(); glLoadIdentity();
    gluOrtho2D(0, WINDOW_WIDTH, WINDOW_HEIGHT, 0);
    glMatrixMode(GL_MODELVIEW); glPushMatrix(); glLoadIdentity();
    glDisable(GL_DEPTH_TEST);

    Player *p = &game.local_player;
    char buf[64];
    
    // HP/Ammo
    sprintf(buf, "HP %d", p->health);
    draw_text(buf, 50, WINDOW_HEIGHT - 80, 40.0f, 1, p->health < 30 ? 0 : 1, 0.2);
    sprintf(buf, "AMMO %d", p->ammo[p->current_weapon]);
    draw_text(buf, WINDOW_WIDTH - 300, WINDOW_HEIGHT - 80, 40.0f, 1, 1, 0);

    // Timer
    int seconds = (game.last_server_state.match_timer / 60) % 60;
    int minutes = (game.last_server_state.match_timer / 60) / 60;
    sprintf(buf, "%02d:%02d", minutes, seconds);
    draw_text(buf, WINDOW_WIDTH/2 - 80, 50, 40.0f, 1, 1, 1);

    // Help Hint
    draw_text("[H] HELP", 20, 20, 15.0f, 0.5, 0.5, 0.5);

    // Toggle Help Overlay
    if (show_help) {
        glColor4f(0,0,0,0.8f);
        glBegin(GL_QUADS); glVertex2f(200, 150); glVertex2f(WINDOW_WIDTH-200, 150); glVertex2f(WINDOW_WIDTH-200, 450); glVertex2f(200, 450); glEnd();
        draw_text("COMMANDS", WINDOW_WIDTH/2 - 100, 180, 30.0f, 1, 1, 0);
        draw_text("WASD : MOVE", 250, 240, 20.0f, 1,1,1);
        draw_text("MOUSE : AIM / SHOOT", 250, 270, 20.0f, 1,1,1);
        draw_text("1 - 5 : WEAPONS", 250, 300, 20.0f, 1,1,1);
        draw_text("G : GRENADE", 250, 330, 20.0f, 1,1,1);
        draw_text("E : ENTER VEHICLE", 250, 360, 20.0f, 1,1,1);
        draw_text("TAB : SCOREBOARD", 250, 390, 20.0f, 1,1,1);
    }

    if (kill_timer > 0) {
        kill_timer--;
        for(int i=0; i<5; i++) { if(strlen(kill_feed[i]) > 0) draw_text(kill_feed[i], WINDOW_WIDTH - 400, 100 + (i*40), 20.0f, 1, 0.5, 0); }
    }

    glBegin(GL_LINES); glColor3f(0,1,1);
    glVertex2f(WINDOW_WIDTH/2 - 15, WINDOW_HEIGHT/2); glVertex2f(WINDOW_WIDTH/2 + 15, WINDOW_HEIGHT/2);
    glVertex2f(WINDOW_WIDTH/2, WINDOW_HEIGHT/2 - 15); glVertex2f(WINDOW_WIDTH/2, WINDOW_HEIGHT/2 + 15);
    glEnd();

    if (show_scoreboard) {
        glColor4f(0,0,0,0.8f);
        glBegin(GL_QUADS); glVertex2f(200, 100); glVertex2f(WINDOW_WIDTH-200, 100); glVertex2f(WINDOW_WIDTH-200, WINDOW_HEIGHT-100); glVertex2f(200, WINDOW_HEIGHT-100); glEnd();
        draw_text("SCOREBOARD", WINDOW_WIDTH/2 - 150, 120, 30.0f, 1,1,1);
        for(int i=0; i<MAX_CLIENTS; i++) {
            PlayerState *s = &game.last_server_state.players[i];
            if(s->active) {
                sprintf(buf, "PLAYER %d   K: %d   D: %d", i, s->kills, s->deaths);
                draw_text(buf, 250, 200 + (i*50), 25.0f, 0.8, 0.8, 0.8);
            }
        }
    }

    glEnable(GL_DEPTH_TEST);
    glMatrixMode(GL_PROJECTION); glPopMatrix();
    glMatrixMode(GL_MODELVIEW); glPopMatrix();
}

void render() {
    glClearColor(0.1f, 0.1f, 0.15f, 1.0f);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();
    glRotatef(-game.local_player.pitch, 1.0f, 0.0f, 0.0f);
    glRotatef(-game.local_player.yaw, 0.0f, 1.0f, 0.0f);
    glTranslatef(-game.local_player.pos.x, -game.local_player.pos.y - 1.7f, -game.local_player.pos.z);

    for(int i=0; i<level.wall_count; i++) {
        Wall *w = &level.walls[i]; glPushMatrix(); glTranslatef(w->x, w->y, w->z); glScalef(w->sx, w->sy, w->sz);
        glColor3f(w->r, w->g, w->b);
        glBegin(GL_QUADS);
            glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1);
            glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1);
            glVertex3f(-1,-1,-1); glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1);
            glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); glVertex3f(1,-1,1);
            glVertex3f(-1,1,-1); glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1);
            glVertex3f(-1,-1,-1); glVertex3f(1,-1,-1); glVertex3f(1,-1,1); glVertex3f(-1,-1,1);
        glEnd(); glPopMatrix();
    }
    
    glBegin(GL_LINES); glColor3f(0.3f, 0.3f, 0.3f); for(int i=-50; i<=50; i+=2) { glVertex3f(i,0,-50); glVertex3f(i,0,50); glVertex3f(-50,0,i); glVertex3f(50,0,i); } glEnd();
    update_render_particles(0.016f);

    for(int i=0; i<MAX_CLIENTS; i++) {
        PlayerState *p = &game.last_server_state.players[i];
        if (!p->active || i == 0 || p->vehicle_id != -1) continue; 
        if (p->hit_feedback == 1) spawn_particle(p->pos, 0, 0, 1);
        if (p->hit_feedback == 2) spawn_particle(p->pos, 1, 0, 0);
        glPushMatrix(); glTranslatef(p->pos.x, p->pos.y, p->pos.z);
        glColor3f(1.0f, 0.2f, 0.2f); if (p->is_shooting > 0) glColor3f(1.0f, 1.0f, 0.0f);
        glBegin(GL_QUADS); glVertex3f(-0.5f, 0.0f, 0.5f); glVertex3f(0.5f, 0.0f, 0.5f); glVertex3f(0.5f, 1.8f, 0.5f); glVertex3f(-0.5f, 1.8f, 0.5f); glEnd();
        glPopMatrix();
    }
    render_hud(); SDL_GL_SwapWindow(game.window);
}

int main(int argc, char* argv[]) {
    if (SDL_Init(SDL_INIT_VIDEO) < 0) return 1;
    game.window = SDL_CreateWindow("SHANK PIT ONLINE", SDL_WINDOWPOS_CENTERED, SDL_WINDOWPOS_CENTERED, WINDOW_WIDTH, WINDOW_HEIGHT, SDL_WINDOW_OPENGL | SDL_WINDOW_SHOWN);
    game.gl_context = SDL_GL_CreateContext(game.window);
    init_opengl(); input_init(&game); net_init(&game, "127.0.0.1"); map_init(&level); particles_init();
    game.running = true; Uint32 last_time = SDL_GetTicks();
    while (game.running) {
        Uint32 current_time = SDL_GetTicks(); float delta_time = (current_time - last_time) / 1000.0f; last_time = current_time;
        
        SDL_Event event;
        while(SDL_PollEvent(&event)) {
            if(event.type == SDL_QUIT) game.running = 0;
            if(event.type == SDL_KEYDOWN) {
                if(event.key.keysym.sym == SDLK_ESCAPE) game.running = 0;
                if(event.key.keysym.sym == SDLK_h) show_help = !show_help;
            }
        }

        const Uint8 *state = SDL_GetKeyboardState(NULL); show_scoreboard = state[SDL_SCANCODE_TAB];
        input_update(&game, delta_time); net_send_input(&game); net_receive_state(&game); render();
    }
    SDL_GL_DeleteContext(game.gl_context); SDL_DestroyWindow(game.window); SDL_Quit();
    return 0;
}
