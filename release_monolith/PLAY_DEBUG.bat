@echo off
cd /d "%~dp0"
echo STARTING SHANK PIT...
ShankPit.exe
echo.
echo GAME CLOSED. CHECK CRASH_LOG.TXT IF IT CRASHED.
pause
