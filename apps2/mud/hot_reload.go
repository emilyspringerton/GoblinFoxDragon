package main

// hot_reload.go — S412-05/S412-10, founder real-time: "we need hot reload for that" (item data),
// "also that should be hot reload" (mob-drop rates). SIGHUP is the standard, long-established
// Unix daemon convention for "reload config without a restart" (nginx, most syslog daemons) --
// chosen over a new in-game command because this MUD has no admin/GM role concept anywhere in its
// own command dispatch today (a real, checked-directly gap, not invented here to justify this
// choice): an in-game command would be reachable by any guest with zero gate, where a signal only
// reaches someone who already has shell access to the box -- the same real trust boundary every
// other ops action in this monorepo (systemctl restart, editing data/items.json itself) already
// relies on. `kill -HUP $(pgrep -f 'mud -port')` or `systemctl --user reload gfd-mud.service`
// (once the unit file adds ExecReload=, not done in this pass) both work without dropping any
// live player connection -- a real, meaningful property a full restart doesn't have.

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// reloadableDataFiles is the one, real, explicit list of what SIGHUP actually reloads --
// deliberately not "everything main() loads at startup" (some startup loads, e.g. crystal-seeded
// Meadow NPCs, are real one-time world-generation, not editable content tables a live operator
// would want to hot-tune) so this list only grows when a real, new hot-reloadable data source is
// added, not silently along with unrelated future startup-only loads.
func reloadableDataFiles() {
	if err := itemdefReg.LoadFile("data/items.json"); err != nil {
		log.Printf("[hot-reload] itemdef reload failed (kept previous data): %v", err)
	} else {
		log.Printf("[hot-reload] data/items.json reloaded")
	}
	if err := mobDropReg.LoadFile("data/mob_drops.json"); err != nil {
		log.Printf("[hot-reload] mobdrop reload failed (kept previous data): %v", err)
	} else {
		log.Printf("[hot-reload] data/mob_drops.json reloaded")
	}
}

// watchForReloadSignal blocks forever handling SIGHUP -- run in its own goroutine from main().
func watchForReloadSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	for range sigCh {
		log.Printf("[hot-reload] SIGHUP received, reloading data files")
		reloadableDataFiles()
	}
}
