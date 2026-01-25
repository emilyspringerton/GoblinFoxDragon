#ifndef FONT_H
#define FONT_H

#include <SDL2/SDL_opengl.h>
#include <string.h>
#include <ctype.h>

void draw_char(char c, float x, float y, float s) {
    glPushMatrix();
    glTranslatef(x, y, 0);
    
    // FIX: Flip Y axis (-s) so text draws upright in Top-Left coord system
    glScalef(s, -s, 1.0f); 
    
    glLineWidth(3.0f); 
    glBegin(GL_LINES);
    c = toupper(c);
    
    // (Vector Data)
    if(c=='A'){ glVertex2f(0,0); glVertex2f(0.5,1); glVertex2f(0.5,1); glVertex2f(1,0); glVertex2f(0.2,0.4); glVertex2f(0.8,0.4); }
    if(c=='B'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0.8,0.8); glVertex2f(0.8,0.8); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(0.8,0.2); glVertex2f(0.8,0.2); glVertex2f(0,0); }
    if(c=='C'){ glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='D'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0.8,0.5); glVertex2f(0.8,0.5); glVertex2f(0,0); }
    if(c=='E'){ glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); glVertex2f(0,0.5); glVertex2f(0.8,0.5); }
    if(c=='F'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(0,0.5); glVertex2f(0.8,0.5); }
    if(c=='G'){ glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(0.5,0.5); }
    if(c=='H'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(1,0); glVertex2f(1,1); glVertex2f(0,0.5); glVertex2f(1,0.5); }
    if(c=='I'){ glVertex2f(0.5,0); glVertex2f(0.5,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='J'){ glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(0,0.3); }
    if(c=='K'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,0.5); glVertex2f(1,1); glVertex2f(0,0.5); glVertex2f(1,0); }
    if(c=='L'){ glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='M'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0.5,0.5); glVertex2f(0.5,0.5); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); }
    if(c=='N'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,1); }
    if(c=='O'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); }
    if(c=='P'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(0,0.5); }
    if(c=='Q'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0.5,0.5); glVertex2f(1,0); }
    if(c=='R'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(1,0); }
    if(c=='S'){ glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); }
    if(c=='T'){ glVertex2f(0.5,0); glVertex2f(0.5,1); glVertex2f(0,1); glVertex2f(1,1); }
    if(c=='U'){ glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,1); }
    if(c=='V'){ glVertex2f(0,1); glVertex2f(0.5,0); glVertex2f(0.5,0); glVertex2f(1,1); }
    if(c=='W'){ glVertex2f(0,1); glVertex2f(0.2,0); glVertex2f(0.2,0); glVertex2f(0.5,0.5); glVertex2f(0.5,0.5); glVertex2f(0.8,0); glVertex2f(0.8,0); glVertex2f(1,1); }
    if(c=='X'){ glVertex2f(0,0); glVertex2f(1,1); glVertex2f(0,1); glVertex2f(1,0); }
    if(c=='Y'){ glVertex2f(0,1); glVertex2f(0.5,0.5); glVertex2f(1,1); glVertex2f(0.5,0.5); glVertex2f(0.5,0.5); glVertex2f(0.5,0); }
    if(c=='Z'){ glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='0'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,1); }
    if(c=='1'){ glVertex2f(0.5,0); glVertex2f(0.5,1); }
    if(c=='2'){ glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='3'){ glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0,0.5); glVertex2f(1,0.5); }
    if(c=='4'){ glVertex2f(0,1); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(1,0.5); glVertex2f(1,1); glVertex2f(1,0); }
    if(c=='5'){ glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); }
    if(c=='6'){ glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(0,0.5); }
    if(c=='7'){ glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,0); }
    if(c=='8'){ glVertex2f(0,0); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0); glVertex2f(1,0); glVertex2f(0,0); glVertex2f(0,0.5); glVertex2f(1,0.5); }
    if(c=='9'){ glVertex2f(1,0); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(1,0.5); }
    if(c==':'){ glVertex2f(0.5,0.8); glVertex2f(0.5,0.7); glVertex2f(0.5,0.3); glVertex2f(0.5,0.2); }
    if(c=='-'){ glVertex2f(0,0.5); glVertex2f(1,0.5); }
    if(c=='>'){ glVertex2f(0,1); glVertex2f(0.5,0.5); glVertex2f(0.5,0.5); glVertex2f(0,0); }
    if(c=='<'){ glVertex2f(0.5,1); glVertex2f(0,0.5); glVertex2f(0,0.5); glVertex2f(0.5,0); }
    if(c=='['){ glVertex2f(0.5,1); glVertex2f(0,1); glVertex2f(0,1); glVertex2f(0,0); glVertex2f(0,0); glVertex2f(0.5,0); }
    if(c==']'){ glVertex2f(0,1); glVertex2f(0.5,1); glVertex2f(0.5,1); glVertex2f(0.5,0); glVertex2f(0.5,0); glVertex2f(0,0); }
    if(c=='_'){ glVertex2f(0,0); glVertex2f(1,0); }
    if(c=='?'){ glVertex2f(0,1); glVertex2f(1,1); glVertex2f(1,1); glVertex2f(1,0.5); glVertex2f(1,0.5); glVertex2f(0.5,0.5); glVertex2f(0.5,0.5); glVertex2f(0.5,0.2); }

    glEnd();
    glPopMatrix();
}

void draw_text(const char* str, float x, float y, float size, float r, float g, float b) {
    glColor3f(r, g, b);
    float cx = x;
    while(*str) {
        draw_char(*str, cx, y, size);
        cx += size * 1.5f; 
        str++;
    }
}
#endif
