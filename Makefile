# 🛹 SHANKPIT Master Makefile (CI-safe, stupid-simple)

CC = gcc
CFLAGS = -O2 -Wall -D_REENTRANT
INCLUDES = -Ipackages/common -Ipackages/simulation

LIBS_GL = -lSDL2 -lGL -lGLU -lm

all: setup lobby server

setup:
	mkdir -p bin

# -------- CLIENT / LOBBY --------
lobby:
	@echo "🔨 Building Lobby Client..."
	$(CC) $(CFLAGS) $(INCLUDES) apps/lobby/src/main.c -o bin/shank_lobby $(LIBS_GL)

# -------- GAME SERVER --------
server:
	@echo "🔨 Building Game Server..."
	$(CC) $(CFLAGS) $(INCLUDES) apps/server/src/main.c -o bin/shank_server -lm

# -------- SERVER CONTROL (OPTIONAL, LOCAL ONLY) --------
serverctl:
	@echo "🖥️ Building Server Control (requires ncurses)..."
	$(CC) -O2 apps/server/serverctl.c -o bin/serverctl -lncurses

clean:
	rm -rf bin
