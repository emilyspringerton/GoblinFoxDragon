#include <stdio.h>
#include <SDL2/SDL.h>
#include <GL/gl.h>
#include <GL/glu.h>
#include <math.h>

// --- CONSTANTS ---
#define GRAVITY 0.015f
#define GRAVITY_FLOAT 0.008f // Slower descent if space held
#define JUMP_FORCE 0.35f
#define MOVE_ACCEL 0.04f
#define FRICTION 0.90f
#define AIR_FRICTION 0.98f
#define CUBE_COUNT 10

// --- WEAPONS ---
int current_weapon = 1;
float recoil_anim = 0.0f;

// --- PHYSICS STATE ---
typedef struct {
    float x, y, z;      // Position
    float vx, vy, vz;   // Velocity
    float yaw, pitch;   // Look Angles
    int on_ground;
    int crouching;
} Player;

Player p = {0, 5, 10, 0, 0, 0, 0, 0, 0, 0};

// --- MAP GEOMETRY (CUBES) ---
typedef struct { float x, y, z, size; } Cube;
Cube map_cubes[CUBE_COUNT] = {
    {0, 0, 0, 50},       // Floor
    {10, 2, 10, 4},      // Low Box
    {15, 4, 15, 4},      // Med Box
    {20, 7, 10, 4},      // High Box
    {-15, 3, -15, 6},    // Big Platform
    {-25, 6, -10, 4},    // Step 1
    {-30, 9, -5, 4},     // Step 2
    {0, 5, -20, 2},      // Tiny Pillar
    {5, 2, -5, 3},       // Cover
    {-5, 2, 5, 3}        // Cover
};

// --- COLLISION LOGIC (AABB) ---
int check_collision(float new_x, float new_y, float new_z) {
    // Player Hitbox (Approximation)
    float pw = 1.0f; // Player Width
    float ph = p.crouching ? 1.5f : 3.0f; // Player Height (Crouch vs Stand)
    
    // Check against all cubes
    for(int i=0; i<CUBE_COUNT; i++) {
        Cube c = map_cubes[i];
        // Simple AABB check
        if (new_x + pw > c.x - c.size && new_x - pw < c.x + c.size &&
            new_y < c.y + c.size && new_y + ph > c.y - c.size && // Height check
            new_z + pw > c.z - c.size && new_z - pw < c.z + c.size) {
            
            // If we are falling onto it, snap to top
            if (p.vy <= 0 && p.y >= c.y + c.size) {
                p.y = c.y + c.size + 0.01f;
                p.vy = 0;
                p.on_ground = 1;
                // Add tiny momentum boost if jumping off logic triggers here?
                // For now, simple landing.
                return 1; // Handled as ground
            }
            return 2; // Wall Hit
        }
    }
    return 0; // Air
}

// --- RENDER FUNCTIONS ---
void draw_map() {
    // Draw Cubes
    for(int i=0; i<CUBE_COUNT; i++) {
        Cube c = map_cubes[i];
        glPushMatrix();
        glTranslatef(c.x, c.y, c.z);
        glScalef(c.size, c.size, c.size);
        
        glBegin(GL_QUADS);
        // Neon Edges look
        if(i==0) glColor3f(0.1f, 0.1f, 0.2f); // Floor
        else glColor3f(0.8f, 0.3f, 0.1f);     // Red Rock Cubes
        
        // Simple Cube Mesh
        glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1); // Top
        glColor3f(0.5f, 0.2f, 0.1f); // Sides Darker
        glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1); // Front
        glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1); // Back
        glVertex3f(-1,-1,-1); glVertex3f(-1,-1,1); glVertex3f(-1,1,1); glVertex3f(-1,1,-1); // Left
        glVertex3f(1,-1,1); glVertex3f(1,-1,-1); glVertex3f(1,1,-1); glVertex3f(1,1,1); // Right
        glEnd();
        glPopMatrix();
    }
}

void draw_weapon() {
    glPushMatrix();
    glLoadIdentity();
    
    // Weapon Sway + Recoil
    float bob = sinf(SDL_GetTicks() * 0.005f) * 0.02f;
    // Apply Recoil Kick (Back and Up)
    glTranslatef(0.4f, -0.4f + bob + (recoil_anim * 0.1f), -1.2f + (recoil_anim * 0.2f));
    glRotatef(-5.0f - (recoil_anim * 10.0f), 1, 0, 0); // Pitch up on fire

    // Basic Model (Switch Color based on WPN)
    if(current_weapon == 1) glColor3f(0.3f, 0.3f, 0.3f); // Magnum
    if(current_weapon == 2) glColor3f(0.2f, 0.5f, 0.2f); // AR
    if(current_weapon == 3) glColor3f(0.5f, 0.2f, 0.1f); // Shotgun
    if(current_weapon == 4) glColor3f(0.1f, 0.1f, 0.1f); // Sniper
    
    glScalef(0.1f, 0.15f, 0.5f);
    glBegin(GL_QUADS);
    glVertex3f(-1,1,1); glVertex3f(1,1,1); glVertex3f(1,1,-1); glVertex3f(-1,1,-1);
    glVertex3f(-1,-1,1); glVertex3f(1,-1,1); glVertex3f(1,1,1); glVertex3f(-1,1,1);
    glVertex3f(-1,-1,-1); glVertex3f(-1,1,-1); glVertex3f(1,1,-1); glVertex3f(1,-1,-1);
    glEnd();
    
    glPopMatrix();
}

int main(int argc, char* argv[]) {
    SDL_Init(SDL_INIT_VIDEO);
    SDL_Window *win = SDL_CreateWindow("SHANKPIT // HALO PHYSICS", 100, 100, 1280, 720, SDL_WINDOW_OPENGL);
    SDL_GL_CreateContext(win);
    SDL_SetRelativeMouseMode(SDL_TRUE);

    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluPerspective(70.0, 1280.0/720.0, 0.1, 1000.0);
    glMatrixMode(GL_MODELVIEW);
    glEnable(GL_DEPTH_TEST);

    int running = 1;
    int space_held = 0;

    while(running) {
        SDL_Event e;
        while(SDL_PollEvent(&e)) {
            if(e.type == SDL_QUIT) running = 0;
            if(e.type == SDL_MOUSEMOTION) {
                p.yaw += e.motion.xrel * 0.1f;
                p.pitch += e.motion.yrel * 0.1f;
                if(p.pitch > 89.0f) p.pitch = 89.0f;
                if(p.pitch < -89.0f) p.pitch = -89.0f;
            }
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
            if(e.type == SDL_MOUSEBUTTONDOWN) {
                recoil_anim = 1.0f; // Fire!
            }
        }

        // --- PHYSICS UPDATE ---
        const Uint8 *state = SDL_GetKeyboardState(NULL);
        
        // Acceleration (WASD) - Relative to YAW
        float rad = p.yaw * 0.0174533f;
        float cx = sinf(rad);
        float cz = -cosf(rad);
        
        if(state[SDL_SCANCODE_W]) { p.vx += cx * MOVE_ACCEL; p.vz += cz * MOVE_ACCEL; }
        if(state[SDL_SCANCODE_S]) { p.vx -= cx * MOVE_ACCEL; p.vz -= cz * MOVE_ACCEL; }
        if(state[SDL_SCANCODE_A]) { p.vx -= cz * MOVE_ACCEL; p.vz += cx * MOVE_ACCEL; } // Strafe Left
        if(state[SDL_SCANCODE_D]) { p.vx += cz * MOVE_ACCEL; p.vz -= cx * MOVE_ACCEL; } // Strafe Right
        
        // Jump Logic
        if(space_held && p.on_ground) {
             p.vy = JUMP_FORCE;
             p.on_ground = 0;
             // "Extra momentum off cube" - slight forward boost
             p.vx *= 1.1f; p.vz *= 1.1f;
        }

        // Gravity (Variable)
        if(p.vy < 0 && space_held) p.vy -= GRAVITY_FLOAT; // Float down
        else p.vy -= GRAVITY; // Normal gravity

        // Apply Velocity
        p.x += p.vx;
        p.z += p.vz;
        
        // Collision Checks (Wall/Ground)
        if(p.y <= 0) { p.y = 0; p.vy = 0; p.on_ground = 1; } // Floor
        int col = check_collision(p.x, p.y + p.vy, p.z);
        if(col != 2) p.y += p.vy; // Only move Y if not hitting ceiling/floor hard
        
        // Friction
        float f = p.on_ground ? FRICTION : AIR_FRICTION;
        p.vx *= f;
        p.vz *= f;

        // Recoil Decay
        if(recoil_anim > 0) recoil_anim -= 0.1f;

        // --- RENDER ---
        glClearColor(0.1f, 0.15f, 0.2f, 1.0f);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
        glLoadIdentity();

        // Camera Transform
        float cam_h = p.crouching ? 1.5f : 3.0f;
        
        // Look Rotation
        glRotatef(-p.pitch, 1, 0, 0);
        glRotatef(-p.yaw, 0, 1, 0);
        // Position translation (Inverse of player pos)
        glTranslatef(-p.x, -(p.y + cam_h), -p.z);

        draw_map();
        draw_weapon();

        SDL_GL_SwapWindow(win);
        SDL_Delay(16);
    }
    SDL_Quit();
    return 0;
}
