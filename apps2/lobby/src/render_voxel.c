#include "render_voxel.h"
#include "shim_gles.h"

#define TEX_LOG 4
#define TEX_LEAVES 5

static void draw_cube_faces(void) {
    shim_glBegin(GL_QUADS);
    // Front
    glTexCoord2f(0, 0); shim_glVertex3f(0, 0, 1);
    glTexCoord2f(1, 0); shim_glVertex3f(1, 0, 1);
    glTexCoord2f(1, 1); shim_glVertex3f(1, 1, 1);
    glTexCoord2f(0, 1); shim_glVertex3f(0, 1, 1);
    // Back
    glTexCoord2f(0, 0); shim_glVertex3f(1, 0, 0);
    glTexCoord2f(1, 0); shim_glVertex3f(0, 0, 0);
    glTexCoord2f(1, 1); shim_glVertex3f(0, 1, 0);
    glTexCoord2f(0, 1); shim_glVertex3f(1, 1, 0);
    // Left
    glTexCoord2f(0, 0); shim_glVertex3f(0, 0, 0);
    glTexCoord2f(1, 0); shim_glVertex3f(0, 0, 1);
    glTexCoord2f(1, 1); shim_glVertex3f(0, 1, 1);
    glTexCoord2f(0, 1); shim_glVertex3f(0, 1, 0);
    // Right
    glTexCoord2f(0, 0); shim_glVertex3f(1, 0, 1);
    glTexCoord2f(1, 0); shim_glVertex3f(1, 0, 0);
    glTexCoord2f(1, 1); shim_glVertex3f(1, 1, 0);
    glTexCoord2f(0, 1); shim_glVertex3f(1, 1, 1);
    // Top
    glTexCoord2f(0, 0); shim_glVertex3f(0, 1, 1);
    glTexCoord2f(1, 0); shim_glVertex3f(1, 1, 1);
    glTexCoord2f(1, 1); shim_glVertex3f(1, 1, 0);
    glTexCoord2f(0, 1); shim_glVertex3f(0, 1, 0);
    // Bottom
    glTexCoord2f(0, 0); shim_glVertex3f(0, 0, 0);
    glTexCoord2f(1, 0); shim_glVertex3f(1, 0, 0);
    glTexCoord2f(1, 1); shim_glVertex3f(1, 0, 1);
    glTexCoord2f(0, 1); shim_glVertex3f(0, 0, 1);
    shim_glEnd();
}

static void draw_minecraft_block(float x, float y, float z, int type) {
    if (type == 17) {
        glBindTexture(GL_TEXTURE_2D, TEX_LOG);
        glDisable(GL_ALPHA_TEST);
    } else if (type == 18) {
        glBindTexture(GL_TEXTURE_2D, TEX_LEAVES);
        glEnable(GL_ALPHA_TEST);
        glAlphaFunc(GL_GREATER, 0.5f);
    } else {
        return;
    }

    glPushMatrix();
    glTranslatef(x, y, z);
    draw_cube_faces();
    glPopMatrix();
}

void render_received_voxels(const NetVoxelPacket *packet) {
    if (!packet) return;

    for (int i = 0; i < packet->block_count; i++) {
        const VoxelBlock *blk = &packet->blocks[i];
        draw_minecraft_block(blk->x, blk->y, blk->z, blk->block_id);
    }
}
