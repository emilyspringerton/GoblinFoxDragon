# Art Direction Reference — GoblinFoxDragon Tier Palettes

**Scope:** DragonsNShit MMO — Initiate through Endgame tiers. Armors 1–5 (Leather + Chainmail).  
**Status:** Palette spec LOCKED. 3D model budget + UV spec LOCKED. Actual meshes pending artist.  
**Owner:** Emily Springerton  
**Updated:** 2026-06-27

---

## Tier Palette System

Five progression tiers, each with a distinct material identity and color family. All palettes are optimized for low-poly (PS1-era) rendering.

| Tier | Name | Level Range | Primary Hue | Accent | Material Feel |
|------|------|-------------|-------------|--------|--------------|
| 1 | Initiate | 1–9 | Raw leather brown (#7B4F2E) | Brass (#B5922A) | Unfinished, worn; hero is nobody yet |
| 2 | Adventurer | 10–24 | Grey chainmail (#9CA3AF) | Iron (#6B7280) | Functional; starting to look like someone |
| 3 | Champion | 25–44 | Bronze (#CD7F32) | Ochre (#D4A017) | Battle-worn polish; starting to shine |
| 4 | Veteran | 45–59 | Dark steel (#374151) | Cobalt trim (#1D4ED8) | Military authority; muted but deliberate |
| 5 | Endgame | 60+ | Void black (#111827) | Electric violet (#7C3AED) | Transcendent; wrong to be here |

---

## Per-Armor Spec (First 5 Sets)

### Set 1: Leather (Tier 1 — Initiate)

- **Primary:** Raw cowhide brown (#7B4F2E), diffuse only
- **Secondary:** Undyed linen straps (#D6C7A1)
- **Accent:** Brass buckles (#B5922A) — 2–4 pixels wide at 64×64 UV
- **Silhouette:** Minimal padding. No shoulder plates. Belt visible.
- **Poly budget:** ≤ 280 tris (body), ≤ 80 tris (helmet)
- **UV res:** 64×64 px per piece
- **Normal map:** None (flat-shaded, PS1 aesthetic)
- **Damage state variants:** None at Tier 1

### Set 2: Chainmail (Tier 2 — Adventurer)

- **Primary:** Mid-grey (#9CA3AF), procedural chain texture at 64×64
- **Secondary:** Padded undercoat in off-white (#F3F4F6)
- **Accent:** Iron clasps (#6B7280)
- **Silhouette:** Added shoulder pauldrons (low). Coif over head.
- **Poly budget:** ≤ 340 tris (body), ≤ 90 tris (helmet/coif)
- **UV res:** 64×64 px per piece; chain pattern tileable 4×4
- **Normal map:** Chain normal baked at 128×128 (optional, lower priority)
- **Damage state variants:** None at Tier 2

### Set 3: Bronze (Tier 3 — Champion)

- **Primary:** Polished bronze (#CD7F32), specular highlight pass
- **Secondary:** Ox-blood leather (#6B1A1A) trim
- **Accent:** Ochre (#D4A017) rune lines on breastplate (2px UV)
- **Silhouette:** Full cuirass. Winged pauldrons. Greaves visible.
- **Poly budget:** ≤ 420 tris (body), ≤ 100 tris (helmet)
- **UV res:** 128×128 px per piece
- **Normal map:** Optional specular-only bake
- **Damage state variants:** 1 (cracked bronze texture swap)

### Set 4: Dark Steel (Tier 4 — Veteran)

- **Primary:** Dark anthracite (#374151), matte
- **Secondary:** Cobalt blue inlay (#1D4ED8) at seams
- **Accent:** Gold filigree (#F59E0B) at shoulder trim only
- **Silhouette:** Full plate. High collar. Integrated greaves + gauntlets.
- **Poly budget:** ≤ 520 tris (body), ≤ 120 tris (helmet)
- **UV res:** 128×128 px per piece
- **Normal map:** Required — rivet + plate seam bake
- **Damage state variants:** 2 (scorched, dented)

### Set 5: Void (Tier 5 — Endgame)

- **Primary:** Void black (#111827), slight iridescent sheen (emissive pass at 15%)
- **Secondary:** Electric violet (#7C3AED) edge emission glow
- **Accent:** White eye slit glow (#FFFFFF, emissive only)
- **Silhouette:** Alien geometry. Asymmetric pauldrons. Floating elements (no physical strap).
- **Poly budget:** ≤ 640 tris (body), ≤ 140 tris (helmet)
- **UV res:** 256×256 px per piece
- **Normal map:** Required — complex surface with void cracks
- **Emissive map:** Required — violet edge glow baked to emissive channel
- **Damage state variants:** 3 (cracked void, bleed effect)

---

## Shader Rules (All Tiers)

- Diffuse only at Tiers 1–2 (no specular, no emissive)
- Specular optional at Tier 3 (max 0.4 spec power)
- Normal map required at Tier 4+
- Emissive required at Tier 5
- No transparency. No alpha blend. Edge emission only.
- Max 2 texture slots per armor piece (diffuse + normal OR diffuse + emissive)

---

## Pending Artist Deliverables

1. [ ] Leather armor mesh (≤280 tris body + ≤80 helmet) + UV @ 64×64 — Tier 1
2. [ ] Chainmail armor mesh (≤340 tris body + ≤90 coif) + UV @ 64×64 + chain tile — Tier 2
3. [ ] Bronze armor mesh (≤420 tris) + UV @ 128×128 — Tier 3
4. [ ] Dark Steel armor mesh (≤520 tris) + normal map — Tier 4
5. [ ] Void armor mesh (≤640 tris) + emissive map — Tier 5
6. [ ] Swatch sheet: one 512×512 PNG per tier, all 5 palette cells + material label

Deliver to: `GoblinFoxDragon/assets/art/armor/` (file naming: `tier{N}_armor_{piece}.blend + .png`)
