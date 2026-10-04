// Command wsudprelay bridges browser WebSocket connections to the real UDP
// arena/matchmaker protocol REDGARDEN's server-authoritative model uses, so
// GoblinFoxDragon's WASM build (apps2/battlegrounds_gui/wasm) can actually
// play a live match rather than sit on a blank canvas.
//
// Founder real-time, 2026-08-20: "ensure GFD web is live on okemily - link
// it from WOTAN" -> "live demos that are more than just demos." The WASM
// client's own wasm/README.md already names this exact gap: browsers have
// no UDP socket API, and Emscripten's socket emulation links successfully
// but "says nothing about whether [the sockets] actually work at runtime
// without a real WebSocket-to-UDP proxy server in front of the arena/
// matchmaker backends."
//
// How this works, and why it needs zero changes to the WASM client:
// Emscripten's default socket emulation (no Module.websocket.url override,
// which this build doesn't set) opens a WebSocket to the exact same
// "ws://{host}:{port}" the C code's connect()/sendto() targeted -- so this
// relay listens on ONE WebSocket (TCP) listener PER port the real UDP
// backend uses, on the SAME port numbers, and blind-forwards each frame:
// one incoming WS binary message -> one outgoing UDP datagram to that same
// port on 127.0.0.1, and vice versa. The client and the real REDGARDEN
// binaries never know a relay is involved.
//
// Port range covers the redgarden-stable deployment this relay fronts
// (see REDGARDEN/CLAUDE.md's Deployments table): the matchmaker itself
// (8778) plus a bounded range of dynamically-assigned game-server ports
// starting at --first-game-port (8300 for redgarden-stable's arena pool).
// The range is generous, not exhaustive -- sized for a public web demo's
// realistic concurrent-match count, not the internal bot pool's own load.
//
// Usage:
//
//	go run ./apps2/wsudprelay --udp-host 127.0.0.1 --matchmaker-port 8778 \
//	    --game-port-start 8300 --game-port-count 100
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/websocket"
)

// wsToUDP returns the WebSocket handler that bridges one browser connection to udpAddr.
func wsToUDP(port int, udpAddr *net.UDPAddr) websocket.Handler {
	return func(ws *websocket.Conn) {
		ws.PayloadType = websocket.BinaryFrame

		udpConn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			log.Printf("wsudprelay[%d]: dial UDP: %v", port, err)
			return
		}
		defer udpConn.Close()

		done := make(chan struct{})

		// WS -> UDP: each browser-side sendto() arrives as one binary
		// WebSocket message; forward it as one datagram, preserving the
		// message-per-packet framing the real UDP protocol expects.
		go func() {
			defer close(done)
			buf := make([]byte, 65536)
			for {
				n, err := ws.Read(buf)
				if err != nil {
					return
				}
				if _, err := udpConn.Write(buf[:n]); err != nil {
					return
				}
			}
		}()

		// UDP -> WS: each datagram from the real arena/matchmaker server
		// becomes one binary WebSocket message back to the browser.
		buf := make([]byte, 65536)
		for {
			n, err := udpConn.Read(buf)
			if err != nil {
				break
			}
			if _, err := ws.Write(buf[:n]); err != nil {
				break
			}
		}
		<-done
	}
}

func relayPort(port int, udpHost string) {
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(udpHost, strconv.Itoa(port)))
	if err != nil {
		log.Fatalf("wsudprelay: resolve UDP target %s:%d: %v", udpHost, port, err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", wsToUDP(port, udpAddr))
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(port))
	log.Printf("wsudprelay: listening ws://%s -> udp %s:%d", addr, udpHost, port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("wsudprelay[%d]: listener died: %v", port, err)
	}
}

// servePathMode is the Kubernetes shape (2026-10-04): ONE HTTP listener behind the GCE Ingress
// (TLS from Certificate Manager), the target UDP port carried in the path -- /gfd-ws/<port> --
// instead of one TCP listener per port. Only the matchmaker port and the configured game-port
// range are accepted, so this is not an open UDP proxy. "/" answers 200 for the LB health check.
func servePathMode(listen int, udpHost string, mmPort, gameStart, gameCount int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("wsudprelay ok\n"))
	})
	mux.HandleFunc("/gfd-ws/", func(w http.ResponseWriter, r *http.Request) {
		port, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/gfd-ws/"))
		ok := err == nil && ((mmPort != 0 && port == mmPort) || (port >= gameStart && port < gameStart+gameCount))
		if !ok {
			http.NotFound(w, r)
			return
		}
		udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(udpHost, strconv.Itoa(port)))
		if err != nil {
			http.Error(w, "backend unresolved", http.StatusBadGateway)
			return
		}
		wsToUDP(port, udpAddr).ServeHTTP(w, r)
	})
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(listen))
	log.Printf("wsudprelay: path mode on %s -> udp %s (matchmaker=%d, game ports %d-%d)", addr, udpHost, mmPort, gameStart, gameStart+gameCount-1)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func main() {
	udpHost := flag.String("udp-host", "127.0.0.1", "host the real UDP arena/matchmaker backends listen on")
	matchmakerPort := flag.Int("matchmaker-port", 8778, "matchmaker's UDP port to relay (0 to skip)")
	gamePortStart := flag.Int("game-port-start", 8300, "first dynamically-assigned game-server port to relay")
	gamePortCount := flag.Int("game-port-count", 100, "how many consecutive game-server ports to relay")
	pathPort := flag.Int("path-mode-port", 0, "if set: serve ONE listener on this port, routing /gfd-ws/<udp-port> (Kubernetes/Ingress shape)")
	flag.Parse()

	if *pathPort != 0 {
		servePathMode(*pathPort, *udpHost, *matchmakerPort, *gamePortStart, *gamePortCount)
		return
	}

	if *matchmakerPort != 0 {
		go relayPort(*matchmakerPort, *udpHost)
	}
	for p := *gamePortStart; p < *gamePortStart+*gamePortCount; p++ {
		go relayPort(p, *udpHost)
	}

	log.Printf("wsudprelay: %d listeners up (matchmaker=%d, game ports %d-%d)",
		*gamePortCount+1, *matchmakerPort, *gamePortStart, *gamePortStart+*gamePortCount-1)
	select {}
}
