
🛹 HOW TO RUN SHANK PIT

PREREQUISITES:
1. Windows: Install MinGW (GCC) and download SDL2 Development Libraries (MinGW version).
   - Extract SDL2 to C:\SDL2 (or edit build_win.bat).
2. Linux: sudo apt install libsdl2-dev libgl1-mesa-dev libglu1-mesa-dev build-essential

COMPILING:
- Windows: Double click 'build_win.bat'
- Linux/Mac: Open terminal, type 'make'

PLAYING:
1. Open Terminal/CMD.
2. Run Server: ./bin/shank_server (or bin\shank_server.exe)
3. Open New Terminal.
4. Run Client: ./bin/shank_client (or bin\shank_client.exe)

CONTROLS:
- WASD: Move
- Mouse: Look / Shoot
- 1-5: Switch Weapons (Knife, Magnum, AR, Shotgun, Sniper)
- G: Grenade
- E: Enter Vehicle
- TAB: Scoreboard
