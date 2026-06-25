# DragonsNShit — Inventory & Equipment Northstar

**Target fidelity**: FFXI retail era (2002–2007 PS2/PC).  
**Aesthetic target**: Low-poly, high-intention. Every polygon earns its place.  
**Status**: Planning → implementation begins 2026-06-25.

---

## Why This Matters

Equipment in FFXI was not a stat spreadsheet. It was a character's relationship with the world. A piece of gear told you where someone had been, what they'd done, who'd trusted them with it. The inventory system was the ledger of that relationship.

DragonsNShit already has provenance chains on items. The infrastructure exists to make gear *mean something*. This northstar is the path from "items exist" to "gear defines a character."

---

## Current State (2026-06-25)

| Layer | Status | Location |
|---|---|---|
| Equipment slots (16 FFXI-standard) | ✓ Done | `server/gear/gear.go` |
| Item level averaging (EffectiveIL) | ✓ Done | `server/gear/gear.go` |
| Loot pool system (lot/pass) | ✓ Done | `server/loot/loot.go` |
| Items table (bare: type, name, IL, qty, provenance) | ✓ Done | IDUNA migration 202606230001 |
| Item definitions (stats, restrictions, category) | ✗ Missing | — |
| Inventory container (bags, stacks, transfer) | ✗ Missing | — |
| Equipment stat computation (sum + IL scale) | ✗ Missing | — |
| Job/race restrictions on gear | ✗ Missing | — |
| Item registry (JSON-loaded, server-authoritative) | ✗ Missing | — |
| Character equipment persistence (DB) | ✗ Missing | — |
| Character inventory persistence (DB) | ✗ Missing | — |
| Low-poly equipment model pipeline | ✗ Missing | — |

---

## Design Reference: FFXI Fidelity Points

The following FFXI systems are the direct target. Each is canonical to the era.

### Equipment Slots (16)

```
Main      Off       Ammo
Head      Neck      Ear-L   Ear-R
Body      Hand-L    Hand-R
Ring-L    Ring-R
Back      Waist     Legs    Feet
```

Already implemented in `server/gear/gear.go`. No changes needed.

### Item Categories

| Category | Stackable | Bag | Notes |
|---|---|---|---|
| Weapon | No | Inventory | Main/Off/Ammo/Ranged |
| Armor | No | Inventory | All body slots |
| Accessory | No | Inventory | Neck/Ear/Ring/Back/Waist |
| Consumable | Yes (up to 12 or 99) | Inventory | Food, medicines, scrolls |
| Material | Yes (up to 12 or 99) | Inventory | Crafting materials |
| Crystal | Yes (up to 12) | Inventory | Elemental crystals |
| Key Item | No | Key Items tab | Cannot be dropped/traded |
| Temporary | No | Temp tab | BCNM/dungeon only; auto-cleared |

### Item Stats (FFXI-era canonical set)

**Base stats**: STR, DEX, VIT, AGI, INT, MND, CHR  
**Combat**: Accuracy, Attack, Defense, Evasion, Magic Accuracy, Magic Attack Bonus, Magic Defense Bonus  
**Resists**: Fire, Ice, Wind, Earth, Thunder, Water, Light, Dark (resist delta values)  
**Special**: Haste (%), Store TP, Subtle Blow, Double Attack (%), Enmity (+/−), Fast Cast (%)  
**Crafting**: Skills bonuses (Woodworking, Smithing, Goldsmithing, Clothcraft, Leathercraft, Bonecraft, Alchemy, Cooking)  

Stats are stored as `map[string]int` on the `ItemDef`. Negative values are valid (cursed gear, debuffs).

### Job Restrictions

19 jobs matching FFXI retail: WAR, MNK, WHM, BLM, RDM, THF, PLD, DRK, BST, BRD, RNG, SAM, NIN, DRG, SMN, BLU, COR, PUP, DNC, SCH.

Stored as a bitmask (`uint32`). A job can equip an item if `(jobMask >> jobIndex) & 1 == 1`. 0 = all jobs. Special value `AllJobs = 0x7FFFF` (19 bits set).

### Item Flags

| Flag | Meaning |
|---|---|
| Rare | Only one of this item per character across all bags + equipped |
| Ex | Cannot be traded, dropped, or bazaared |
| Temporary | Cleared on zone exit or death |
| NoSave | Not persisted to DB (memory-only) |

### Inventory Bags (FFXI-era storage model)

| Bag | Default Slots | Expandable | Notes |
|---|---|---|---|
| Inventory | 30 | → 80 (Gobbiebag quests) | Primary carry bag |
| Storage | 80 | No | Mog House safe deposit |
| Key Items | ∞ (tab, not slots) | N/A | Separate non-spatial tab |
| Equipped | 16 | No | Virtual — mirrors gear.Equipment |
| Temporary | 10 | No | BCNM/instanced content |

---

## Architecture

### New Packages

#### `server/itemdef/` — Item Definition Registry

The canonical source of truth for what items *are*. Loaded from `data/items.json` at server start. Never mutated at runtime.

```go
type Category string
const (
    CatWeapon     Category = "weapon"
    CatArmor      Category = "armor"
    CatAccessory  Category = "accessory"
    CatConsumable Category = "consumable"
    CatMaterial   Category = "material"
    CatCrystal    Category = "crystal"
    CatKeyItem    Category = "key_item"
    CatTemporary  Category = "temporary"
)

type ItemDef struct {
    ID           int             // numeric item ID (FFXI-style ITEMID)
    Name         string
    Description  string
    Category     Category
    EquipSlots   []gear.Slot     // which slots this item can be equipped in
    Jobs         uint32          // job restriction bitmask; 0 = all jobs
    Level        int             // minimum level to equip
    Stats        map[string]int  // stat name → delta (negative OK)
    StackSize    int             // 1 = not stackable; 12/99 = stack max
    Flags        ItemFlags       // Rare | Ex | Temporary | NoSave
    ModelID      string          // which low-poly mesh to render
    IconID       int             // UI icon index
}

type Registry struct { ... }
func NewRegistry(path string) (*Registry, error)
func (r *Registry) ByID(id int) (*ItemDef, bool)
func (r *Registry) ByName(name string) (*ItemDef, bool)
```

#### `server/inventory/` — Inventory Containers

Bag-style containers. Slots hold item UUIDs (from the items DB table) + stack quantity. The bag does not own item metadata — it holds references. `itemdef.Registry` resolves what the item *is*.

```go
type Stack struct {
    ItemID   string   // UUID from items table
    DefID    int      // itemdef.ItemDef.ID
    Quantity int
}

type Bag struct {
    Capacity int
    slots    []*Stack // nil = empty slot
}

func NewBag(capacity int) *Bag
func (b *Bag) Add(s Stack) (slotIndex int, err error)
func (b *Bag) Remove(slotIndex, qty int) error
func (b *Bag) Find(defID int) []int  // returns slot indices
func (b *Bag) Move(from, to int) error
func (b *Bag) Count() int
func (b *Bag) Full() bool

// Mog is the full character storage: all bags + key items
type Mog struct {
    Inventory  *Bag
    Storage    *Bag
    Temporary  *Bag
    KeyItems   map[int]bool   // defID → carried
    Equipped   *gear.Equipment
}
```

#### `server/gear/` — Extended (existing package, new functions)

Add stat summation and job/race restriction checking:

```go
// ComputeStats sums all stats across equipped items using the item registry.
func (e *Equipment) ComputeStats(reg *itemdef.Registry, items ItemLookup) map[string]int

// CanEquip checks job restriction bitmask + level requirement.
func CanEquip(def *itemdef.ItemDef, job string, level int) error
```

### DB Migrations

#### `character_equipment` table
One row per (character, slot) pair. The slot column is a canonical slot string from `gear.AllSlots`.

```sql
CREATE TABLE character_equipment (
    character_id  CHAR(36) NOT NULL,
    slot          VARCHAR(16) NOT NULL,
    item_id       CHAR(36),          -- NULL = empty slot
    PRIMARY KEY (character_id, slot),
    FOREIGN KEY (character_id) REFERENCES characters(character_id),
    FOREIGN KEY (item_id) REFERENCES items(item_id)
)
```

#### `character_inventory` table
One row per occupied bag slot. `bag` is one of: `inventory`, `storage`, `temporary`.

```sql
CREATE TABLE character_inventory (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id  CHAR(36) NOT NULL,
    bag           VARCHAR(16) NOT NULL,
    slot_index    INTEGER NOT NULL,
    item_id       CHAR(36) NOT NULL,
    def_id        INTEGER NOT NULL,
    quantity      INTEGER NOT NULL DEFAULT 1,
    UNIQUE(character_id, bag, slot_index),
    FOREIGN KEY (character_id) REFERENCES characters(character_id),
    FOREIGN KEY (item_id) REFERENCES items(item_id)
)
```

#### Extend `items` table
The existing `items` table is a runtime ledger of item instances. It needs `def_id` and `flags` to join with item definitions:

```sql
ALTER TABLE items ADD COLUMN def_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE items ADD COLUMN flags  INTEGER NOT NULL DEFAULT 0;
```

---

## Art Direction: Low-Poly, High-Intention

### The Aesthetic Goal

FFXI PS2/PC era: approximately 2002–2004. The constraint was polygon budget, not artistic ambition. The team worked *with* the constraint, producing an aesthetic that reads as timeless rather than dated. We want that same relationship: polygons as a material, not a limitation.

**Reference frame**: 500–2000 triangles per character body. 100–500 per equipment piece. 256×256 textures for body, 128×128 for accessories.

### Principles

**1. Silhouette over surface**  
A helm must read as a helm at 20 meters. A great sword must read as a great sword from the minimap. Never sacrifice silhouette for surface detail. A clean edge catches light. A noisy edge disappears.

**2. Every polygon earns its place**  
If removing a polygon does not change the silhouette or the shadow, remove it. The best low-poly art is ruthlessly edited, not just low-resolution.

**3. Palette anchoring**  
Each armor tier gets a primary hue, a shadow hue, and one highlight accent. Tier 1 (level 1–25): muted earth tones. Tier 2 (26–50): deeper, richer base tones. Tier 3 (51–75): strong contrast, metallic accents. Rare/EX items: distinct palette break — a color that says "this is different" without glow spam.

**4. Equipment changes geometry, not just texture**  
A helm swaps the head model — the hair disappears, the silhouette changes. Body armor changes chest + shoulder width. Hands/legs/feet are distinct mesh swaps. No texture-only upgrades. Each piece must justify its polygon budget with a real geometry change.

**5. Light does the work**  
Diffuse + specular only. No normal maps, no PBR. A single directional light + ambient. The palette is designed to read under one light source. This also means zone-specific light direction and color temperature carries enormous narrative weight — day/night, interior/exterior, dungeon/sky — each changes the gear's perceived value.

**6. Particle discipline**  
Magic weapon procs: one small billboard emitter. Rare gear aura: subtle, slow-moving additive blend. No particle fireworks. The player's eye should always know where the character is. Particles that fight the silhouette are design failures.

### Gear Visual Tier System

| Tier | Level Range | Polygon Budget (per piece) | Texture | Color Signal |
|---|---|---|---|---|
| Initiate | 1–15 | 80–150 tris | 64×64 | Undyed cloth, rough iron |
| Journeyman | 16–30 | 100–200 tris | 128×128 | Dyed cloth, worked leather |
| Adventurer | 31–50 | 150–300 tris | 128×128 | Chainmail, bronze plate |
| Veteran | 51–65 | 200–400 tris | 256×256 | Engraved steel, fine silk |
| Endgame | 66–75 | 300–600 tris | 256×256 | Mythril, abjuration glow |
| Rare/EX | Any | Same as tier + accent mesh | 256×256 | Palette break + subtle aura |

### Model Pipeline (target workflow)

1. **Block out** in Blender at target triangle count with hard edges only
2. **Silhouette pass** — check at 50px height. If it doesn't read, redo the block-out
3. **Seam map** — UV unwrap for a single 128×128 or 256×256 atlas per piece
4. **Handpaint** in Krita: base color → shadow color → 1 accent color. No gradients. No texture photographs
5. **Export** as OBJ + PNG. Model pipeline converts to engine mesh format
6. **In-engine test**: equip on base character in zone day/night. Verify silhouette, verify palette reads at distance

---

## Milestone Roadmap

### M1 — Item Definition Foundation (S129-01 to S129-04)
*"Items know what they are"*

- `server/itemdef/` package: ItemDef, Category, JobMask, ItemFlags, Registry (JSON-loaded)
- `data/items.json`: seed with 50 canonical items (10 weapons, 15 armor pieces, 10 accessories, 5 consumables, 10 materials)
- `server/gear/` extended: `ComputeStats()`, `CanEquip()`
- DB migration: `def_id` + `flags` on items table
- Tests: registry load, stat sum, job restriction check

### M2 — Inventory Container (S129-05 to S129-07)
*"Items have a home"*

- `server/inventory/` package: Bag, Stack, Mog
- DB migrations: `character_inventory`, `character_equipment`
- IDUNA HTTP endpoints: `GET /api/v1/mmo/characters/{id}/inventory`, `GET /api/v1/mmo/characters/{id}/equipment`
- Tests: bag add/remove/move, stack merging, Rare flag enforcement, Ex flag enforcement

### M3 — Equip/Unequip Loop (S129-08 to S129-10)
*"Gear changes the character"*

- `EQUIP`/`UNEQUIP` game commands wired to `Mog.Equipped`
- Stat delta broadcast to character state on equip change
- `server/gear/ComputeStats()` integrated into combat stat pipeline
- Job + level restriction enforcement at equip time
- MUD command: `equip <slot> <item-name>`, `unequip <slot>`, `equipment` (list)
- Tests: full equip cycle, stat delta, restriction rejection

### M4 — Art Direction + Gear Visual Tier (S129-11 to S129-12)
*"Gear looks like what it is"*

- Art direction spec finalized (this document, plus per-tier reference sheets)
- First 5 armor sets modeled at correct polygon budget (Initiate through Journeyman)
- Model pipeline documented in `docs2/specs/MODEL_PIPELINE.md`
- In-engine equip → model swap wired for head/body slots

### M5 — Storage + Key Items (S129-13 to S129-14)
*"There is a place for everything"*

- Mog House storage bag (80 slots, character must be at home point)
- Key Items tab (separate from spatial inventory)
- `DEPOSIT`/`WITHDRAW` commands
- Gobbiebag expansion quest hooks (capacity +10 per completion, up to 80)

---

## Scope Lock

**In scope for this northstar:**
- Item definition system, inventory bags, equipment stat computation
- Job/level restrictions
- Persistence (character_inventory + character_equipment tables)
- Low-poly art direction spec and first 5 armor sets
- MUD equip/unequip/inventory commands

**Out of scope (later):**
- Augment/trial system (FFXI ToAU-era gear augments)
- Synthesis/crafting integration (hook exists in `server/craft/`)
- Bazaar (player-to-player trade)
- Equipment sets / macro swap system
- Race restrictions
- Cursed gear (must be uncursed by NPC)

---

*Authored 2026-06-25*
