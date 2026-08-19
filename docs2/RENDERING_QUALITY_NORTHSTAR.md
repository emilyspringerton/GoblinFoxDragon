# Rendering Quality Northstar — DragonsNShit / apps/arena

**Status:** Tier 1 (cel-shading) started — outline + quantized-banding shader pass shipped
(REDGARDEN `apps/arena`). Everything else below is scoped, not built.
**Owner:** Emily Springerton
**Written:** 2026-08-19

---

## Why this doc exists

Founder, real-time: *"we have the duck reportedly telekinetic output from promptoverse we need
to iterate dragonsnshit interface and graphics to that in terms of quality"* → *"i know you need
models but lets lay the groundwork"* → *"do as much as you can programmatically and
procedurally"* → *"the abraxas FFXI gen is an awesome cell shaded low poly look, can we have that
as the gold standard for what we can achieve before going more realistic"* → *"like cell shading
engine then source engine quality before unreal engine quality."*

This is a different axis from `docs2/art_direction_tiers.md` (armor palette/mesh progression,
Tiers 1–5, locked spec, pending an artist). That doc is about *what geometry and textures exist*.
This doc is about *the rendering technique the engine uses to draw whatever geometry exists*, at
any armor tier — a cel-shading pass improves a Tier 1 leather-armor box exactly as much as a
Tier 5 void-armor one. The two roadmaps are complementary, not competing: `art_direction_tiers.md`
already specifies "flat-shaded, diffuse only" for Tiers 1–2 and "specular optional" at Tier 3 —
this doc's Tier-1 cel-shading pass (quantized diffuse, no specular/normal/emissive) is compatible
with that spec as written, not a violation of it.

## Reference images (real Prompt-o-verse generations, not invented)

- **"A Duck, Reportedly Telekinetic," FFXI style** —
  `okemily.com/prompt-o-verse/a-duck-reportedly-telekinetic-ffxi/`. Shows a full FFXI-style game
  HUD: circular minimap (top-right), multiple action/ability bars (bottom-left and bottom-right),
  a named health/MP bar with ornate metallic borders, atmospheric lighting (soft bloom around the
  moon, haze on distant mountains). The character render itself is a semi-realistic low-poly
  duck, not the target for the *shading style* (see Abraxas below) — this image's real value is
  the **HUD/interface richness**, which `apps/arena` almost entirely lacks today (no minimap at
  all; ability bars exist but as flat unbordered squares; no atmospheric post-processing).
- **"Abraxas," FFXI style** — `okemily.com/prompt-o-verse/abraxas-ffxi/`. Founder-named **gold
  standard for the current tier**: hard black ink outlines around the whole silhouette, flat
  quantized color bands (no smooth gradients — each surface reads as 2–3 discrete tones), bold
  saturated palette, faceted/low-poly geometry. This is a real, well-known, fully programmatic
  real-time rendering technique — **cel-shading / toon-shading** — not something that requires new
  3D art to approximate on existing geometry.

## The three-tier roadmap (founder's own framing)

1. **Cel-shading engine quality (now).** Outline pass (inverted-hull: back faces expanded along
   their own normal, drawn solid near-black, culled `GL_FRONT`) + quantized 3-band diffuse
   lighting, replacing the smooth-gradient Lambertian shading `apps/arena` had before. Needs zero
   new geometry or textures — works on the existing flat-colored hero boxes today. **Shipped**,
   see below.
2. **Source engine quality (next).** Real-time dynamic lighting beyond one directional light +
   ambient floor, basic normal mapping (the art_direction_tiers.md Tier 4+ spec already calls for
   this), specular highlights (Tier 3+), simple shadow mapping. A real step up, not started.
3. **Unreal engine quality (last, aspirational).** Full PBR materials, global illumination,
   screen-space reflections. Explicitly the founder's own last/aspirational tier — not scoped
   further here; premature to plan in detail before Tier 2 exists.

## What's shipped (Tier 1, this pass)

`REDGARDEN/apps/arena/src/main.c`:
- `FS_SRC` (main fragment shader): diffuse term quantized into 3 discrete bands (1.0 / 0.65 / 0.35)
  instead of a smooth `max(dot(...), 0.2)` gradient. Applies to **every** existing draw call for
  free — heroes, creeps, towers, wards, everything sharing the one shader program — zero call-site
  changes needed.
- `VS_OUTLINE_SRC` / `FS_OUTLINE_SRC`: new outline shader pair, inverted-hull technique (Wind
  Waker's approach, not invented here). Wired into `draw_hero_box_facing` (the shared primitive
  every hero-silhouette box draw call goes through) via a global `g_outline_prog` — draws each box
  twice: once expanded 0.12 units outward along its normal, back-faces only, solid near-black;
  once normally, front-facing, cel-shaded.
- Live-verified via Xvfb + `--observe` replay mode (no live match/server needed): screenshotted a
  real hero ("Unicorn") mid-match, confirmed visible outline + banding, no regressions to
  surrounding geometry (trees, wards, other props unaffected — outline is hero-box-specific by
  design). `bash scripts/test_arena.sh` and the full `bash scripts/build.sh` both pass clean.

**Known limitation, real and not hidden:** the outline is applied *per box*, not per whole
character. Since a multi-box hero silhouette (e.g. body + horn + tail as three separate
`draw_hero_box_facing` calls) draws each box's outline independently, there's a visible seam
outline between a hero's own body-part boxes, not one single clean silhouette line. A proper fix
is a screen-space edge-detection post-process pass (render to an FBO, Sobel/depth-discontinuity
edge pass, composite) — that's real, standard, and not attempted here since this renderer has no
FBO/post-process pipeline at all yet. Worth doing before/during the Source-engine tier push, since
that tier likely wants a post-process pipeline anyway (bloom, etc.).

## What's still open (programmatic/procedural, no new art needed)

Roughly in priority order — the minimap is the single biggest visual gap versus the Duck reference:

1. **Procedural minimap.** Doesn't exist at all today. Fully buildable from data `apps/arena`
   already has (hero/creep/tower positions, map bounds) — top-down orthographic render to an FBO
   texture, clipped to a circle (fragment-shader distance mask), composited into the existing 2D
   HUD pass top-right, matching the Duck reference's layout. No art assets required.
2. **Post-process edge-detection outline pass.** Fixes the per-box-seam limitation above; also the
   prerequisite for outlining any future non-box mesh (skinned Tyler, real character models once
   they exist) with one clean silhouette line instead of per-submesh seams.
3. **Procedural UI chrome.** Ornate-looking beveled/gradient borders around HUD panels (health
   bar, minimap, ability bars) via shader-based framing (SDF or simple beveled-quad borders), not
   hand-drawn texture art — matching the Duck reference's metallic panel borders.
4. **Atmospheric/post-process polish.** Soft bloom around bright light sources (the Duck
   reference's moon glow), distance fog/haze — standard post-process passes once an FBO pipeline
   exists (shares infrastructure with item 2).
5. **Ability icon system, procedural fallback.** The Duck reference's action bars have real icon
   art per ability; `apps/arena`'s ability tiles are flat colored squares today. Real icon art is
   blocked on an artist (same as `art_direction_tiers.md`'s armor meshes) — but a procedural
   fallback (per-ability generated glyph/shape, same "`sprite_path` empty means fall back to a
   procedural default" pattern BRAWLPIT already uses for its own sprite system) can ship before
   real art exists, same "lay the groundwork" spirit as this whole doc.

## What's genuinely blocked on real 3D art/models (not attempted here)

- Matching the Duck/Abraxas images' actual *character geometry* (faceted low-poly duck/gorgon-
  cobra-rooster shapes) — `apps/arena`'s heroes are boxes; no amount of shader work turns a box
  into a duck. Needs either a real 3D artist (same blocker `art_direction_tiers.md` already names
  for armor meshes) or a 2D-image-to-3D-model AI pipeline this monorepo doesn't have yet.
- Real ability-icon art (item 5 above's *actual* art, not its procedural fallback).
- Environment/prop models beyond the current flat-colored primitives (trees, rocks).

## Explicitly out of scope for this doc

A separate founder ask surfaced in the same real-time thread — unifying login so both SHANKPIT's
`apps/lobby` and GoblinFoxDragon's MUD can enter the same persistent world (near-term), eventually
extending to dropping from the MMO/battlegrounds mode into an FPS mode (longer-term vision) — is a
cross-repo (SHANKPIT ↔ GFD ↔ IDUNA) auth/protocol project, not a rendering-quality question.
Logged via `emily observe` (2026-08-19), not scoped further in this doc.
