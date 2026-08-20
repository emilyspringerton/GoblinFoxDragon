// Injected via emcc --pre-js (runs before Emscripten's own runtime/socket
// init, after the HTML shell's inline `Module = {...}` has already run).
//
// Why this exists: okemily.com serves this page over HTTPS, so browsers
// block any *insecure* ws:// connection a script tries to open from it
// (mixed-active-content blocking, enforced by every modern browser) --
// Emscripten's own default socket-emulation template (`ws://{host}:{port}`,
// what this build used with zero config) would silently fail here. Routes
// every socket connection through nginx's wss:// reverse-proxy instead
// (ops/nginx-okemily.conf's `/gfd-ws/<port>` location, added alongside this
// file -- same-origin so no CORS/mixed-content issue, real TLS termination
// at nginx, plain ws:// only on the loopback hop to gfd-wsudprelay).
// {port} is Emscripten's own template placeholder, substituted with the
// actual UDP port the C code's connect()/sendto() targeted -- {host} is
// deliberately unused since the relay dispatches purely by port (see
// GoblinFoxDragon/apps2/wsudprelay's own doc comment).
Module = typeof Module !== 'undefined' ? Module : {};
Module.websocket = Module.websocket || {};
Module.websocket.url = 'wss://okemily.com/gfd-ws/{port}';
