@echo off
title Shank Pit Builder
color 0A

echo ==========================================
echo 🛹 SHANK PIT BUILD SYSTEM (WINDOWS)
echo ==========================================

REM --- CONFIGURATION ---
REM Change this to where you installed SDL2
set SDL_PATH=C:\SDL2

if not exist "%SDL_PATH%" (
    color 0C
    echo [ERROR] SDL2 not found at %SDL_PATH%
    echo Please install SDL2 Development Libs (MinGW) and edit this file.
    pause
    exit /b
)

if not exist bin mkdir bin

echo.
echo [1/2] Compiling SERVER...
gcc services\game-server\src\server.c -o bin\shank_server.exe -O2 -lws2_32
if %errorlevel% neq 0 (
    color 0C
    echo [FAIL] Server Compilation Failed!
    pause
    exit /b
)
echo    - Success.

echo.
echo [2/2] Compiling CLIENT...
gcc apps\shank-fps\src\main.c -o bin\shank_client.exe -O2 ^
    -I "%SDL_PATH%\include" ^
    -L "%SDL_PATH%\lib" ^
    -lmingw32 -lSDL2main -lSDL2 -lopengl32 -lglu32 -lws2_32
if %errorlevel% neq 0 (
    color 0C
    echo [FAIL] Client Compilation Failed!
    pause
    exit /b
)
echo    - Success.

echo.
echo ==========================================
echo ✅ BUILD COMPLETE. 
echo Run bin\shank_server.exe then bin\shank_client.exe
echo ==========================================
pause
