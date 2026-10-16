package main

// Browser transport for the MUD (founder real-time: "once you authenticate and connect you can
// create a name with the same code that ssh used it should all be just the mud over a websocket
// instead of on a real ssh").
//
// Same seam ssh_listener.go uses: handleConn only ever needs a net.Conn, so a WebSocket wrapped
// in wsConn runs the unmodified game loop. Identity differs only in where it comes from -- an
// IDUNA SSO token (ES256, verified offline against IDUNA's public JWKS) instead of an SSH key --
// and is bound to a character through the SAME IDUNA fingerprint table SSH uses, under the
// namespaced fingerprint "web:<sub>#<slot>". An unbound account runs runClaimFlow, the exact
// name-creation code SSH runs.
//
// Characters per account: the binding is slot-based so multibox licensing can raise the limit
// later with no migration. webCharacterSlotLimit is 1 today; there is deliberately no character
// select -- a login resumes slot 1.
//
// RFC 6455 is implemented here directly (server side only, no extensions, no compression)
// because this module has no WebSocket dependency and the surface needed is small.

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dragonsnshit/server/idunaclient"
)

const (
	wsGUID             = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	wsMaxFrame         = 64 << 10 // client frames are keystrokes/paste; anything larger is abuse
	wsAuthTimeout      = 10 * time.Second
	wsPingInterval     = 25 * time.Second // keeps the GKE/L7 idle timeout from cutting quiet sessions
	wsMaxSessionsPerIP = 5
	jwksRefreshMin     = 30 * time.Second
	jwksTTL            = 10 * time.Minute
)

// wsCloseAuth is the application close code for "token missing/invalid/expired"; the browser
// client treats it as "sign in again" rather than "reconnect".
const wsCloseAuth = 4401

// webCharacterSlotLimit is how many characters an IDUNA account may own. Multibox licensing will
// make this per-account; until then it is 1 and no character-select UI exists.
func webCharacterSlotLimit(sub string) int { return 1 }

func webFingerprint(sub string, slot int) string { return fmt.Sprintf("web:%s#%d", sub, slot) }

// ---------------------------------------------------------------------------------------------
// IDUNA token verification (ES256 against the public JWKS)

type jwksCache struct {
	mu      sync.Mutex
	url     string
	keys    map[string]*ecdsa.PublicKey
	fetched time.Time
	tried   time.Time
}

func newJWKSCache(url string) *jwksCache {
	return &jwksCache{url: url, keys: map[string]*ecdsa.PublicKey{}}
}

func (c *jwksCache) fetch() error {
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(c.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("jwks: status %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kty, Crv, Kid, X, Y string
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return err
	}
	keys := map[string]*ecdsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue
		}
		xb, e1 := base64.RawURLEncoding.DecodeString(k.X)
		yb, e2 := base64.RawURLEncoding.DecodeString(k.Y)
		if e1 != nil || e2 != nil {
			continue
		}
		pub := &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xb), Y: new(big.Int).SetBytes(yb)}
		if !pub.Curve.IsOnCurve(pub.X, pub.Y) {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks: no usable keys")
	}
	c.keys, c.fetched = keys, time.Now()
	return nil
}

// key returns the public key for kid, refetching when the cache is stale or the kid is unknown
// (key rotation) -- but never more often than jwksRefreshMin, so a flood of bad tokens carrying
// random kids cannot turn this into a request amplifier against IDUNA.
func (c *jwksCache) key(kid string) (*ecdsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stale := time.Since(c.fetched) > jwksTTL
	if k, ok := c.keys[kid]; ok && !stale {
		return k, nil
	}
	if time.Since(c.tried) >= jwksRefreshMin {
		c.tried = time.Now()
		if err := c.fetch(); err != nil && len(c.keys) == 0 {
			return nil, err
		}
	}
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	return nil, errors.New("unknown signing key")
}

type idunaClaims struct {
	Sub         string
	DisplayName string
}

// verifyIdunaToken checks signature, alg, expiry and (when set) issuer. The returned Sub is the
// only identity the server trusts -- never a player_id the client typed alongside the token.
func verifyIdunaToken(c *jwksCache, token string, now time.Time) (*idunaClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("bad header")
	}
	var hdr struct{ Alg, Kid string }
	if json.Unmarshal(hb, &hdr) != nil || hdr.Alg != "ES256" {
		return nil, errors.New("unsupported alg")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		return nil, errors.New("bad signature encoding")
	}
	pub, err := c.key(hdr.Kid)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(pub, digest[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		return nil, errors.New("signature check failed")
	}
	cb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("bad claims")
	}
	var cl struct {
		Sub         string  `json:"sub"`
		DisplayName string  `json:"display_name"`
		Exp         float64 `json:"exp"`
		Iss         string  `json:"iss"`
	}
	if json.Unmarshal(cb, &cl) != nil || cl.Sub == "" {
		return nil, errors.New("bad claims")
	}
	if cl.Exp == 0 || now.Unix() > int64(cl.Exp) {
		return nil, errors.New("token expired") // exp is required: a non-expiring token is refused
	}
	if want := os.Getenv("GFD_WS_ISSUER"); want != "" && cl.Iss != want {
		return nil, errors.New("wrong issuer")
	}
	return &idunaClaims{Sub: cl.Sub, DisplayName: cl.DisplayName}, nil
}

// ---------------------------------------------------------------------------------------------
// WebSocket framing

type wsConn struct {
	raw    net.Conn
	br     *bufio.Reader
	wmu    sync.Mutex
	rbuf   []byte // unread payload of the current data message
	closed int32
	last   int64 // unix nanos of last client frame, for the idle watchdog
	// echo: browser terminals do not echo locally, so the server must (like an SSH PTY).
	echo bool
}

func newWSConn(raw net.Conn, br *bufio.Reader) *wsConn {
	return &wsConn{raw: raw, br: br, last: time.Now().UnixNano(), echo: true}
}

func (w *wsConn) echoesInput() bool { return w.echo }

func (w *wsConn) writeFrame(op byte, payload []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()
	hdr := []byte{0x80 | op}
	switch n := len(payload); {
	case n < 126:
		hdr = append(hdr, byte(n))
	case n < 1<<16:
		hdr = append(hdr, 126, 0, 0)
		binary.BigEndian.PutUint16(hdr[2:], uint16(n))
	default:
		hdr = append(hdr, 127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(hdr[2:], uint64(n))
	}
	_ = w.raw.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if _, err := w.raw.Write(hdr); err != nil {
		return err
	}
	_, err := w.raw.Write(payload)
	return err
}

// readMessage returns the next complete data message (text or binary), answering pings and
// honoring close frames along the way. Client frames MUST be masked (RFC 6455 §5.1).
func (w *wsConn) readMessage() (op byte, data []byte, err error) {
	var msg []byte
	var msgOp byte
	for {
		var h [2]byte
		if _, err = io.ReadFull(w.br, h[:]); err != nil {
			return
		}
		atomic.StoreInt64(&w.last, time.Now().UnixNano())
		fin, opc := h[0]&0x80 != 0, h[0]&0x0f
		if h[0]&0x70 != 0 {
			return 0, nil, errors.New("ws: reserved bits set")
		}
		if h[1]&0x80 == 0 {
			return 0, nil, errors.New("ws: unmasked client frame")
		}
		n := uint64(h[1] & 0x7f)
		switch n {
		case 126:
			var e [2]byte
			if _, err = io.ReadFull(w.br, e[:]); err != nil {
				return
			}
			n = uint64(binary.BigEndian.Uint16(e[:]))
		case 127:
			var e [8]byte
			if _, err = io.ReadFull(w.br, e[:]); err != nil {
				return
			}
			n = binary.BigEndian.Uint64(e[:])
		}
		if n > wsMaxFrame || uint64(len(msg))+n > wsMaxFrame {
			return 0, nil, errors.New("ws: frame too large")
		}
		var mask [4]byte
		if _, err = io.ReadFull(w.br, mask[:]); err != nil {
			return
		}
		p := make([]byte, n)
		if _, err = io.ReadFull(w.br, p); err != nil {
			return
		}
		for i := range p {
			p[i] ^= mask[i%4]
		}
		switch opc {
		case 0x8: // close
			_ = w.writeFrame(0x8, p)
			return 0, nil, io.EOF
		case 0x9: // ping
			_ = w.writeFrame(0xA, p)
			continue
		case 0xA: // pong
			continue
		case 0x0: // continuation
			if msgOp == 0 {
				return 0, nil, errors.New("ws: stray continuation")
			}
		case 0x1, 0x2:
			if msgOp != 0 {
				return 0, nil, errors.New("ws: interleaved data frames")
			}
			msgOp = opc
		default:
			return 0, nil, errors.New("ws: unknown opcode")
		}
		msg = append(msg, p...)
		if fin {
			return msgOp, msg, nil
		}
	}
}

// Read/Write make wsConn a net.Conn for handleConn. Every data message is a chunk of the MUD's
// byte stream, exactly as telnet/SSH would deliver it. A message of the form {"t":"resize",...}
// would be a control message, but none exist yet: the terminal is a plain text stream.
func (w *wsConn) Read(p []byte) (int, error) {
	for len(w.rbuf) == 0 {
		_, data, err := w.readMessage()
		if err != nil {
			return 0, err
		}
		w.rbuf = data
	}
	n := copy(p, w.rbuf)
	w.rbuf = w.rbuf[n:]
	return n, nil
}

func (w *wsConn) Write(p []byte) (int, error) {
	if atomic.LoadInt32(&w.closed) != 0 {
		return 0, net.ErrClosed
	}
	// Chunk so one huge room dump never exceeds what a browser reassembles comfortably.
	for off := 0; off < len(p); off += 16 << 10 {
		end := off + 16<<10
		if end > len(p) {
			end = len(p)
		}
		if err := w.writeFrame(0x2, p[off:end]); err != nil {
			return off, err
		}
	}
	return len(p), nil
}

// CloseWith sends a close frame carrying an application code (4401 = token refused, which the
// browser client treats as "sign in again" rather than "reconnect").
func (w *wsConn) CloseWith(code uint16, reason string) error {
	if atomic.SwapInt32(&w.closed, 1) == 0 {
		p := make([]byte, 2+len(reason))
		binary.BigEndian.PutUint16(p, code)
		copy(p[2:], reason)
		_ = w.writeFrame(0x8, p)
	}
	return w.raw.Close()
}

func (w *wsConn) Close() error {
	if atomic.SwapInt32(&w.closed, 1) == 0 {
		_ = w.writeFrame(0x8, []byte{0x03, 0xe8}) // 1000 normal closure
	}
	return w.raw.Close()
}
func (w *wsConn) LocalAddr() net.Addr                { return w.raw.LocalAddr() }
func (w *wsConn) RemoteAddr() net.Addr               { return w.raw.RemoteAddr() }
func (w *wsConn) SetDeadline(t time.Time) error      { return nil }
func (w *wsConn) SetReadDeadline(t time.Time) error  { return nil }
func (w *wsConn) SetWriteDeadline(t time.Time) error { return nil }

// ---------------------------------------------------------------------------------------------
// Listener

var (
	wsPerIPMu       sync.Mutex
	wsPerIPSessions = map[string]int{}
)

// wsTrustXFF is set by -web-trust-xff when the listener sits behind Google's external load
// balancer, which appends "<client-ip>,<lb-ip>" to X-Forwarded-For: the client is the SECOND-TO-
// LAST entry (anything earlier is client-supplied and spoofable). Off by default so a directly
// exposed listener never believes a forged header.
var wsTrustXFF bool

// wsRemoteIP is the address used for per-IP session caps and claim rate limiting.
func wsRemoteIP(r *http.Request) string {
	if wsTrustXFF {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[len(parts)-2])
			}
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// addrOverride lets the claim flow's per-IP rate limit see the real client IP behind the proxy.
type wsAddr struct{ ip string }

func (a wsAddr) Network() string { return "tcp" }
func (a wsAddr) String() string  { return net.JoinHostPort(a.ip, "0") }

type wsClaimConn struct {
	*wsConn
	addr net.Addr
}

func (c wsClaimConn) RemoteAddr() net.Addr { return c.addr }

func wsOriginAllowed(origin string) bool {
	allowed := os.Getenv("GFD_WS_ORIGINS")
	if allowed == "" {
		allowed = "https://wotan.okemily.com"
	}
	for _, o := range strings.Split(allowed, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}

// startWebListener serves GET /ws (and /GFD/ws, the path the public route forwards unchanged).
func startWebListener(port int) {
	jwksURL := os.Getenv("GFD_JWKS_URL")
	if jwksURL == "" {
		jwksURL = "https://iam.okemily.com/.well-known/jwks.json"
	}
	cache := newJWKSCache(jwksURL)
	mux := http.NewServeMux()
	h := func(w http.ResponseWriter, r *http.Request) { serveWebSocket(cache, w, r) }
	mux.HandleFunc("/ws", h)
	mux.HandleFunc("/GFD/ws", h)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	addr := fmt.Sprintf(":%d", port)
	log.Printf("[ws] listening on %s (IDUNA-token auth, jwks=%s)", addr, jwksURL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("[ws] listener stopped: %v", err)
	}
}

func serveWebSocket(cache *jwksCache, rw http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		!strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") ||
		r.Header.Get("Sec-WebSocket-Version") != "13" || r.Header.Get("Sec-WebSocket-Key") == "" {
		http.Error(rw, "websocket upgrade required", http.StatusUpgradeRequired)
		return
	}
	// Browsers always send Origin; refusing foreign origins stops other sites from driving a
	// logged-in player's session (cross-site WebSocket hijacking). Non-browser clients with no
	// Origin are allowed -- they still have to present a valid token.
	if o := r.Header.Get("Origin"); o != "" && !wsOriginAllowed(o) {
		log.Printf("[ws] refused origin %q", o)
		http.Error(rw, "origin not allowed", http.StatusForbidden)
		return
	}
	ip := wsRemoteIP(r)
	wsPerIPMu.Lock()
	if wsPerIPSessions[ip] >= wsMaxSessionsPerIP {
		wsPerIPMu.Unlock()
		http.Error(rw, "too many sessions", http.StatusTooManyRequests)
		return
	}
	wsPerIPSessions[ip]++
	wsPerIPMu.Unlock()
	defer func() {
		wsPerIPMu.Lock()
		if wsPerIPSessions[ip]--; wsPerIPSessions[ip] <= 0 {
			delete(wsPerIPSessions, ip)
		}
		wsPerIPMu.Unlock()
	}()

	hj, ok := rw.(http.Hijacker)
	if !ok {
		http.Error(rw, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	raw, brw, err := hj.Hijack()
	if err != nil {
		return
	}
	sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + wsGUID))
	fmt.Fprintf(brw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		base64.StdEncoding.EncodeToString(sum[:]))
	if brw.Flush() != nil {
		raw.Close()
		return
	}
	disableNagle(raw)
	ws := newWSConn(raw, brw.Reader)
	defer ws.Close()

	// Authenticate: the first message must be {"token":"<IDUNA SSO JWT>"}. The token travels in a
	// frame, never the URL, so it stays out of proxy and access logs.
	_ = raw.SetReadDeadline(time.Now().Add(wsAuthTimeout))
	_, first, err := ws.readMessage()
	_ = raw.SetReadDeadline(time.Time{})
	if err != nil {
		return
	}
	var auth struct{ Token string }
	if json.Unmarshal(first, &auth) != nil || auth.Token == "" {
		ws.Write([]byte("Authentication required -- sign in with IDUNA.\r\n"))
		ws.CloseWith(wsCloseAuth, "auth required")
		return
	}
	claims, err := verifyIdunaToken(cache, auth.Token, time.Now())
	if err != nil {
		log.Printf("[ws] auth refused ip=%s: %v", ip, err)
		ws.Write([]byte("Your IDUNA session is invalid or expired -- sign in again.\r\n"))
		ws.CloseWith(wsCloseAuth, "token refused")
		return
	}

	cc := wsClaimConn{wsConn: ws, addr: wsAddr{ip}}
	preset := resolveWebIdentity(cc, claims)
	if preset == nil {
		return // claim abandoned / IDUNA down: web has no guest tier, the account is the identity
	}
	log.Printf("[ws] session sub=%s character=%s ip=%s", claims.Sub, preset.name, ip)

	done := make(chan struct{})
	go wsKeepAlive(ws, done)
	handleConn(ws, false, preset, true)
	close(done)
}

func wsKeepAlive(ws *wsConn, done <-chan struct{}) {
	t := time.NewTicker(wsPingInterval)
	defer t.Stop()
	start := time.Now()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			if time.Since(time.Unix(0, atomic.LoadInt64(&ws.last))) > sshIdleTimeout || time.Since(start) > sshMaxSessionDur {
				log.Printf("[ws] closing idle/over-long session %s", ws.RemoteAddr())
				ws.Close()
				return
			}
			if ws.writeFrame(0x9, nil) != nil {
				return
			}
		}
	}
}

// resolveWebIdentity maps an authenticated IDUNA account to a character:
//
//  1. slot 1's fingerprint is already bound -> resume it silently;
//  2. else the account already owns a character (made in GFD Town, or earlier) -> bind slot 1 to
//     it and resume, so nobody is asked to re-create a character that exists;
//  3. else run the SAME name-claim flow SSH uses, creating the character under the account's own
//     player_id (so the WOTAN hat store, which looks characters up by player_id, finds it) and
//     binding slot 1.
//
// Only slot 1 exists today (webCharacterSlotLimit); further slots are reserved for multibox
// licensing and there is deliberately no character select.
func resolveWebIdentity(c wsClaimConn, claims *idunaClaims) *presetIdentity {
	if webCharacterSlotLimit(claims.Sub) < 1 {
		c.Write([]byte("This account is not licensed to play.\r\n"))
		return nil
	}
	fp := webFingerprint(claims.Sub, 1)
	characterID, err := gw.iduna.ResolveSSHFingerprint(fp)
	switch {
	case err == nil:
		ch, gerr := gw.iduna.GetCharacter(characterID)
		if gerr != nil {
			c.Write([]byte("Could not load your character -- try again shortly.\r\n"))
			return nil
		}
		return &presetIdentity{name: ch.Name, characterID: characterID, fingerprint: fp}
	case errors.Is(err, idunaclient.ErrNotFound):
		// fall through to the account lookup below
	default:
		c.Write([]byte("Could not reach IDUNA -- try again shortly.\r\n"))
		return nil
	}

	switch ch, perr := gw.iduna.GetCharacterByPlayer(claims.Sub); {
	case perr == nil:
		if berr := gw.iduna.BindSSHKey(ch.CharacterID, fp, fp); berr != nil && !errors.Is(berr, idunaclient.ErrConflict) {
			log.Printf("[ws] could not bind existing character %s to %s: %v (resuming anyway)", ch.CharacterID, fp, berr)
		}
		return &presetIdentity{name: ch.Name, characterID: ch.CharacterID, fingerprint: fp}
	case errors.Is(perr, idunaclient.ErrNotFound):
		return runClaimFlow(c, fp, fp, "IDUNA account", claims.Sub)
	default:
		c.Write([]byte("Could not reach IDUNA -- try again shortly.\r\n"))
		return nil
	}
}
