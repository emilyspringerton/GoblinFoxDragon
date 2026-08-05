# GOLDENBAND → GFD Animation Integration — Northstar

*Written 2026-08-05. Founder, real-time, across two messages: "how are we gonna do animations for
arena or dragonsnshit? i can make some models in blender? where do i rig and animate also blender
or do we build our own? then how do we play the animations in the game do we need to write some
kind of engine?" → "ok i need it rigged up for GFD" → (after being told GOLDENBAND has no
mesh/skin format yet and no Blender asset exists anywhere in the monorepo to design one against)
→ "write the full implementation plan including what blender assets are needed from the founder
full plan to md somewhere synced to git."*

**Status:** Plan only. No code written yet. S144-06 (Phase 1 below) is immediately buildable by a
future session with no founder dependency. S144-07 (Phase 2) is blocked on the founder producing a
real Blender asset per §5.

---

## 1. What exists today, checked directly against the real code

### GOLDENBAND (`/home/fatbaby/GOLDENBAND`, standalone repo, HQ-SPEC-SIM-100 §8 build step 1)

`.gband` is a **flat, semantics-free motion asset** — a fixed 84-byte header (magic, version,
tick_rate, duration_ticks, num_channels, skeleton_hash, content_hash) followed by row-major
float32 channel data. The C sampler (`src/gband.c`) is exactly what it claims to be and nothing
more:

- `gb_sample(clip, tick)` — clamps `tick`, returns a pointer into `clip->data`. Pure array
  indexing.
- `gb_blend(a, b, w, out, n)` — `out[i] = a[i] + (b[i]-a[i])*w` for every channel. A plain per-
  channel lerp with **no rotation-awareness** (no quaternion/euler special-casing at this layer).
- `gb_verify` — sha256 check against `content_hash`.

There is no joint/bone/skeleton struct anywhere in `gband.h`/`gband.c`. `skeleton_hash` is a raw
32-byte field carried through the header for future validation; nothing resolves it to an actual
skeleton today.

`tools/gbtool`'s BVH importer (`bvh.go`) **does** parse the BVH `HIERARCHY` block — joint names,
parent/child nesting, `OFFSET` lines — but only to derive the flat channel-name list
(`"Hips/Chest.Zrotation"`-style dotted/slashed strings) and channel count. `OFFSET` values are
read past but never stored (`BVHJoint` only has `Name`+`Channels`). After import, the tree and all
rest-pose offsets are discarded; the manifest (`<name>.gband.json`) stores only `channels: []string`
in flat order, and the binary's `skeleton_hash` is hard-coded to 32 zero bytes with an explicit
"no skeleton asset resolution yet (v0 scope)" comment.

No mesh, skin-weight, vertex, or glTF code exists anywhere in the repo — confirmed by grep and by
`GBAND_FORMAT.md`'s own "what v0 does not cover" list (glTF import, skeleton assets, retargeting).

**Bottom line: GOLDENBAND today can sample and blend an animation curve. It cannot tell you what a
character looks like, what its skeleton is, or how to skin a mesh to it. All of that is new work.**

### GFD's `apps2/battlegrounds_gui` (the bespoke SDL2/OpenGL client — not the real Bedrock client)

- **Rendering**: modern shader-based, VAO/VBO + GLSL (`main.c:1182-1203`, `#version 150`, Lambert
  diffuse, `uMVP`/`uModel`/`uColor`/`uLightDir` uniforms). `upload_mesh`/`draw_mesh`
  (`main.c:1323-1342`) build/draw interleaved pos+normal geometry via `glDrawArrays`. (HUD text
  uses separate legacy immediate-mode GL — irrelevant to this work.)
- **Characters today**: every hero/mob is a hardcoded stack of unit cubes. `draw_hero_model`
  (`main.c:1434+`) is a `switch(hero_id)` where each of ~18-20 heroes is 1-4 `BOX(...)` macro
  calls transforming/drawing one shared `cube_mesh`. No mesh loading, no skinning, no bones,
  anywhere.
- **"Animation" today**: `compute_squish` (`main.c:2581`) is a decaying-cosine squash/stretch
  bounce applied uniformly to every box on movement/cast triggers. `hero_facing_rad[]`
  (`update_facing_from_motion`, `main.c:2558-2570`) is derived client-side from position deltas
  between snapshots — there is **no facing/yaw field on the wire protocol at all**
  (`ArenaHeroSnapshot`, `protocol.h:220-353`, carries position/hp/cooldowns/status timers, nothing
  directional). There is no idle/walk/attack state machine, just one scalar bounce.
- **Texturing**: none exists. No `IMG_Load`/`SDL_image`/`glTexImage2D`/`stbi_*` anywhere in this
  app. All geometry is flat-shaded, `uColor`-tinted only.
- No `.blend`/`.fbx`/`.gltf`/`.glb` file exists anywhere in the monorepo (checked, repo-wide).

---

## 2. Two-phase plan

Rigging up GOLDENBAND for GFD is really two separable projects: **(1) prove the animation-playback
mechanism works at all**, which needs no art asset, and **(2) render a real, continuously-deforming
skinned character**, which needs one. Building (2) blind — inventing a mesh/skin file format with
no real content to test it against — is how you end up redesigning it the moment real geometry
shows up. So: Phase 1 first, Phase 2 once there's something to import.

### Phase 1 — procedural box-rig proof (buildable now, zero founder dependency) — **S144-06**

Goal: a small skeleton, driven by a **real** `.gband` clip sampled through the **real** `gb_sample`/
`gb_blend` C functions, visibly animating inside `battlegrounds_gui` — proving the sample → forward-
kinematics → render chain end to end, before any mesh/skin design work happens.

- **Skeleton**: a small hand-authored joint list (suggested: `Hips` (root) → `Spine` → `Head`,
  plus `L_Arm`/`R_Arm` off `Spine` — 5 joints), hardcoded as a plain C struct array (name, parent
  index, rest local translation) in a new bridge file. **Not a new file format yet** — this is
  intentionally the cheapest possible skeleton representation, since inventing `.gskel` now with
  only this one synthetic rig to validate it against would be the same blind-design problem Phase
  2 is deferred to avoid.
- **Motion data — via the pipeline that already ships, not a new generator**: hand-write
  `idle.bvh`/`walk.bvh` text fixtures matching that 5-joint hierarchy (BVH is plain text — trivial
  to author by hand for 5 joints/a few frames), run them through the existing, tested
  `gbtool import --bvh` (S144-01) to produce real `.gband`/`.gband.json` files. Commit those as
  test assets under `apps2/battlegrounds_gui/assets/goldenband_test/`. This exercises the real
  shipped importer, not a shortcut.
- **Runtime FK**: per joint, per tick — sample the clip's rotation channels (`Xrotation`/
  `Yrotation`/`Zrotation`, standard BVH convention: only the root carries translation channels;
  children's local translation is their fixed rest offset), convert Euler → quaternion, compose
  local transform = rest-offset-translation + quaternion rotation, multiply by the parent's
  already-computed world transform (joints processed in parent-before-child order, guaranteed by
  construction since the array is authored that way). This is standard, well-understood forward
  kinematics — no new math concepts, just needs writing.
- **Render**: draw **one existing unit cube per joint**, scaled/oriented along the parent→child
  offset, at that joint's computed world transform — reusing `cube_mesh`/`draw_mesh`/the existing
  shader completely unchanged. Zero new vertex format, zero shader changes, zero new rendering
  primitives.
- **Integration point**: a new debug-spawned test entity behind a dev hotkey, **not** a
  replacement of any real hero's art — keeps blast radius on live gameplay at zero.
- **Animation selection**: reuse the existing pattern exactly — `update_facing_from_motion`
  already derives client state (facing) from position deltas with no wire-protocol change; idle-
  vs-walk selection for the test entity follows the same client-derived-from-motion approach (the
  test entity can patrol a short back-and-forth path to demonstrate both states).

Deliverable: a real, visible, screenshot-verifiable animated stick-figure-of-boxes in
`battlegrounds_gui`, moved by real `.gband` data through the real C sampler. This is the "do we
need to write some kind of engine" answer, concretely built: yes, and this is it, minus the mesh.

### Phase 2 — true vertex-weighted skinned mesh (blocked on a real Blender asset) — **S144-07**

Goal: replace the box-per-joint placeholder with a real continuous mesh that deforms smoothly
across joints, authored in Blender.

- **New formats in `GOLDENBAND/format/`**, specified in the same fixed-binary-table style as
  `GBAND_FORMAT.md`:
  - **`.gskel`** — joint count, then per joint: name, parent index (`-1` for root), rest local
    translation (vec3) + rest local rotation (quaternion), and a **precomputed inverse-bind
    matrix** (4×4 float32 — computed once by the exporter via Blender's own `mathutils`, so the C
    runtime never has to invert a matrix).
  - **`.gmesh`** — vertex count, then per vertex: position (vec3), normal (vec3), UV (vec2, held
    for Phase 2 but unused for texturing until a later pass), up to 4 bone indices + 4 normalized
    weights; plus a triangle index buffer.
- **Blender export script** (new, `GOLDENBAND/tools/blender_export/export_gband_rig.py`): a `bpy`
  script that walks the selected Armature + Mesh + vertex groups to emit `.gskel`/`.gmesh`.
  **Animation export needs no new code at all** — it reuses Blender's own built-in BVH exporter
  feeding straight into the existing, already-shipped `gbtool import --bvh` path.
- **Runtime skinning — CPU, deliberately, not GPU**: for each vertex, `skinned_pos = Σ weight_k *
  (joint_world[k] * inverse_bind[k] * bind_pos)`, recomputed each frame and pushed via
  `glBufferSubData` into the existing dynamic vertex buffer. Chosen specifically so **the existing
  shader (`VS_SRC`/`FS_SRC`) needs zero changes** — no bone-index/weight attributes, no shader-side
  skinning matrix palette. One character at a time, this is cheap enough on CPU; a GPU skinning
  pass is a real, separate, later optimization if/when many skinned characters are on screen at
  once.
- **Texturing**: genuinely new work (no `IMG_Load`/`SDL_image`/`glTexImage2D` exists in this app
  today) — not required for a first cut. Flat-shaded `uColor` tinting, matching the current art
  style, is an acceptable v1; a textured pass is separate follow-on work.

---

## 3. Open design decisions, settled now so a future session doesn't re-litigate them

- Rotations: quaternion throughout (rest pose and animated), BVH Euler channels converted to
  quaternion at bind/sample time.
- Translation: root-only carries translation channels (standard BVH/motion-capture convention);
  every other joint's local translation is its fixed rest offset.
- Skinning: CPU linear blend skinning, not GPU/shader-based, for both phases.
- No LOD, no multi-mesh-per-character, no texturing in v1 of either phase.
- Inverse-bind matrices are precomputed by the exporter, never inverted at runtime in C.

---

## 4. What the founder needs to do in Blender

This is the concrete, actionable list — Phase 2 cannot start without these, but **items 1-4 can
be done anytime, in parallel with someone building Phase 1**, since they don't depend on our
export tooling existing yet:

1. **Model** any low-poly character — doesn't need to be final art, just needs to exist. A simple
   humanoid or creature is fine.
2. **Rig it with a single Armature.** Name the bones simply and consistently. A suggested starter
   list, matching Phase 1's test skeleton so an early animation is directly reusable across both
   phases: `Hips`, `Spine`, `Head`, `L_Arm`, `R_Arm` (add legs/more joints freely — Phase 2's real
   exporter isn't restricted to this list, it reads whatever bones the armature actually has).
3. **Weight-paint vertex groups matching the bone names** — this is Blender's standard rigging
   workflow (Weight Paint mode, one vertex group per bone), no special tooling or plugin needed.
4. **Author (or apply) an Idle action and a Walk action** on the armature — Blender's Action
   Editor/NLA, standard keyframe animation. A short, simple loop is enough for a first pass; it
   doesn't need to be polished.
5. **Export each action via Blender's built-in exporter**: `File → Export → Motion Capture
   (.bvh)`. No plugin required — this is a stock Blender feature, and it's exactly the file format
   `gbtool import --bvh` already consumes today (S144-01, shipped). One `.bvh` file per action
   (e.g. `idle.bvh`, `walk.bvh`).
6. **Mesh/skin export (Phase 2 only)** needs the custom `export_gband_rig.py` script from §2,
   which doesn't exist yet — so for now, only the BVH action exports in step 5 are immediately
   usable by us. Steps 1-4 (model + rig + weight-paint) can happen anytime; the actual `.gskel`/
   `.gmesh` export step is a Phase 2 dependency on our side, not something blocked on the founder.

---

## 5. Backlog

Tracked as `S144-06` (Phase 1) and `S144-07` (Phase 2) under `EMILY/BACKLOG.md` SECTION 144.

## Related

- `GOLDENBAND/CLAUDE.md`, `GOLDENBAND/format/GBAND_FORMAT.md` — the existing `.gband` spec this
  plan builds on top of.
- `EMILY/docs/hq-specs/HQ-SPEC-SIM-100-springerton-seam-golden-band.md` — the source spec GOLDENBAND
  itself was built from (build steps 1-5; this doc's Phase 1/2 sit alongside step 2,
  "SHANKPIT integration," as a second Path-A consumer).
