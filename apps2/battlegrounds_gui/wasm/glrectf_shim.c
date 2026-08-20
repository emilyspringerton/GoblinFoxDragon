#include <GL/gl.h>

/* Emscripten's LEGACY_GL_EMULATION covers glBegin/glEnd/glVertex2f/glColor3f/
 * glOrtho/etc but not glRectf specifically -- it's a rarely-used convenience
 * wrapper, not part of the core immediate-mode set most GL-emulation shims
 * bother reimplementing. Trivial in terms of what IS emulated. */
void glRectf(GLfloat x1, GLfloat y1, GLfloat x2, GLfloat y2) {
    glBegin(GL_QUADS);
    glVertex2f(x1, y1);
    glVertex2f(x2, y1);
    glVertex2f(x2, y2);
    glVertex2f(x1, y2);
    glEnd();
}
