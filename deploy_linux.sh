#!/bin/bash
echo "🚀 DEPLOYING SHANK PIT SERVER..."

# 1. Install Deps
sudo apt-get update
sudo apt-get install -y gcc make libsdl2-dev libgl1-mesa-dev libglu1-mesa-dev screen git golang

# 2. Compile Master
echo "🛠️ Building Master Server..."
cd services/master-server
go build -o ../../bin/master_server .
cd ../..

# 3. Compile Game Server
echo "🛠️ Building Game Server..."
gcc services/game-server/src/server.c -o bin/shank_server -lm

# 4. Config
echo "⚙️ Writing Config..."
cat > bin/config.ini <<EOF
[Network]
MasterIP=127.0.0.1
MasterPort=8080
GamePort=6969
EOF

# 5. Run
echo "🔥 Launching..."
killall master_server
killall shank_server

screen -dmS master ./bin/master_server
screen -dmS game1 ./bin/shank_server

echo "✅ SERVER ONLINE. Use 'screen -r master' to view logs."
