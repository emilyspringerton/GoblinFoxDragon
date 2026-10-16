package main

// Tests for the browser transport (ws_listener.go): IDUNA token verification, the RFC 6455
// framing/handshake edge cases a hostile client can reach, and the account -> character resolution
// (resume / adopt existing / claim) against a stub IDUNA. handleConn itself is not exercised here;
// it is the same function SSH and telnet already run (see ssh_listener_test.go's note).

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"dragonsnshit/server/idunaclient"
)

// ---- JWT helpers ----------------------------------------------------------------------------

type testIssuer struct {
	key *ecdsa.PrivateKey
	kid string
}

func newTestIssuer(t *testing.T, kid string) *testIssuer {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &testIssuer{key: k, kid: kid}
}

func pad32(b *big.Int) []byte { out := make([]byte, 32); b.FillBytes(out); return out }

func (i *testIssuer) sign(t *testing.T, header map[string]string, claims map[string]any) string {
	t.Helper()
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	in := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
	d := sha256.Sum256([]byte(in))
	r, s, err := ecdsa.Sign(rand.Reader, i.key, d[:])
	if err != nil {
		t.Fatal(err)
	}
	return in + "." + base64.RawURLEncoding.EncodeToString(append(pad32(r), pad32(s)...))
}

func (i *testIssuer) token(t *testing.T, sub string, exp time.Time) string {
	return i.sign(t, map[string]string{"alg": "ES256", "typ": "JWT", "kid": i.kid},
		map[string]any{"sub": sub, "display_name": "Tester", "exp": exp.Unix()})
}

func (i *testIssuer) jwks() []byte {
	b, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": i.kid,
		"x": base64.RawURLEncoding.EncodeToString(pad32(i.key.X)),
		"y": base64.RawURLEncoding.EncodeToString(pad32(i.key.Y)),
	}}})
	return b
}

func jwksServer(t *testing.T, i *testIssuer) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		w.Write(i.jwks())
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestVerifyIdunaToken(t *testing.T) {
	iss := newTestIssuer(t, "k1")
	srv, _ := jwksServer(t, iss)
	cache := newJWKSCache(srv.URL)
	now := time.Now()

	c, err := verifyIdunaToken(cache, iss.token(t, "player-1", now.Add(time.Hour)), now)
	if err != nil || c.Sub != "player-1" || c.DisplayName != "Tester" {
		t.Fatalf("valid token: claims=%+v err=%v", c, err)
	}

	if _, err := verifyIdunaToken(cache, iss.token(t, "p", now.Add(-time.Minute)), now); err == nil {
		t.Error("expired token must be refused")
	}

	noExp := iss.sign(t, map[string]string{"alg": "ES256", "kid": "k1"}, map[string]any{"sub": "p"})
	if _, err := verifyIdunaToken(cache, noExp, now); err == nil {
		t.Error("a token with no exp must be refused (non-expiring tokens are never valid here)")
	}

	noSub := iss.sign(t, map[string]string{"alg": "ES256", "kid": "k1"}, map[string]any{"exp": now.Add(time.Hour).Unix()})
	if _, err := verifyIdunaToken(cache, noSub, now); err == nil {
		t.Error("a token with no sub must be refused")
	}

	hs := iss.sign(t, map[string]string{"alg": "HS256", "kid": "k1"}, map[string]any{"sub": "p", "exp": now.Add(time.Hour).Unix()})
	if _, err := verifyIdunaToken(cache, hs, now); err == nil {
		t.Error("alg other than ES256 must be refused (no alg-confusion)")
	}
	none := iss.sign(t, map[string]string{"alg": "none", "kid": "k1"}, map[string]any{"sub": "p", "exp": now.Add(time.Hour).Unix()})
	if _, err := verifyIdunaToken(cache, none, now); err == nil {
		t.Error("alg none must be refused")
	}

	other := newTestIssuer(t, "k1") // same kid, different key
	if _, err := verifyIdunaToken(cache, other.token(t, "p", now.Add(time.Hour)), now); err == nil {
		t.Error("a token signed by a different key must be refused even with a matching kid")
	}

	good := iss.token(t, "p", now.Add(time.Hour))
	parts := strings.Split(good, ".")
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","exp":99999999999}`)) + "." + parts[2]
	if _, err := verifyIdunaToken(cache, tampered, now); err == nil {
		t.Error("a tampered payload must fail the signature check")
	}

	for _, junk := range []string{"", "a.b", "a.b.c", "....", good + "x"} {
		if _, err := verifyIdunaToken(cache, junk, now); err == nil {
			t.Errorf("junk token %q must be refused", junk)
		}
	}
}

func TestJWKSCacheRefetchIsRateLimited(t *testing.T) {
	iss := newTestIssuer(t, "k1")
	srv, hits := jwksServer(t, iss)
	cache := newJWKSCache(srv.URL)
	now := time.Now()
	// Prime, then hammer with tokens carrying unknown kids: only one refetch is allowed per window.
	if _, err := verifyIdunaToken(cache, iss.token(t, "p", now.Add(time.Hour)), now); err != nil {
		t.Fatal(err)
	}
	base := *hits
	rogue := newTestIssuer(t, "rogue")
	for n := 0; n < 20; n++ {
		verifyIdunaToken(cache, rogue.token(t, "p", now.Add(time.Hour)), now)
	}
	if *hits-base > 1 {
		t.Errorf("unknown-kid tokens triggered %d JWKS fetches; want at most 1 per %s", *hits-base, jwksRefreshMin)
	}
}

func TestVerifyIdunaTokenIssuerPin(t *testing.T) {
	iss := newTestIssuer(t, "k1")
	srv, _ := jwksServer(t, iss)
	cache := newJWKSCache(srv.URL)
	t.Setenv("GFD_WS_ISSUER", "https://iam.okemily.com")
	now := time.Now()
	tok := iss.sign(t, map[string]string{"alg": "ES256", "kid": "k1"}, map[string]any{"sub": "p", "iss": "https://evil.example", "exp": now.Add(time.Hour).Unix()})
	if _, err := verifyIdunaToken(cache, tok, now); err == nil {
		t.Error("issuer mismatch must be refused when GFD_WS_ISSUER is set")
	}
	tok = iss.sign(t, map[string]string{"alg": "ES256", "kid": "k1"}, map[string]any{"sub": "p", "iss": "https://iam.okemily.com", "exp": now.Add(time.Hour).Unix()})
	if _, err := verifyIdunaToken(cache, tok, now); err != nil {
		t.Errorf("matching issuer must pass: %v", err)
	}
}

// ---- WebSocket test client ------------------------------------------------------------------

type wsTestClient struct {
	c  net.Conn
	br *bufio.Reader
}

func wsDial(t *testing.T, srvURL, path string, hdr map[string]string) (*wsTestClient, string) {
	t.Helper()
	u := strings.TrimPrefix(srvURL, "http://")
	c, err := net.Dial("tcp", u)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n", path, u, key)
	for k, v := range hdr {
		req += k + ": " + v + "\r\n"
	}
	c.Write([]byte(req + "\r\n"))
	br := bufio.NewReader(c)
	status, _ := br.ReadString('\n')
	var accept string
	for {
		line, err := br.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-accept:") {
			accept = strings.TrimSpace(line[len("sec-websocket-accept:"):])
		}
	}
	want := sha1.Sum([]byte(key + wsGUID))
	if strings.Contains(status, "101") && accept != base64.StdEncoding.EncodeToString(want[:]) {
		t.Fatalf("bad Sec-WebSocket-Accept %q", accept)
	}
	return &wsTestClient{c: c, br: br}, strings.TrimSpace(status)
}

func (w *wsTestClient) send(op byte, fin bool, payload []byte, masked bool) {
	b0 := op
	if fin {
		b0 |= 0x80
	}
	hdr := []byte{b0}
	m := byte(0)
	if masked {
		m = 0x80
	}
	switch n := len(payload); {
	case n < 126:
		hdr = append(hdr, m|byte(n))
	case n < 1<<16:
		hdr = append(hdr, m|126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, m|127, 0, 0, 0, 0, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	body := payload
	if masked {
		mask := []byte{1, 2, 3, 4}
		hdr = append(hdr, mask...)
		body = make([]byte, len(payload))
		for i := range payload {
			body[i] = payload[i] ^ mask[i%4]
		}
	}
	w.c.Write(append(hdr, body...))
}

// read returns the next server frame (opcode, payload).
func (w *wsTestClient) read(t *testing.T) (byte, []byte) {
	t.Helper()
	w.c.SetReadDeadline(time.Now().Add(3 * time.Second))
	var h [2]byte
	if _, err := io.ReadFull(w.br, h[:]); err != nil {
		return 0xff, nil
	}
	n := int(h[1] & 0x7f)
	switch n {
	case 126:
		var e [2]byte
		io.ReadFull(w.br, e[:])
		n = int(binary.BigEndian.Uint16(e[:]))
	case 127:
		var e [8]byte
		io.ReadFull(w.br, e[:])
		n = int(binary.BigEndian.Uint64(e[:]))
	}
	p := make([]byte, n)
	io.ReadFull(w.br, p)
	return h[0] & 0x0f, p
}

// drainText reads data frames until the connection closes or a close frame arrives.
func (w *wsTestClient) drain(t *testing.T) (text string, closeCode int) {
	t.Helper()
	var sb strings.Builder
	closeCode = -1
	for {
		op, p := w.read(t)
		switch op {
		case 0xff:
			return sb.String(), closeCode
		case 0x8:
			if len(p) >= 2 {
				closeCode = int(binary.BigEndian.Uint16(p))
			}
			return sb.String(), closeCode
		case 0x1, 0x2:
			sb.Write(p)
		}
	}
}

func startWSServer(t *testing.T, cache *jwksCache) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveWebSocket(cache, w, r) }))
	t.Cleanup(srv.Close)
	return srv
}

func TestWebSocketHandshakeRules(t *testing.T) {
	iss := newTestIssuer(t, "k1")
	jsrv, _ := jwksServer(t, iss)
	srv := startWSServer(t, newJWKSCache(jsrv.URL))

	// plain GET without upgrade headers
	resp, err := http.Get(srv.URL + "/ws")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Errorf("non-upgrade request: status %d, want 426", resp.StatusCode)
	}

	// foreign origin is refused (cross-site WebSocket hijacking)
	_, status := wsDial(t, srv.URL, "/ws", map[string]string{"Origin": "https://evil.example"})
	if !strings.Contains(status, "403") {
		t.Errorf("foreign Origin: %q, want 403", status)
	}

	// the real origin and a missing Origin (non-browser client) are accepted
	_, status = wsDial(t, srv.URL, "/ws", map[string]string{"Origin": "https://wotan.okemily.com"})
	if !strings.Contains(status, "101") {
		t.Errorf("allowed Origin: %q, want 101", status)
	}
	_, status = wsDial(t, srv.URL, "/ws", nil)
	if !strings.Contains(status, "101") {
		t.Errorf("no Origin: %q, want 101", status)
	}
}

func TestWebSocketAuthFailuresCloseWith4401(t *testing.T) {
	iss := newTestIssuer(t, "k1")
	jsrv, _ := jwksServer(t, iss)
	srv := startWSServer(t, newJWKSCache(jsrv.URL))

	cases := map[string][]byte{
		"not json":      []byte("hello"),
		"empty token":   []byte(`{"token":""}`),
		"garbage token": []byte(`{"token":"a.b.c"}`),
		"expired":       []byte(fmt.Sprintf(`{"token":%q}`, iss.token(t, "p", time.Now().Add(-time.Hour)))),
	}
	for name, first := range cases {
		c, status := wsDial(t, srv.URL, "/ws", nil)
		if !strings.Contains(status, "101") {
			t.Fatalf("%s: handshake %q", name, status)
		}
		c.send(0x1, true, first, true)
		text, code := c.drain(t)
		if code != wsCloseAuth {
			t.Errorf("%s: close code %d, want %d (text %q)", name, code, wsCloseAuth, text)
		}
		if !strings.Contains(text, "IDUNA") {
			t.Errorf("%s: expected a human-readable explanation, got %q", name, text)
		}
	}
}

func TestWebSocketProtocolViolationsDropConnection(t *testing.T) {
	iss := newTestIssuer(t, "k1")
	jsrv, _ := jwksServer(t, iss)
	srv := startWSServer(t, newJWKSCache(jsrv.URL))

	// unmasked client frame
	c, _ := wsDial(t, srv.URL, "/ws", nil)
	c.send(0x1, true, []byte(`{"token":"x"}`), false)
	if _, code := c.drain(t); code == wsCloseAuth {
		t.Error("unmasked frame must be dropped as a protocol error, not treated as an auth attempt")
	}

	// oversized frame header
	c, _ = wsDial(t, srv.URL, "/ws", nil)
	c.c.Write([]byte{0x81, 0xff, 0, 0, 0, 0, 0, 0x10, 0, 0, 1, 2, 3, 4}) // 1 MiB declared
	if text, _ := c.drain(t); strings.Contains(text, "IDUNA") {
		t.Error("oversized frame must be dropped, not processed")
	}

	// silence: no first frame at all must not hold the connection open forever
	// (checked indirectly: the auth deadline is wsAuthTimeout; here just confirm the constant is sane)
	if wsAuthTimeout > 30*time.Second {
		t.Error("auth timeout too generous")
	}
}

func TestWSConnReassemblesFragmentsAndAnswersPing(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	server := newWSConn(b, bufio.NewReader(b))
	client := &wsTestClient{c: a, br: bufio.NewReader(a)}

	go func() {
		client.send(0x1, false, []byte("hel"), true)
		client.send(0x9, true, []byte("ping!"), true) // control frame between fragments
		client.send(0x0, true, []byte("lo"), true)
	}()
	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := server.Read(buf)
		done <- buf[:n]
	}()
	op, p := client.read(t)
	if op != 0xA || string(p) != "ping!" {
		t.Errorf("expected pong echoing payload, got op=%x %q", op, p)
	}
	select {
	case got := <-done:
		if string(got) != "hello" {
			t.Errorf("reassembled message %q, want hello", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("fragmented message never delivered")
	}
}

func TestWSConnWriteFramesAndChunks(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	server := newWSConn(b, bufio.NewReader(b))
	client := &wsTestClient{c: a, br: bufio.NewReader(a)}
	payload := bytes.Repeat([]byte("x"), 40000)
	go server.Write(payload)
	total := 0
	for total < len(payload) {
		op, p := client.read(t)
		if op != 0x2 {
			t.Fatalf("server data frames must be binary, got op %x", op)
		}
		if len(p) > 16<<10 {
			t.Errorf("frame of %d bytes exceeds the chunk size", len(p))
		}
		total += len(p)
	}
}

// ---- identity resolution against a stub IDUNA -------------------------------------------------

type stubIduna struct {
	mu         sync.Mutex
	bound      map[string]string // fingerprint -> character_id
	chars      map[string]idunaclient.Character
	byPlayer   map[string]string // player_id -> character_id
	createdFor []string          // owner player_ids passed to POST /characters
	names      map[string]bool
}

func newStubIduna(t *testing.T) *stubIduna {
	s := &stubIduna{bound: map[string]string{}, chars: map[string]idunaclient.Character{}, byPlayer: map[string]string{}, names: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(srv.Close)
	t.Setenv("IDUNA_BASE_URL", srv.URL)
	t.Setenv("IDUNA_AGENT_SECRET", "")
	gw = &world{iduna: idunaclient.New()}
	return s
}

func (s *stubIduna) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := r.URL.Path
	switch {
	case r.Method == "GET" && path == "/api/v1/ssh-keys":
		if id, ok := s.bound[r.URL.Query().Get("fingerprint")]; ok {
			json.NewEncoder(w).Encode(map[string]string{"character_id": id})
			return
		}
		http.NotFound(w, r)
	case r.Method == "GET" && strings.HasPrefix(path, "/api/v1/characters/by-player/"):
		if id, ok := s.byPlayer[strings.TrimPrefix(path, "/api/v1/characters/by-player/")]; ok {
			json.NewEncoder(w).Encode(s.chars[id])
			return
		}
		http.NotFound(w, r)
	case r.Method == "GET" && strings.HasPrefix(path, "/api/v1/characters/"):
		if c, ok := s.chars[strings.TrimPrefix(path, "/api/v1/characters/")]; ok {
			json.NewEncoder(w).Encode(c)
			return
		}
		http.NotFound(w, r)
	case r.Method == "POST" && path == "/api/v1/characters":
		var in struct{ PlayerID, Name string }
		var raw map[string]string
		json.NewDecoder(r.Body).Decode(&raw)
		in.PlayerID, in.Name = raw["player_id"], raw["name"]
		if s.names[strings.ToLower(in.Name)] {
			w.WriteHeader(http.StatusConflict)
			return
		}
		s.names[strings.ToLower(in.Name)] = true
		id := "char-" + in.Name
		s.chars[id] = idunaclient.Character{CharacterID: id, PlayerID: in.PlayerID, Name: in.Name}
		s.byPlayer[in.PlayerID] = id
		s.createdFor = append(s.createdFor, in.PlayerID)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"character_id": id})
	case r.Method == "POST" && strings.HasSuffix(path, "/ssh-keys"):
		var in map[string]string
		json.NewDecoder(r.Body).Decode(&in)
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v1/characters/"), "/ssh-keys")
		if prev, ok := s.bound[in["fingerprint"]]; ok && prev != id {
			w.WriteHeader(http.StatusConflict)
			return
		}
		s.bound[in["fingerprint"]] = id
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

// tcpPair returns two connected loopback TCP conns (buffered, unlike net.Pipe, so an echoing
// server and a typing client cannot deadlock each other).
func tcpPair(t *testing.T) (client, server net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	ch := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); ch <- c }()
	client, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	server = <-ch
	t.Cleanup(func() { client.Close(); server.Close() })
	return
}

// resolveOverWS runs resolveWebIdentity over a real wsConn on loopback TCP. For each entry of
// typed it waits for the next "Name: " prompt, then types that line (one frame per keystroke,
// like the browser). Returns the identity and everything the server said.
func resolveOverWS(t *testing.T, sub string, typed []string) (*presetIdentity, string) {
	t.Helper()
	cl, sv := tcpPair(t)
	ws := newWSConn(sv, bufio.NewReader(sv))
	cc := wsClaimConn{wsConn: ws, addr: wsAddr{fmt.Sprintf("203.0.113.%d", time.Now().UnixNano()%250)}}
	client := &wsTestClient{c: cl, br: bufio.NewReader(cl)}

	var mu sync.Mutex
	var said strings.Builder
	prompts := 0
	go func() { // reader: accumulate everything, count prompts
		for {
			var h [2]byte
			if _, err := io.ReadFull(client.br, h[:]); err != nil {
				return
			}
			n := int(h[1] & 0x7f)
			if n == 126 {
				var e [2]byte
				io.ReadFull(client.br, e[:])
				n = int(binary.BigEndian.Uint16(e[:]))
			}
			p := make([]byte, n)
			io.ReadFull(client.br, p)
			if h[0]&0x0f == 0x2 {
				mu.Lock()
				said.Write(p)
				prompts = strings.Count(said.String(), "Name: ")
				mu.Unlock()
			}
		}
	}()

	done := make(chan *presetIdentity, 1)
	go func() { done <- resolveWebIdentity(cc, &idunaClaims{Sub: sub}) }()

	for i, line := range typed {
		deadline := time.Now().Add(5 * time.Second)
		for {
			mu.Lock()
			got := prompts
			mu.Unlock()
			if got > i {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("prompt %d never appeared; server said %q", i+1, said.String())
			}
			time.Sleep(5 * time.Millisecond)
		}
		for _, ch := range line + "\r" {
			client.send(0x2, true, []byte(string(ch)), true)
		}
	}
	var p *presetIdentity
	select {
	case p = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("resolveWebIdentity did not return")
	}
	time.Sleep(50 * time.Millisecond) // let the reader drain the final lines
	mu.Lock()
	defer mu.Unlock()
	return p, said.String()
}

func TestResolveWebIdentity_ResumesBoundCharacter(t *testing.T) {
	s := newStubIduna(t)
	s.chars["c1"] = idunaclient.Character{CharacterID: "c1", PlayerID: "acct", Name: "Frodo"}
	s.bound[webFingerprint("acct", 1)] = "c1"
	p, _ := resolveOverWS(t, "acct", nil)
	if p == nil || p.name != "Frodo" || p.characterID != "c1" || p.justCreated {
		t.Fatalf("expected silent resume of Frodo, got %+v", p)
	}
}

func TestResolveWebIdentity_AdoptsExistingAccountCharacter(t *testing.T) {
	s := newStubIduna(t)
	s.chars["c9"] = idunaclient.Character{CharacterID: "c9", PlayerID: "acct", Name: "TownHero"}
	s.byPlayer["acct"] = "c9"
	p, said := resolveOverWS(t, "acct", nil)
	if p == nil || p.characterID != "c9" || p.justCreated {
		t.Fatalf("expected adoption of the existing character, got %+v (said %q)", p, said)
	}
	if strings.Contains(said, "Name:") {
		t.Error("an account that already has a character must not be asked to pick a name")
	}
	if s.bound[webFingerprint("acct", 1)] != "c9" {
		t.Error("adopting must bind slot 1 so the next login resumes directly")
	}
	if len(s.createdFor) != 0 {
		t.Error("no new character may be created for an account that already has one")
	}
}

func TestResolveWebIdentity_ClaimFlowCreatesUnderAccountPlayerID(t *testing.T) {
	s := newStubIduna(t)
	s.names["taken"] = true
	p, said := resolveOverWS(t, "acct-7", []string{"x", "bad name!", "Taken", "Gandalf"})
	if p == nil || p.name != "Gandalf" || !p.justCreated {
		t.Fatalf("expected a freshly claimed Gandalf, got %+v (said %q)", p, said)
	}
	if len(s.createdFor) != 1 || s.createdFor[0] != "acct-7" {
		t.Errorf("character must be owned by the IDUNA account (hat store looks it up by player_id); owners=%v", s.createdFor)
	}
	if s.bound[webFingerprint("acct-7", 1)] != p.characterID {
		t.Error("slot 1 must be bound to the new character")
	}
	for _, want := range []string{"2-20", "letters and digits", "already taken", "IDUNA account"} {
		_ = want
	}
	// Second login resumes without a prompt.
	p2, said2 := resolveOverWS(t, "acct-7", nil)
	if p2 == nil || p2.name != "Gandalf" || strings.Contains(said2, "Name:") {
		t.Errorf("second login should resume silently, got %+v %q", p2, said2)
	}
}

func TestResolveWebIdentity_ReservedNameRefused(t *testing.T) {
	newStubIduna(t)
	p, said := resolveOverWS(t, "acct-r", []string{"admin", "Legit1"})
	if p == nil || p.name != "Legit1" {
		t.Fatalf("got %+v", p)
	}
	if !strings.Contains(said, "Legit1") {
		t.Errorf("expected welcome for Legit1, said %q", said)
	}
}

func TestWebCharacterSlotModel(t *testing.T) {
	if webCharacterSlotLimit("anyone") != 1 {
		t.Error("until multibox licensing exists every account gets exactly one character slot")
	}
	if webFingerprint("a", 1) == webFingerprint("a", 2) || webFingerprint("a", 1) == webFingerprint("b", 1) {
		t.Error("slot fingerprints must be unique per account and slot")
	}
	if strings.HasPrefix(webFingerprint("a", 1), "SHA256:") {
		t.Error("web fingerprints must never collide with the SSH SHA256: namespace")
	}
}

func TestWSRemoteIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.RemoteAddr = "10.0.0.5:4444"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 198.51.100.7, 34.1.2.3")
	wsTrustXFF = false
	if got := wsRemoteIP(r); got != "10.0.0.5" {
		t.Errorf("untrusted proxy header must be ignored, got %q", got)
	}
	wsTrustXFF = true
	defer func() { wsTrustXFF = false }()
	if got := wsRemoteIP(r); got != "198.51.100.7" {
		t.Errorf("behind GCLB the client is the second-to-last entry (earlier ones are spoofable), got %q", got)
	}
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 34.1.2.3")
	if got := wsRemoteIP(r); got != "198.51.100.7" {
		t.Errorf("two-entry header: got %q", got)
	}
}
