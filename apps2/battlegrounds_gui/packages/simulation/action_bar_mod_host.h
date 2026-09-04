/* action_bar_mod_host.h -- real extern declarations for action_bar_mod.c's own exported entry
 * point plus its internal per-job helpers (VS0 doesn't mark unexported top-level defns static, so
 * they're technically visible too -- declared here for completeness, not because host code calls
 * them directly). Same "-include this header before compiling the generated C" pattern ECOWAR's
 * own card_effect_mod_host.h already established -- pure C linking, no cgo layer needed.
 *
 * on_gfd_ability_for_slot has no host-side call-INTO-C requirement -- it's pure PARENA decision
 * logic, called FROM host C (town_ability_for_slot, apps2/battlegrounds_gui/src/main.c), not
 * calling back out. See action_bar_mod.prn's own header comment (PARENA/stdlib/gfd/) for the
 * real ABI and the job-id/ability-id encodings this function's int params/return actually mean.
 *
 * parena_runtime.h in this same directory is DELIBERATELY the original, minimal 41-line version
 * of that file (git rev 9bdf91e in PARENA itself), not the current upstream one -- same real
 * "each repo pins its own compatible copy" precedent ECOWAR's own packages/simulation/
 * parena_runtime.h already established (confirmed by diffing the two directly). Upstream's
 * current version pulls in SDL2/SDL_ttf and POSIX pty/socket/process helpers this repo's I32-
 * only mods never call -- real, found live trying to WASM-build this exact mod: the modern
 * header's pty glue (`forkpty`) doesn't even compile under Emscripten's libc. This mod (and any
 * future GFD mod that stays I32-only, matching every ECOWAR/PAPERCRAFT mod's own real VS0
 * ceiling) needs nothing from parena_runtime.h beyond the bare Arena declarations it never
 * actually calls -- if a future GFD mod needs real arena/string helpers from a newer runtime,
 * that's a real, separate upgrade to make deliberately, not something to inherit by accident.
 */
#ifndef ACTION_BAR_MOD_HOST_H
#define ACTION_BAR_MOD_HOST_H

extern int on_gfd_ability_for_slot(int job, int slot);
extern int blm_ability_for_slot(int slot);
extern int war_ability_for_slot(int slot);
extern int mnk_ability_for_slot(int slot);
extern int whm_ability_for_slot(int slot);
extern int rdm_ability_for_slot(int slot);
extern int thf_ability_for_slot(int slot);
extern int smn_ability_for_slot(int slot);

#endif /* ACTION_BAR_MOD_HOST_H */
