#include <stdio.h>
#include <SDL2/SDL.h>
#include <GL/gl.h>
#include <GL/glu.h>
#include <math.h>

// --- HALO-LIKE PHYSICS CONSTANTS ---
#define GRAVITY 0.012f          // Lower gravity for floatiness
#define GRAVITY_FLOAT 0.006f    // Feather falling (Space held)
#define JUMP_FORCE 0.38f        // High jump
#define MOVE_ACCEL 0.025f       // Slow acceleration (builds up speed)
#define FRICTION 0.96f          // Slippery ground (drift)
#define AIR_FRICTION 0.99f      // Very slippery air
#define MOUSE_SENS 0.15f

// --- GAME STATE ---
int current_weapon = 1;
float recoil_anim = 0.0f;
int running = 1;

typedef struct {
    float x, y, z;
    float vx, vy, vz;
    float yaw, pitch;
    int on_ground;
    int crouching;
} Player;

Player p = {0, 10, 0, 0, 0, 0, 0, 0, 0, 0};

// --- MAP DATA (The Arena) ---
typedef struct { float x, y, z, w, h, d; } Box;
Box geometry[] = {
    {0, -1, 0, 200, 2, 200},      // Floor
    {15, 2.5, 15, 10, 5, 10},     // Big Red Box
    {-15, 1.5, -15, 10, 3, 10},   // Low Blue Box
    {0, 2.0, 20, 20, 4, 2},       // Wall
    {-20, 4.0, 5, 6, 8, 6},       // High Pillar (Needs crouch jump)
    {0, 6.0, -25, 4, 12, 4},      // Sniper Perch
    {10, 1.0, -10, 4, 2, 4}       // Step
};
int geo_count = 7;

// --- PHYSICS ENGINE ---
void update_physics(int space_held) {
    // 1. Gravity
    if (p.vy < 0 && space_held) p.vy -= GRAVITY_FLOAT;
    else p.vy -= GRAVITY;

    // 2. Apply Velocity
    p.x += p.vx;
    p.y += p.vy;
    p.z += p.vz;

    // 3. Floor Collision (Simple Plane)
    if (p.y < 0) {
        p.y = 0;
        p.vy = 0;
        p.on_ground = 1;
    } else {
        p.on_ground = 0;
    }

    // 4. Box Collision (AABB)
    float pw = 1.0f; // Player Width radius
    float ph = p.crouching ? 2.5f : 4.0f; // Player Height (Crouch vs Stand)
    
    for(int i=1; i<geo_count; i++) { // Skip floor (i=0) handled above
        Box b = geometry[i];
        // Check overlap
        if (p.x + pw > b.x - b.w/2 && p.x - pw < b.x + b.w/2 &&
            p.z + pw > b.z - b.d/2 && p.z - pw < b.z + b.d/2) {
            
            // Vertical check
            if (p.y < b.y + b.h/2 && p.y + ph > b.y - b.h/2) {
                // COLLISION!
                
                // Are we falling ONTO it?
                float prev_y = p.y - p.vy;
                if (prev_y >= b.y + b.h/2) {
                    p.y = b.y + b.h/2; // Snap to top
                    p.vy = 0;
                    p.on_ground = 1;
                    // JUMP BOOST logic: If holding space right as we land, preserve momentum
                    if (space_held) {
                        p.vx *= 1.05f; p.vz *= 1.05f;
                    }
                } else {
                    // Hit side/bottom - Stop horizontal movement (Wall slide)
                    p.vx *= 0.5f; 
                    p.vz *= 0.5f;
                    // Push out simple (not perfect, but works for arena)
                    float dx = p.x - b.x;
                    float dz = p.z - b.z;
                    if (fabs(dx) > fabs(dz)) {
                         p.x = (dx > 0) ? b.x + b.w/2 + pw : b.x - b.w/2 - pw;
                    } else {
                         p.z = (dz > 0) ? b.z + b.d/2 + pw : b.z - b.d/2 - pw;
                    }
                }
            }
        }
    }

    // 5. Friction
    float f = p.on_ground ? FRICTION : AIR_FRICTION;
    p.vx *= f;
    p.vz *= f;
}

// --- RENDER ---
void draw_grid() {
    glBegin(GL_LINES);
    // Neon Purple Grid
    glColor3f(0.6f, 0.0f, 1.0f);
    for(int i=-100; i<=100; i+=5) {
        glVertex3f(i, 0, -100); glVertex3f(i, 0, 100);
        glVertex3f(-100, 0, i); glVertex3f(100, 0, i);
    }
    glEnd();
}

void draw_boxes() {
    for(int i=1; i<geo_count; i++) {
        Box b = geometry[i];
        glPushMatrix();
        glTranslatef(b.x, b.y, b.z);
        glScalef(b.w, b.h, b.d);
        
        glBegin(GL_QUADS);
        // Top (Lit)
        glColor3f(0.8f, 0.2f, 0.2f);
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        // Sides (Darker)
        glColor3f(0.5f, 0.1f, 0.1f);
        glVertex3f(-0.5,-0.5, 0.5); glVertex3f(0.5,-0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(-0.5, 0.5, 0.5); // Front
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5, 0.5,-0.5); glVertex3f(0.5, 0.5,-0.5); glVertex3f(0.5,-0.5,-0.5); // Back
        glVertex3f(-0.5,-0.5,-0.5); glVertex3f(-0.5,-0.5, 0.5); glVertex3f(-0.5, 0.5, 0.5); glVertex3f(-0.5, 0.5,-0.5); // Left
        glVertex3f(0.5,-0.5, 0.5); glVertex3f(0.5,-0.5,-0.5); glVertex3f(0.5, 0.5,-0.5); glVertex3f(0.5, 0.5, 0.5); // Right
        glEnd();
        
        // Wireframe Outline (Neon Style)
        glColor3f(1.0f, 0.5f, 0.0f);
        glLineWidth(2.0f);
        glBegin(GL_LINE_LOOP); // Top Loop
        glVertex3f(-0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, 0.5); glVertex3f(0.5, 0.5, -0.5); glVertex3f(-0.5, 0.5, -0.5);
        glEnd();
        
        glPopMatrix();
    }
}

void draw_weapon_model() {
    glPushMatrix();
    glLoadIdentity();
    
    // Recoil Kick
    float kick = recoil_anim * 0.3f;
    glTranslatef(0.5f, -0.6f + kick, -1.5f + (kick * 0.5f));
    glRotatef(-recoil_anim * 15.0f, 1, 0, 0);

    // Color based on weapon
    if(current_weapon == 1) glColor3f(0.6f, 0.6f, 0.6f); // Magnum
    if(current_weapon == 2) glColor3f(0.2f, 0.8f, 0.2f); // AR
    if(current_weapon == 3) glColor3f(0.6f, 0.4f, 0.2f); // Shotgun
    if(current_weapon == 4) glColor3f(0.2f, 0.2f, 0.2f); // Sniper

    glScalef(0.1f, 0.15f, 0.6f);
    glBegin(GL_QUADS);
        // Simple Gun Box
        glVertex3f(-1, 1, 1); glVertex3f(1, 1, 1); glVertex3f(1, 1, -1); glVertex3f(-1, 1, -1);
        glVertex3f(-1, -1, 1); glVertex3f(1, -1, 1); glVertex3f(1, 1, 1); glVertex3f(-1, 1, 1);
        glVertex3f(-1, -1, -1); glVertex3f(-1, 1, -1); glVertex3f(1, 1, -1); glVertex3f(1, -1, -1);
        glVertex3f(-1, -1, -1); glVertex3f(-1, -1, 1); glVertex3f(-1, 1, 1); glVertex3f(-1, 1, -1);
        glVertex3f(1, -1, 1); glVertex3f(1, -1, -1); glVertex3f(1, 1, -1); glVertex3f(1, 1, 1);
    glEnd();
    glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT // PHYSICS TUNED", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    SDL_SetRelativeMouseMode(SDL_TRUE); // LOCK MOUSE

    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluPerspective(75.0, 1280.0/720.0, 0.1, 1000.0); // Wider FOV (75)
    glMatrixMode(GL_MODELVIEW);
    glEnable(GL_DEPTH_TEST);

    int space_held = 0;

    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_KEYDOWN) {
                if(e.key.keysym.sym == SDLK_ESCAPE) running = 0;
                if(e.key.keysym.sym >= SDLK_1 && e.key.keysym.sym <= SDLK_5) 
                    current_weapon = e.key.keysym.sym - SDLK_0;
                if(e.key.keysym.sym == SDLK_SPACE) space_held = 1;
                if(e.key.keysym.sym == SDLK_LCTRL) p.crouching = 1;
            }
            if(e.type == SDL_KEYUP) {
                if(e.key.keysym.sym == SDLK_SPACE) space_held = 0;
                if(e.key.keysym.sym == SDLK_LCTRL) p.crouching = 0;
            }
            if(e.type == SDL_MOUSEMOTION) {
                p.yaw += e.motion.xrel * MOUSE_SENS;
                p.pitch += e.motion.yrel * MOUSE_SENS;
                if(p.pitch > 89.0f) p.pitch = 89.0f;
                if(p.pitch < -89.0f) p.pitch = -89.0f;
            }
            if(e.type == SDL_MOUSEBUTTONDOWN) recoil_anim = 1.0f;
        }

        // --- ACCELERATION ---
        const Uint8 *s = SDL_GetKeyboardState(NULL);
        float rad = p.yaw * 0.0174533f;
        float cx = sinf(rad);
        float cz = -cosf(rad);
        float sx = cosf(rad); // Strafe vector X
        float sz = sinf(rad); // Strafe vector Z

        if(s[SDL_SCANCODE_W]) { p.vx += cx * MOVE_ACCEL; p.vz += cz * MOVE_ACCEL; }
        if(s[SDL_SCANCODE_S]) { p.vx -= cx * MOVE_ACCEL; p.vz -= cz * MOVE_ACCEL; }
        if(s[SDL_SCANCODE_A]) { p.vx -= sx * MOVE_ACCEL; p.vz -= sz * MOVE_ACCEL; }
        if(s[SDL_SCANCODE_D]) { p.vx += sx * MOVE_ACCEL; p.vz += sz * MOVE_ACCEL; }

        // Jump
        if(space_held && p.on_ground) {
             p.vy = JUMP_FORCE;
             p.on_ground = 0;
        }

        update_physics(space_held);
        if(recoil_anim > 0) recoil_anim -= 0.08f;

        // --- RENDER FRAME ---
        glClearColor(0.05f, 0.02f, 0.1f, 1.0f); // Deep Space Purple
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
        glLoadIdentity();

        // Camera Logic (Crouch lowers view)
        float cam_h = p.crouching ? 2.5f : 6.0f; // Tall spartan height
        
        glRotatef(-p.pitch, 1, 0, 0);
        glRotatef(-p.yaw, 0, 1, 0);
        glTranslatef(-p.x, -(p.y + cam_h), -p.z);

        draw_grid();
        draw_boxes();
        draw_weapon_model();

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
