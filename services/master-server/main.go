package main

import (
    "bufio"
    "fmt"
    "net"
    "strings"
    "sync"
    "time"
)

const (
    Port = ":8080"
    MatchSize = 2 // Small for testing, use 4+ for prod
)

type GameServer struct {
    IP, Port, Name, Players, MaxPlayers string
    LastSeen time.Time
}

type Client struct {
    Conn net.Conn
    Name string
}

var (
    servers    = make(map[string]*GameServer)
    clients    = make(map[net.Conn]*Client)
    queue      = make([]*Client, 0)
    mu         sync.RWMutex
    broadcast  = make(chan string)
)

func main() {
    fmt.Println("🌍 SKATECHAIN SOCIAL HUB LISTENING ON " + Port)

    // Broadcaster
    go func() {
        for msg := range broadcast {
            mu.RLock()
            for conn := range clients {
                conn.Write([]byte(msg + "\n"))
            }
            mu.RUnlock()
        }
    }()

    // Matchmaker Loop
    go func() {
        for range time.Tick(2 * time.Second) {
            mu.Lock()
            if len(queue) >= MatchSize {
                // Find a server
                var target *GameServer
                for _, s := range servers { target = s; break } // Pick first available

                if target != nil {
                    fmt.Printf("[MATCH] Grouping %d players -> %s\n", MatchSize, target.Name)
                    // Notify Players
                    msg := fmt.Sprintf("MATCH_FOUND|%s|%s", target.IP, target.Port)
                    for i := 0; i < MatchSize; i++ {
                        client := queue[0]
                        client.Conn.Write([]byte(msg + "\n"))
                        queue = queue[1:] // Pop
                    }
                }
            }
            mu.Unlock()
        }
    }()

    ln, _ := net.Listen("tcp", Port)
    for {
        conn, _ := ln.Accept()
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()
    
    // Register Client
    client := &Client{Conn: conn, Name: "Anon"}
    mu.Lock()
    clients[conn] = client
    mu.Unlock()

    // Cleanup on disconnect
    defer func() {
        mu.Lock()
        delete(clients, conn)
        // Remove from queue if present
        for i, c := range queue {
            if c == client {
                queue = append(queue[:i], queue[i+1:]...)
                break
            }
        }
        mu.Unlock()
    }()

    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.Split(line, "|")
        cmd := parts[0]

        if cmd == "HEARTBEAT" && len(parts) >= 5 {
            // Server Registration (Legacy)
            ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
            mu.Lock()
            servers[ip+":"+parts[1]] = &GameServer{IP: ip, Port: parts[1], Name: parts[2], Players: parts[3], MaxPlayers: parts[4], LastSeen: time.Now()}
            mu.Unlock()

        } else if cmd == "LIST" {
            mu.RLock()
            for _, s := range servers {
                conn.Write([]byte(fmt.Sprintf("SERVER|%s|%s|%s|%s|%s\n", s.IP, s.Port, s.Name, s.Players, s.MaxPlayers)))
            }
            mu.RUnlock()

        } else if cmd == "CHAT" && len(parts) >= 3 {
            // CHAT|Name|Msg
            client.Name = parts[1]
            broadcast <- fmt.Sprintf("MSG|%s|%s", parts[1], parts[2])

        } else if cmd == "QUEUE" && len(parts) >= 2 {
            // QUEUE|Name
            client.Name = parts[1]
            mu.Lock()
            // Check duplicates
            inQueue := false
            for _, c := range queue { if c == client { inQueue = true } }
            if !inQueue {
                queue = append(queue, client)
                conn.Write([]byte("MSG|System|Joined Matchmaking Queue...\n"))
            }
            mu.Unlock()
        }
    }
}
