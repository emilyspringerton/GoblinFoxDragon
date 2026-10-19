#!/usr/bin/env python3
"""Generate the TYLER-lore crafting content: items, recipes, vendor stock.

Writes (idempotent; rows this script owns are replaced, everything else is left alone):
  data/items.json    items with ids 2000-2999 (stable: an existing name keeps its id)
  data/recipes.json  the whole recipe book (loaded by server/craft.LoadRecipesFile)
  apps2/mud/zz_craft_content_gen.go   vendor stock + food/consumable tables

Run from the repo root:  python3 scripts/gen_lore_items.py && gofmt -w apps2/mud/zz_craft_content_gen.go
Every ingredient must be obtainable: mined, fished, dropped by a mob, sold by a vendor, or itself
crafted. scripts test (apps2/mud/craft_content_test.go) enforces that.
"""
import json, os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
def P(*a): return os.path.join(ROOT, *a)
def slug(n): return n.lower().replace(" ", "-")

OWN_LO, OWN_HI = 2000, 3000

HEAVY = ["WAR", "PLD", "DRK", "SAM", "DRG", "RUN"]
LIGHT = ["THF", "NIN", "DNC", "MNK", "BST", "RNG", "ASN", "COR", "PUP", "BLU"]
ROBE = ["WHM", "BLM", "RDM", "SMN", "SCH", "GEO", "BRD"]
SWORD = ["WAR", "PLD", "DRK", "RDM", "NIN", "RUN", "BLU", "COR"]
DAGGER = ["WAR", "MNK", "RDM", "THF", "BST", "NIN", "DNC", "ASN", "COR"]
AXE = ["WAR", "DRK", "BST", "RUN"]
STAFF = ["WHM", "BLM", "RDM", "SMN", "SCH", "GEO", "PUP"]
BOW = ["WAR", "RNG", "NIN", "COR"]
KATANA = ["SAM", "NIN"]
GREAT = ["WAR", "DRK", "RUN"]
H2H = ["MNK", "PUP"]
INSTR = ["BRD"]

items, recipes = [], []
_stable = {}

def load_stable():
    try:
        for i in json.load(open(P("data/items.json"))):
            if OWN_LO <= i["id"] < OWN_HI:
                _stable[i["name"]] = i["id"]
    except FileNotFoundError:
        pass

_next = [OWN_LO]
_used = set()
def alloc(name):
    if name in _stable:
        i = _stable[name]
    else:
        used = set(_stable.values()) | _used
        while _next[0] in used:
            _next[0] += 1
        i = _next[0]
    _used.add(i)
    return i

def add(name, category, **kw):
    d = {"id": alloc(name), "name": name, "category": category, "stack_size": kw.pop("stack", 1)}
    d.update({k: v for k, v in kw.items() if v not in (None, [], {})})
    items.append(d)
    return slug(name)

# ---------------------------------------------------------------- builders
def weapon(name, wtype, L, model, jobs, delay, extra=None, desc=None, slots=None):
    atk = round(10 + L * 1.7)
    mult = {"dagger": .8, "axe": 1.1, "greatsword": 1.6, "katana": 1.15, "sword": 1.0, "bow": 1.0, "h2h": .9}.get(wtype, 1.0)
    st = {}
    if wtype == "staff":
        st = {"magic_attack_bonus": round(3 + L * .8), "mnd": 1 + L // 9, "int": 1 + L // 9}
    elif wtype == "instrument":
        st = {"chr": 1 + L // 8, "mnd": 1 + L // 10}
    elif wtype == "bow":
        st = {"attack": round(atk * mult), "ranged_attack": round(atk * .9), "accuracy": 1 + L // 6}
    else:
        st = {"attack": round(atk * mult), "str": 1 + L // 10, "accuracy": L // 8}
    st.update(extra or {})
    s = slots or (["main", "off"] if wtype in ("sword", "dagger", "axe") else ["main"])
    base = add(name, "weapon", weapon_type=wtype, equip_slots=s, jobs=jobs, level=L, stats=st,
               model_id=model, delay=delay, description=desc)
    plus(name, "weapon", weapon_type=wtype, equip_slots=s, jobs=jobs, level=L, stats=st, model_id=model, delay=delay, description=desc)
    return base

def armor(name, slot, L, model, jobs, extra=None, desc=None, heavy=False):
    d = round((6 + L * .9 + (4 if slot == "body" else 0)) * (1.25 if heavy else 1))
    if slot in ("hand-l", "head", "feet", "legs"):
        d = max(2, round(d * (.75 if slot != "legs" else .9)))
    st = {"defense": d}
    st.update(extra or {})
    slots = ["hand-l", "hand-r"] if slot == "hand-l" else [slot]
    base = add(name, "armor", equip_slots=slots, jobs=jobs, level=L, stats=st, model_id=model, description=desc)
    plus(name, "armor", equip_slots=slots, jobs=jobs, level=L, stats=st, model_id=model, description=desc)
    return base

def acc(name, slots, L, stats, model, jobs=None, desc=None):
    base = add(name, "accessory", equip_slots=slots, jobs=jobs, level=L, stats=stats, model_id=model, description=desc)
    plus(name, "accessory", equip_slots=slots, jobs=jobs, level=L, stats=stats, model_id=model, description=desc)
    return base

def plus(name, category, **kw):
    """HQ variant: every stat +10% (min +1)."""
    st = {k: v + max(1, v // 10) for k, v in kw["stats"].items()}
    kw = dict(kw, stats=st, level=kw["level"])
    desc = kw.get("description")
    kw["description"] = (desc + " " if desc else "") + "(High quality.)"
    items.append(dict({"id": alloc(name + " +1"), "name": name + " +1", "category": category, "stack_size": 1},
                      **{k: v for k, v in kw.items() if v not in (None, [], {})}))

def mat(name, desc=None, model="item_material_fat", stack=12):
    return add(name, "material", model_id=model, description=desc, stack=stack)

def cons(name, desc, model="item_potion", stack=12, stats=None):
    return add(name, "consumable", model_id=model, description=desc, stack=stack, stats=stats)

def recipe(craft, result_name, diff, crystal, ing, yield_=1, hq=True, name=None):
    r = {"id": slug(result_name), "name": name or result_name, "craft": craft, "difficulty": diff,
         "crystal": crystal, "ingredients": {slug(k) if not k.islower() else k: v for k, v in ing.items()},
         "result": slug(result_name)}
    if hq:
        r["hq1"] = r["hq2"] = r["hq3"] = slug(result_name + " +1")
    if yield_ > 1:
        r["yield"] = yield_
    recipes.append(r)

# raw obtainable ingredient slugs (mined / fished / dropped / vendor) --------------------------
CU, TIN, FE, AG, AU, SW = "copper-ore", "tin-ore", "iron-ore", "silver-ore", "gold-ore", "swampite"
CLAY, PEAT = "clay", "peat"
FIRE, ICE, WIND, EARTH, THUNDER, WATER, LIGHTC, DARK = ("fire-crystal", "ice-crystal", "wind-crystal", "earth-crystal",
                                                       "thunder-crystal", "water-crystal", "light-crystal", "dark-crystal")

# ------------------------------------------------------------ materials
def build_materials():
    m = {}
    # new raw materials sold by vendors
    for n, d in [("Ash Lumber", "Pale pale timber from the Handington lumberyard."),
                 ("Heather Lumber", "Wiry upland timber from Wren's Perch."),
                 ("Reed Lumber", "Light, hollow marsh reed bundled for building."),
                 ("Moor Oak Lumber", "Dark oak that has soaked up a century of fog."),
                 ("Ridge Pine Lumber", "Straight, cold-hardened pine from the Eastwind Ridge."),
                 ("Tanuki Rice", "Short-grain rice from the Osaka party store back room."),
                 ("Party Store Flour", "A paper sack of flour. Best before: ambiguous."),
                 ("Brine Salt", "Coarse grey salt, scraped from Ichthyosapien shell-markings."),
                 ("Shrine Greens", "Bundled greens from a shrine garden. Offered, then collected."),
                 ("Spring Water", "A stoppered flask of clean spring water."),
                 ("Lamp Wick", "Braided wick for the Eastwind lamps.")]:
        m[slug(n)] = mat(n, d)
    # smelted / processed intermediates
    for n, d in [("Copper Ingot", "A bar of copper."), ("Tin Ingot", "A bar of tin."), ("Iron Ingot", "A bar of iron."),
                 ("Quill-Brass Ingot", "Copper and tin alloyed the Eastwind way: bright, plain, hard to tarnish."),
                 ("Silver Ingot", "A bar of silver, stamped with a shell-mark."),
                 ("Shell-Steel Ingot", "Iron folded with crushed beetle carapace. Heikegani smiths guard the ratio."),
                 ("Swampite Ingot", "Black, faintly warm metal from the bog."),
                 ("Gold Ingot", "A bar of gold. The ledger notes its weight."),
                 ("Ward-Steel Ingot", "Steel tempered with protocol stone. It resists being documented."),
                 ("Cured Wolf Pelt", "A supple, cured wolf pelt."), ("Cured Hare Hide", "A soft cured hare hide."),
                 ("Cured Badger Hide", "A tough cured badger hide."), ("Rimefox Leather", "Frost-stiff leather that never quite thaws."),
                 ("Mini-Bike Leather", "Scuffed, oil-stained leather with a very particular smell of gasoline and freedom."),
                 ("Spider Silk Cloth", "A bolt of Kuroshio silk, light and nearly invisible."),
                 ("Down Cloth", "Fledgling down felted into a warm cloth."),
                 ("Vellum Weave", "Archive vellum thread woven with ink. It will not be read aloud."),
                 ("Wraith Cloth", "Cloth spun from windwraith thread; it moves when the air doesn't."),
                 ("Ward Paper", "Talisman paper, blank. For now."),
                 ("Ash Planks", "Sawn ash."), ("Heather Planks", "Sawn heather-wood."), ("Reed Planks", "Pressed reed boards."),
                 ("Moor Oak Planks", "Sawn moor oak."), ("Ridge Pine Planks", "Sawn pine."),
                 ("Bone Meal", "Ground pellet-bone."), ("Ward Ink", "Black ink for talismans and for documents that must outlive their authors."),
                 ("Shell Glue", "Gluey resin boiled from crab shell."), ("Dough", "A lump of Party Store Flour and water."),
                 ("Stock", "A savory stock base.")]:
        m[slug(n)] = mat(n, d)
    return m

def build_raw():
    """Item rows for every raw slug that only existed as a drop/mining id, so each resolves to a
    real, named item (and quest/recipe ingredients display properly)."""
    existing_names = {i["name"] for i in json.load(open(P("data/items.json"))) if not (OWN_LO <= i["id"] < OWN_HI)}
    existing_slugs = {slug(n) for n in existing_names}
    for n, d in [("Copper Ore", "Reddish ore from the Meadow rocks."), ("Tin Ore", "Soft grey ore."), ("Silver Ore", "Bright ore, mined in the murk."),
                 ("Gold Ore", "Heavy yellow ore from the Deep Mire."), ("Swampite", "Black bog ore, warm to the touch."),
                 ("Clay", "Wet grey clay."), ("Peat", "Dense, dark fuel.")]:
        if slug(n) not in existing_slugs:
            mat(n, d)
    try:
        drops = json.load(open(P("data/mob_drops.json")))
    except FileNotFoundError:
        drops = []
    known = set(existing_slugs) | {slug(i["name"]) for i in items}
    for t in drops:
        for it in t["items"]:
            if it["id"] not in known and it["name"] not in existing_names:
                known.add(it["id"])
                if slug(it["name"]) == it["id"]:
                    mat(it["name"], "A material dropped by creatures of the wild.")

# --------------------------------------------------------------- crystals
def build_crystals():
    add("Light Crystal", "crystal", model_id="crystal_water", stack=12, description="A crystal that hums at the edge of hearing.")
    # Dark Crystal exists as a mob-drop slug only; give it an item row so it resolves everywhere.
    add("Dark Crystal", "crystal", model_id="crystal_earth", stack=12, description="A crystal that seems to take the light in rather than give it back.")

# ----------------------------------------------------------------- gear
def blacksmith():
    c = "blacksmith"
    # smelting
    recipe(c, "Copper Ingot", 1, FIRE, {CU: 2}, hq=False)
    recipe(c, "Tin Ingot", 4, FIRE, {TIN: 2}, hq=False)
    recipe(c, "Iron Ingot", 12, FIRE, {FE: 2}, hq=False, name="Smelt Iron Ingot")
    recipes[-1]["id"] = "smelt-iron-ingot"
    recipe(c, "Quill-Brass Ingot", 14, FIRE, {"copper-ingot": 1, "tin-ingot": 1}, hq=False)
    recipe(c, "Silver Ingot", 26, FIRE, {AG: 2}, hq=False)
    recipe(c, "Shell-Steel Ingot", 30, FIRE, {"iron-ingot": 2, "beetle-shell": 2}, hq=False)
    recipe(c, "Swampite Ingot", 36, FIRE, {SW: 2}, hq=False)
    recipe(c, "Gold Ingot", 48, FIRE, {AU: 2}, hq=False)
    recipe(c, "Ward-Steel Ingot", 70, FIRE, {"iron-ingot": 2, "protocol-stone": 2, "dark-crystal": 1}, hq=False)
    # copper
    s = weapon("Copper Shortsword", "sword", 3, "sword_iron_short", SWORD, 220)
    recipe(c, "Copper Shortsword", 3, FIRE, {"copper-ingot": 2})
    weapon("Copper Knife", "dagger", 2, "dagger_basic", DAGGER, 160); recipe(c, "Copper Knife", 2, FIRE, {"copper-ingot": 1})
    weapon("Copper Hatchet", "axe", 5, "axe_hand", AXE, 240); recipe(c, "Copper Hatchet", 5, FIRE, {"copper-ingot": 2, "ash-planks": 1})
    armor("Copper Mail", "body", 5, "body_chain", HEAVY, heavy=True); recipe(c, "Copper Mail", 8, FIRE, {"copper-ingot": 4})
    armor("Copper Cap", "head", 4, "head_iron", HEAVY, heavy=True); recipe(c, "Copper Cap", 6, FIRE, {"copper-ingot": 2})
    # quill-brass (Eastwind)
    weapon("Quill-Brass Sword", "sword", 11, "sword_steel", SWORD, 228, {"int": 1}, "Bright brass, balanced like a pen.")
    recipe(c, "Quill-Brass Sword", 18, FIRE, {"quill-brass-ingot": 2, "sentry-quill": 1})
    weapon("Quill-Brass Stiletto", "dagger", 10, "dagger_003", DAGGER, 164, {"dex": 2}, "Sharpened to a nib.")
    recipe(c, "Quill-Brass Stiletto", 16, FIRE, {"quill-brass-ingot": 1, "sentry-quill": 1})
    weapon("Quill-Brass Axe", "axe", 12, "axe_hand", AXE, 244); recipe(c, "Quill-Brass Axe", 20, FIRE, {"quill-brass-ingot": 2, "heather-planks": 1})
    armor("Quill-Brass Hauberk", "body", 14, "body_chain", HEAVY, {"vit": 2}, heavy=True); recipe(c, "Quill-Brass Hauberk", 24, FIRE, {"quill-brass-ingot": 4, "down-cloth": 1})
    armor("Quill-Brass Helm", "head", 12, "head_iron", HEAVY, heavy=True); recipe(c, "Quill-Brass Helm", 20, FIRE, {"quill-brass-ingot": 2, "brass-plate-shard": 1})
    armor("Quill-Brass Gauntlets", "hand-l", 13, "hands_knuckles", HEAVY, heavy=True); recipe(c, "Quill-Brass Gauntlets", 22, FIRE, {"quill-brass-ingot": 2, "cured-hare-hide": 1})
    armor("Quill-Brass Greaves", "legs", 13, "legs_chain", HEAVY, heavy=True); recipe(c, "Quill-Brass Greaves", 22, FIRE, {"quill-brass-ingot": 3})
    armor("Quill-Brass Sabatons", "feet", 13, "feet_sabatons", HEAVY, heavy=True); recipe(c, "Quill-Brass Sabatons", 22, FIRE, {"quill-brass-ingot": 2, "cured-hare-hide": 1})
    # shell-steel (Heikegani Crustocrats)
    weapon("Shell-Steel Cutlass", "sword", 26, "sabre_01", SWORD, 224, {"vit": 2}, "A debt, collected: shell-steel keeps its edge and its grudges.")
    recipe(c, "Shell-Steel Cutlass", 34, FIRE, {"shell-steel-ingot": 2, "shell-glue": 1})
    weapon("Carapace Cleaver", "axe", 28, "axe_hand", AXE, 252, {"str": 3}, "Cleaves cleanly through interest.")
    recipe(c, "Carapace Cleaver", 36, FIRE, {"shell-steel-ingot": 3, "beetle-shell": 2})
    weapon("Shell Accountant Letter-Knife", "dagger", 25, "dagger_kali_01", DAGGER, 158, {"dex": 3, "accuracy": 4}, "It opens letters. It also opens accounts.")
    recipe(c, "Shell Accountant Letter-Knife", 32, FIRE, {"shell-steel-ingot": 1, "silver-ingot": 1})
    armor("Shell-Steel Cuirass", "body", 30, "body_plate_knight", HEAVY, {"vit": 4}, "Every plate carries a face. They carry their ancestors on their backs.", heavy=True)
    recipe(c, "Shell-Steel Cuirass", 40, FIRE, {"shell-steel-ingot": 5, "shell-glue": 2})
    armor("Shell-Steel Helm", "head", 28, "head_iron", HEAVY, {"vit": 2}, heavy=True); recipe(c, "Shell-Steel Helm", 36, FIRE, {"shell-steel-ingot": 3})
    armor("Shell-Steel Gauntlets", "hand-l", 28, "hands_knuckles", HEAVY, {"str": 2}, heavy=True); recipe(c, "Shell-Steel Gauntlets", 38, FIRE, {"shell-steel-ingot": 3, "cured-badger-hide": 1})
    armor("Shell-Steel Greaves", "legs", 29, "legs_chain", HEAVY, {"vit": 3}, heavy=True); recipe(c, "Shell-Steel Greaves", 38, FIRE, {"shell-steel-ingot": 4})
    armor("Shell-Steel Sabatons", "feet", 28, "feet_sabatons", HEAVY, {"agi": 2}, heavy=True); recipe(c, "Shell-Steel Sabatons", 38, FIRE, {"shell-steel-ingot": 3, "cured-badger-hide": 1})
    # swampite (Kuroshio bog-steel)
    weapon("Swampite Greatsword", "greatsword", 36, "greatsword_icewraith", GREAT, 400, {"str": 4}, "Black bog-steel; it never fully cools.")
    recipe(c, "Swampite Greatsword", 52, FIRE, {"swampite-ingot": 4, "spider-silk": 2})
    weapon("Swampite Tachi", "katana", 34, "katana_juzumaru", KATANA, 210, {"dex": 3}, "A tachi folded in marsh water.")
    recipe(c, "Swampite Tachi", 50, FIRE, {"swampite-ingot": 3, "peat": 2})
    armor("Swampite Plate", "body", 38, "body_plate_knight", HEAVY, {"vit": 5, "str": 2}, heavy=True); recipe(c, "Swampite Plate", 54, FIRE, {"swampite-ingot": 5, "cured-badger-hide": 2})
    # ward-steel (Jiangshi)
    weapon("Ward-Nail Dagger", "dagger", 55, "dagger_sunken_kali", DAGGER, 154, {"dex": 5, "accuracy": 8}, "A spike of ward-steel. It documents things permanently.")
    recipe(c, "Ward-Nail Dagger", 72, FIRE, {"ward-steel-ingot": 2, "ward-ink": 1})
    weapon("Talisman Blade", "sword", 58, "weapon_sword_keen_edge", SWORD, 220, {"int": 3, "mnd": 3}, "Blessed paper folded inside the steel.")
    recipe(c, "Talisman Blade", 76, FIRE, {"ward-steel-ingot": 3, "ward-paper": 2})
    armor("Ward-Steel Plate", "body", 60, "body_plate_knight", HEAVY, {"vit": 7, "str": 3}, "It cannot be filmed. The footage keeps coming out as static.", heavy=True)
    recipe(c, "Ward-Steel Plate", 82, FIRE, {"ward-steel-ingot": 6, "black-bar-shard": 2})
    armor("Ward-Steel Helm", "head", 56, "head_iron", HEAVY, {"vit": 3}, heavy=True); recipe(c, "Ward-Steel Helm", 76, FIRE, {"ward-steel-ingot": 3})

def goldsmith():
    c = "goldsmithing"
    acc("Copper Shell-Mark Ring", ["ringl", "ringr"], 4, {"str": 1, "defense": 2}, "ring_copper", desc="A copper ring stamped with a single shell-mark.")
    recipe(c, "Copper Shell-Mark Ring", 5, FIRE, {"copper-ingot": 1})
    acc("Brass Quill Pin", ["neck"], 9, {"int": 2, "mnd": 1}, "neck_choker_dawn", desc="A brass pin in the shape of a quill.")
    recipe(c, "Brass Quill Pin", 14, FIRE, {"quill-brass-ingot": 1, "sentry-quill": 1})
    acc("Silver Ledger Band", ["ringl", "ringr"], 16, {"mnd": 2, "magic_defense_bonus": 4}, "ring_silver", desc="Engraved with columns of very small numbers.")
    recipe(c, "Silver Ledger Band", 26, FIRE, {"silver-ingot": 1, "ink-vial": 1})
    acc("Hospitality Coin Pendant", ["neck"], 20, {"chr": 3, "evasion": 6}, "neck_priestess_charm", desc="Offer it with a meal and a kind word. The host is bound to be gracious.")
    recipe(c, "Hospitality Coin Pendant", 30, FIRE, {"silver-ingot": 2, "tanuki-rice": 1})
    acc("Tanuki Leaf Charm", ["neck"], 18, {"agi": 3, "evasion": 8}, "neck_priestess_charm", desc="A gilded leaf. Whatever it was a minute ago, it is a charm now.")
    recipe(c, "Tanuki Leaf Charm", 28, FIRE, {"silver-ingot": 1, "shrine-greens": 1})
    acc("Swampite Signet", ["ringl", "ringr"], 30, {"str": 3, "vit": 2, "attack": 6}, "ring_brass_01", desc="Warm to the touch.")
    recipe(c, "Swampite Signet", 42, FIRE, {"swampite-ingot": 1, "silver-ingot": 1})
    acc("Gold Debt Ring", ["ringl", "ringr"], 36, {"dex": 3, "accuracy": 8, "magic_attack_bonus": 4}, "ring_kings_alibi", desc="Fully paid. The ledger insists.")
    recipe(c, "Gold Debt Ring", 52, FIRE, {"gold-ingot": 1, "silver-ingot": 1})
    acc("Chrono-Coin Earring", ["earl", "earr"], 34, {"agi": 3, "haste": 2}, "earring_jordan_01", desc="Heads: 1620. Tails: 1901. It is always both.")
    recipe(c, "Chrono-Coin Earring", 46, FIRE, {"gold-ingot": 1, "wisp-glass": 1})
    acc("Stolas Hum Earring", ["earl", "earr"], 40, {"int": 3, "mnd": 3, "fast_cast": 3}, "ear_mage", desc="Tuned to 7.83 Hz. The hum is not a disguise; it is a frequency.")
    recipe(c, "Stolas Hum Earring", 58, FIRE, {"gold-ingot": 1, "lens-shard": 2})
    acc("Lens Pendant", ["neck"], 32, {"accuracy": 10, "ranged_accuracy": 8}, "neck_choker_dawn", desc="Ground by the Tor lensmen. Everything it shows you is looking back.")
    recipe(c, "Lens Pendant", 44, FIRE, {"silver-ingot": 1, "lens-shard": 2})
    acc("Protocol Seal Ring", ["ringl", "ringr"], 52, {"vit": 4, "defense": 10, "magic_defense_bonus": 8}, "ring_magic_a01", desc="Stamped: NO ACTION DERIVED FROM OBSERVATION.")
    recipe(c, "Protocol Seal Ring", 74, FIRE, {"gold-ingot": 1, "protocol-stone": 1})
    acc("Shell Parliament Signet", ["neck"], 46, {"chr": 5, "enmity": 4, "defense": 8}, "neck_choker_dawn", desc="Certifies, to three generations of accountant, that you are good for it.")
    recipe(c, "Shell Parliament Signet", 64, FIRE, {"gold-ingot": 2, "shell-glue": 2})

def woodworking():
    c = "woodworking"
    for tree, lumber, planks, d in [("Ash", "ash-lumber", "ash-planks", 2), ("Heather", "heather-lumber", "heather-planks", 14),
                                    ("Reed", "reed-lumber", "reed-planks", 18), ("Moor Oak", "moor-oak-lumber", "moor-oak-planks", 34),
                                    ("Ridge Pine", "ridge-pine-lumber", "ridge-pine-planks", 30)]:
        recipe(c, tree + " Planks", d, WIND, {lumber: 2}, hq=False)
    weapon("Fence-Post Staff", "staff", 4, "staff_brass", STAFF, 200, desc="Pulled from the Fledgling Field fence. A numbered owlet was displaced.")
    recipe(c, "Fence-Post Staff", 6, WIND, {"ash-planks": 2, "down-feather": 1})
    weapon("Wren Bow", "bow", 12, "bow_simple_01", BOW, 300, {"agi": 2}, "Strung for observation at a distance, per the Protocol.")
    recipe(c, "Wren Bow", 18, WIND, {"heather-planks": 2, "spider-silk": 2})
    weapon("Pellet Wand", "staff", 10, "wand_willow", STAFF, 190, {"int": 2}, "A wand of pressed pellet and reed.")
    recipe(c, "Pellet Wand", 16, WIND, {"reed-planks": 1, "pellet-bone": 2})
    weapon("Reed Flute", "instrument", 14, "banjo_a01", INSTR, 200, {"chr": 3}, "Marsh-reed hollowed to a particular grief.")
    recipe(c, "Reed Flute", 22, WIND, {"reed-planks": 2, "heron-quill": 1})
    weapon("Boom-Mic Staff", "staff", 22, "staff_shepherd_01", STAFF, 205, {"mnd": 3}, "A documentary crew's boom, fitted with a wooden head. It picks up everything.")
    recipe(c, "Boom-Mic Staff", 30, WIND, {"ridge-pine-planks": 2, "brass-plate-shard": 2})
    weapon("Tuning-Fork Staff", "staff", 32, "staff_healing", STAFF, 195, {"int": 4, "mnd": 4, "fast_cast": 3}, "Struck once, it sings 7.83 Hz until you stop listening.")
    recipe(c, "Tuning-Fork Staff", 44, WIND, {"moor-oak-planks": 2, "lens-shard": 2, "silver-ingot": 1})
    weapon("Eastwind Longbow", "bow", 30, "bow_composite", BOW, 310, {"agi": 3, "ranged_accuracy": 6}, "Drawn against the one wind that never changes.")
    recipe(c, "Eastwind Longbow", 40, WIND, {"ridge-pine-planks": 3, "snowfeather": 2, "spider-silk": 2})
    weapon("Footnote Rod", "staff", 42, "staff_brass", STAFF, 190, {"int": 5, "magic_attack_bonus": 6}, "Every inscription on it is a footnote to another.")
    recipe(c, "Footnote Rod", 60, WIND, {"moor-oak-planks": 3, "ink-vial": 2, "wisp-wax": 2})
    weapon("Protocol Gavel", "h2h", 50, "weapon_h2h_qitornado_01", H2H, 240, {"str": 4}, "The Cataloguer's gavel; it has never been used, which is why it hits so hard.")
    recipe(c, "Protocol Gavel", 70, WIND, {"moor-oak-planks": 2, "protocol-stone": 2, "shell-glue": 1})
    armor("Reed Hat", "head", 8, "armor_head_greenhat", ROBE + LIGHT, {"agi": 1}); recipe(c, "Reed Hat", 12, WIND, {"reed-planks": 1, "down-cloth": 1})
    armor("Pine Clogs", "feet", 20, "feet_rugged", ROBE + LIGHT, {"vit": 2}); recipe(c, "Pine Clogs", 28, WIND, {"ridge-pine-planks": 1, "cured-hare-hide": 1})

def leathercraft():
    c = "leathercraft"
    recipe(c, "Cured Wolf Pelt", 4, THUNDER, {"wolf-pelt": 2}, hq=False)
    recipe(c, "Cured Hare Hide", 8, THUNDER, {"hare-pelt": 2}, hq=False)
    recipe(c, "Cured Badger Hide", 18, THUNDER, {"hedge-hide": 2}, hq=False)
    recipe(c, "Rimefox Leather", 36, THUNDER, {"rime-pelt": 2}, hq=False)
    recipe(c, "Mini-Bike Leather", 22, THUNDER, {"cured-wolf-pelt": 2, "lamp-oil": 1}, hq=False)
    armor("Wolf-Hide Vest", "body", 6, "body_leather", LIGHT, {"agi": 1}); recipe(c, "Wolf-Hide Vest", 10, THUNDER, {"cured-wolf-pelt": 3})
    armor("Wolf-Hide Gloves", "hand-l", 5, "hands_leather", LIGHT); recipe(c, "Wolf-Hide Gloves", 8, THUNDER, {"cured-wolf-pelt": 1, "wolf-fang": 1})
    armor("Wolf-Hide Boots", "feet", 5, "feet_leather", LIGHT); recipe(c, "Wolf-Hide Boots", 8, THUNDER, {"cured-wolf-pelt": 2})
    armor("Hare-Hide Cap", "head", 9, "head_leather", LIGHT + ROBE, {"agi": 1}); recipe(c, "Hare-Hide Cap", 14, THUNDER, {"cured-hare-hide": 2})
    armor("Mini-Bike Jacket", "body", 16, "body_leather", LIGHT, {"agi": 3, "evasion": 6}, "Smells of gasoline and the hot breath of the Osaka store pump.")
    recipe(c, "Mini-Bike Jacket", 26, THUNDER, {"mini-bike-leather": 3, "iron-ingot": 1})
    armor("Mini-Bike Boots", "feet", 14, "feet_leather", LIGHT, {"agi": 2, "haste": 2}); recipe(c, "Mini-Bike Boots", 24, THUNDER, {"mini-bike-leather": 2})
    armor("Badger-Hide Coat", "body", 22, "body_rugged", LIGHT, {"vit": 3}); recipe(c, "Badger-Hide Coat", 32, THUNDER, {"cured-badger-hide": 4})
    armor("Badger-Hide Leggings", "legs", 21, "legs_rugged", LIGHT, {"vit": 2}); recipe(c, "Badger-Hide Leggings", 30, THUNDER, {"cured-badger-hide": 3})
    armor("Rimefox Hood", "head", 34, "head_leather", LIGHT + ROBE, {"int": 2, "magic_defense_bonus": 4}); recipe(c, "Rimefox Hood", 46, THUNDER, {"rimefox-leather": 2, "snowfeather": 1})
    armor("Rimefox Jerkin", "body", 36, "body_leather", LIGHT, {"agi": 4, "evasion": 8}); recipe(c, "Rimefox Jerkin", 50, THUNDER, {"rimefox-leather": 4})
    armor("Shade-Hide Cloak", "body", 40, "body_leather", LIGHT, {"agi": 5, "evasion": 10, "defense": 4}, "Cut from a roost-shade's wing. The light bends around it.")
    recipe(c, "Shade-Hide Cloak", 56, THUNDER, {"rimefox-leather": 2, "shade-feather": 3})

def clothcraft():
    c = "clothcraft"
    recipe(c, "Spider Silk Cloth", 8, WIND, {"spider-silk": 3}, hq=False)
    recipe(c, "Down Cloth", 10, WIND, {"down-feather": 4}, hq=False)
    recipe(c, "Vellum Weave", 40, WIND, {"vellum-scrap": 4, "ink-vial": 1}, hq=False)
    recipe(c, "Wraith Cloth", 55, WIND, {"wraith-thread": 3}, hq=False)
    armor("Fledgling Down Robe", "body", 5, "body_robe", ROBE, {"mnd": 1}); recipe(c, "Fledgling Down Robe", 8, WIND, {"down-cloth": 3})
    armor("Silk Habit", "body", 12, "robe_simple_01", ROBE, {"int": 2, "mnd": 2}, "Kuroshio silk: it catches nothing, not even the eye."); recipe(c, "Silk Habit", 16, WIND, {"spider-silk-cloth": 3})
    armor("Silk Slippers II", "feet", 10, "feet_silk", ROBE, {"agi": 1, "fast_cast": 2}); recipe(c, "Silk Slippers II", 14, WIND, {"spider-silk-cloth": 1, "down-cloth": 1})
    armor("Silk Cowl", "head", 11, "head_leather", ROBE, {"int": 1, "mnd": 1}); recipe(c, "Silk Cowl", 15, WIND, {"spider-silk-cloth": 2})
    armor("Shrine Tender Robe", "body", 24, "body_robe", ROBE, {"mnd": 4, "magic_defense_bonus": 6}, "Worn by those who tend the Yokai shrine gate. Hospitality is the only law."); recipe(c, "Shrine Tender Robe", 34, WIND, {"spider-silk-cloth": 4, "shrine-greens": 2})
    armor("Accountant Coat", "body", 26, "body_wizcoat", ROBE, {"int": 4, "chr": 2, "enmity": -4}, "The stitching is columnar. Everything balances."); recipe(c, "Accountant Coat", 36, WIND, {"spider-silk-cloth": 3, "ink-vial": 2})
    armor("Vellum Cowl", "head", 36, "head_leather", ROBE, {"int": 3, "magic_attack_bonus": 3}); recipe(c, "Vellum Cowl", 48, WIND, {"vellum-weave": 2})
    armor("Cataloguer Robe", "body", 44, "robe_dark", ROBE, {"int": 6, "mnd": 6, "magic_attack_bonus": 5}, "Seventeen footnotes are stitched into the hem. They are private."); recipe(c, "Cataloguer Robe", 62, WIND, {"vellum-weave": 4, "wisp-wax": 2})
    armor("Wraith-Cloth Mantle", "body", 52, "robe_crimson_mdl", ROBE, {"int": 7, "agi": 3, "evasion": 8}, "It moves when the air doesn't."); recipe(c, "Wraith-Cloth Mantle", 72, WIND, {"wraith-cloth": 4, "snowfeather": 2})
    acc("Down Cloak", ["back"], 8, {"vit": 1, "defense": 3}, "back_cloak", desc="Warm. Quietly unfiled."); recipe(c, "Down Cloak", 12, WIND, {"down-cloth": 2})
    acc("Ward-Paper Mantle", ["back"], 28, {"magic_defense_bonus": 8, "defense": 5}, "cape_yellow_01", desc="Talisman paper stitched to cloth, front and back."); recipe(c, "Ward-Paper Mantle", 40, WIND, {"ward-paper": 3, "spider-silk-cloth": 2})

def bonecraft():
    c = "bonecraft"
    recipe(c, "Bone Meal", 3, WATER, {"pellet-bone": 2}, hq=False)
    weapon("Owl-Bone Dagger", "dagger", 9, "dagger_basic", DAGGER, 158, {"dex": 2}, "Whittled from a pellet-bone; faintly labelled."); recipe(c, "Owl-Bone Dagger", 12, WATER, {"pellet-bone": 3, "down-feather": 1})
    weapon("Jawbone Axe", "axe", 14, "axe_hand", AXE, 246, {"str": 2}); recipe(c, "Jawbone Axe", 20, WATER, {"badger-claw": 2, "pellet-bone": 3})
    weapon("Mandible Claws", "h2h", 20, "h2h_heavy_claws_01", H2H, 230, {"dex": 3}); recipe(c, "Mandible Claws", 28, WATER, {"crawler-mandible": 3, "cured-badger-hide": 1})
    weapon("Wolf-Fang Knife", "dagger", 8, "dagger_basic", DAGGER, 156, {"attack": 3}); recipe(c, "Wolf-Fang Knife", 10, WATER, {"wolf-fang": 2, "wolf-pelt": 1})
    weapon("Sunderworm Fang Knife", "dagger", 30, "dagger_kali_01", DAGGER, 152, {"dex": 4, "str": 2}, "Pried from a head of the Sunderworm. It still wants to dig."); recipe(c, "Sunderworm Fang Knife", 42, WATER, {"sunderworm-fang": 1, "pellet-bone": 2})
    armor("Pellet-Bone Cap", "head", 12, "helm_bone_plate_mk1", HEAVY + LIGHT, {"vit": 1}); recipe(c, "Pellet-Bone Cap", 16, WATER, {"pellet-bone": 4, "bone-meal": 1})
    armor("Beetle-Shell Breastplate", "body", 16, "body_chain", HEAVY, {"vit": 3}, heavy=True); recipe(c, "Beetle-Shell Breastplate", 24, WATER, {"beetle-shell": 5, "shell-glue": 1})
    armor("Brine-Shell Mail", "body", 28, "body_plate_knight", HEAVY, {"vit": 4, "magic_defense_bonus": 6}, "Ichthyosapien shell-markings are still legible on the inside.", heavy=True); recipe(c, "Brine-Shell Mail", 38, WATER, {"beetle-shell": 4, "pellet-bone": 4, "brine-salt": 2})
    armor("Roost-Wing Gauntlets", "hand-l", 18, "bone_plate_gauntlets_m", LIGHT + HEAVY, {"dex": 2}); recipe(c, "Roost-Wing Gauntlets", 26, WATER, {"roost-wing": 3, "crawler-mandible": 1})
    armor("Eagle-Plume Crown", "head", 30, "head_vag_crown_a", ROBE + LIGHT, {"agi": 2, "accuracy": 8}); recipe(c, "Eagle-Plume Crown", 40, WATER, {"eagle-owl-plume": 3, "pellet-bone": 2})
    armor("Lizard-Bone Greaves", "legs", 14, "legs_rugged", HEAVY + LIGHT, {"agi": 2}); recipe(c, "Lizard-Bone Greaves", 20, WATER, {"lizard-tail": 2, "pellet-bone": 2})
    armor("Redacted Mask", "head", 48, "head_gladiator_bloodstained_01", HEAVY + LIGHT, {"str": 4, "crit_rate": 3}, "Black bars across the eyes. What it saw was struck from the record."); recipe(c, "Redacted Mask", 66, WATER, {"redacted-feather": 2, "black-bar-shard": 2, "bone-meal": 3})

def alchemy():
    c = "alchemy"
    recipe(c, "Ward Ink", 6, DARK, {"ink-vial": 1, "lamp-oil": 1}, yield_=2, hq=False)
    recipe(c, "Shell Glue", 5, WATER, {"beetle-shell": 1, "slime-oil": 1}, yield_=2, hq=False)
    recipe(c, "Ward Paper", 16, DARK, {"vellum-scrap": 2, "ward-ink": 1}, yield_=2, hq=False)
    cons("Shell-Salve", "A viscous shell-resin salve. Closes wounds, and files a note about them.")
    recipe(c, "Shell-Salve", 8, WATER, {"slime-oil": 1, "shell-glue": 1}, yield_=2, hq=False)
    cons("Hospitality Draught", "Offered with a smile. You owe the host nothing; the host owes you everything.")
    recipe(c, "Hospitality Draught", 24, WATER, {"spring-water": 1, "shrine-greens": 1, "leech-blood": 1}, hq=False)
    cons("Eastwind Tonic", "Cold, clean, and faintly bureaucratic. Restores MP.", model="item_ether")
    recipe(c, "Eastwind Tonic", 14, WATER, {"spring-water": 1, "sentry-quill": 1}, hq=False)
    cons("Vellum Ether", "Ink-black and sharp. A full measure of mana.", model="item_ether")
    recipe(c, "Vellum Ether", 38, DARK, {"spring-water": 1, "ink-vial": 1, "vellum-scrap": 1}, hq=False)
    cons("Owl Tea", "Pellet-leaf tea. Recovers a little of everything.")
    recipe(c, "Owl Tea", 10, WATER, {"spring-water": 1, "down-feather": 1}, hq=False)
    cons("Brine Elixir", "Ichthyosapien brine, distilled. Restores a great deal of HP and MP.", model="item_hipotion")
    recipe(c, "Brine Elixir", 56, WATER, {"spring-water": 1, "brine-salt": 2, "pellet-bone": 1, "wisp-glass": 1}, hq=False)
    cons("Ward-Ink Tonic", "Drinks like a signature. Purges ailments.")
    recipe(c, "Ward-Ink Tonic", 20, DARK, {"ward-ink": 1, "spring-water": 1}, hq=False)
    # ASN / thief reagents (thrown or applied)
    cons("Venom Flask", "A glass flask of concentrated venom. Coats your weapon for several hits.", stack=12)
    recipe(c, "Venom Flask", 12, DARK, {"spider-venom": 1, "slime-oil": 1}, hq=False)
    cons("Numbing Resin", "Grey resin that deadens whatever it touches. Coats your weapon.", stack=12)
    recipe(c, "Numbing Resin", 22, DARK, {"slug-ichor": 1, "shell-glue": 1}, hq=False)
    cons("Sleeping Powder", "Poppy dust in a paper twist. Thrown, it puts nearby foes to sleep.", stack=12)
    recipe(c, "Sleeping Powder", 30, DARK, {"moth-dust": 2, "guano-clump": 1}, hq=False)
    cons("Smoke Pellet", "Burst it and you are somewhere else. Blinds foes and hides you.", stack=12)
    recipe(c, "Smoke Pellet", 18, DARK, {"moth-dust": 1, "lamp-oil": 1}, hq=False)
    cons("Party Store Firework", "Sold behind the counter at the Osaka store. Area fire damage.", stack=12)
    recipe(c, "Party Store Firework", 26, FIRE, {"lamp-oil": 1, "moth-dust": 1, "ward-paper": 1}, hq=False)
    cons("Acid Flask Refill", "Corrosive tincture. Poisons and weakens when thrown.", stack=12)
    recipe(c, "Acid Flask Refill", 34, DARK, {"spider-venom": 1, "slug-ichor": 2}, hq=False)
    cons("Spirit Lantern", "A paper lantern lit with a wisp's wax. Party regen while lit.", stack=12)
    recipe(c, "Spirit Lantern", 44, LIGHTC, {"wisp-wax": 2, "lamp-wick": 1, "ward-paper": 1}, hq=False)

FOODS = []  # (id, name, stats dict, minutes, desc)

def food(name, desc, stats, minutes):
    sl = cons(name, desc, model="item_food_minestrone", stack=12)
    FOODS.append((sl, name, stats, minutes))
    return sl

def cooking():
    c = "cooking"
    recipe(c, "Dough", 2, FIRE, {"party-store-flour": 1, "spring-water": 1}, yield_=2, hq=False)
    recipe(c, "Stock", 6, FIRE, {"fish-carp": 1, "brine-salt": 1}, yield_=2, hq=False)
    food("Party Store Hot Dog", "From the roller grill. Time has not touched it. Neither has hunger.", {"STR": 2, "HP": 20}, 20)
    recipe(c, "Party Store Hot Dog", 4, FIRE, {"dough": 1, "stock": 1}, hq=False)
    food("Tanuki Onigiri", "Rice with a surprise in the middle. The surprise is a leaf.", {"VIT": 2, "HP": 30}, 30)
    recipe(c, "Tanuki Onigiri", 8, FIRE, {"tanuki-rice": 1, "fish-fullmoon-sardine": 1}, yield_=2, hq=False)
    food("Carp Udon", "Thick noodles in carp stock. Warm in a way that is hard to file.", {"VIT": 3, "MND": 2, "HP": 40}, 30)
    recipe(c, "Carp Udon", 16, FIRE, {"dough": 1, "stock": 1, "shrine-greens": 1}, hq=False)
    food("Bass Teriyaki", "A glazed bass on rice.", {"STR": 4, "DEX": 2}, 30)
    recipe(c, "Bass Teriyaki", 20, FIRE, {"fish-bass": 1, "tanuki-rice": 1, "brine-salt": 1}, hq=False)
    food("Trout Meuniere", "Pan-fried trout. The Eastwind quartermaster approves.", {"DEX": 4, "INT": 2}, 30)
    recipe(c, "Trout Meuniere", 24, FIRE, {"fish-trout": 1, "party-store-flour": 1, "brine-salt": 1}, hq=False)
    food("Crab Ramen", "Ramen with a whole crab in it. The crab is a Heikegani cousin. Do not ask.", {"VIT": 4, "HP": 60}, 45)
    recipe(c, "Crab Ramen", 28, FIRE, {"dough": 1, "stock": 1, "fish-crab": 1}, hq=False)
    food("Eel Skewers", "Grilled bog eel on a stick. Earthy.", {"STR": 3, "VIT": 3, "HP": 40}, 30)
    recipe(c, "Eel Skewers", 32, FIRE, {"fish-eel": 2, "brine-salt": 1}, hq=False)
    food("Mud Crab Bisque", "Thick, dark, and quietly nourishing.", {"MND": 4, "MP": 50}, 30)
    recipe(c, "Mud Crab Bisque", 38, FIRE, {"fish-mud-crab": 1, "stock": 1, "party-store-flour": 1}, hq=False)
    food("Bog Eel Stew", "A stew the swamp is proud of.", {"STR": 5, "VIT": 5, "HP": 80}, 45)
    recipe(c, "Bog Eel Stew", 46, FIRE, {"fish-bog-eel": 1, "stock": 1, "peat": 1}, hq=False)
    food("Crimson Trout Roast", "Crimson trout roasted over lamp-oil flame.", {"DEX": 6, "AGI": 3}, 45)
    recipe(c, "Crimson Trout Roast", 52, FIRE, {"fish-crimson-trout": 1, "lamp-oil": 1, "brine-salt": 1}, hq=False)
    food("Squid Ink Pasta", "Black as a redacted line. Sharpens the mind.", {"INT": 7, "MP": 80}, 45)
    recipe(c, "Squid Ink Pasta", 62, FIRE, {"fish-giant-squid": 1, "dough": 1, "ink-vial": 1}, hq=False)
    food("Hospitality Feast", "A full meal offered in good faith. The host is bound to your wellbeing.", {"STR": 4, "VIT": 4, "MND": 4, "HP": 100}, 60)
    recipe(c, "Hospitality Feast", 72, FIRE, {"tanuki-rice": 2, "fish-trophy-bass": 1, "shrine-greens": 2, "brine-salt": 1}, hq=False)
    food("Stolas Soup", "Hums faintly at 7.83 Hz. You will see slightly further ahead.", {"INT": 6, "MND": 6, "MP": 120}, 60)
    recipe(c, "Stolas Soup", 84, FIRE, {"stock": 2, "wisp-glass": 1, "shrine-greens": 1, "fish-giant-squid": 1}, hq=False)

# ---------------------------------------------------------------- vendors
VENDOR_STOCK = {
    # npc id -> [(item slug, price)]
    "quartermaster": [("fire-crystal", 120), ("water-crystal", 120), ("wind-crystal", 120), ("earth-crystal", 120),
                      ("ice-crystal", 120), ("thunder-crystal", 120), ("ash-lumber", 60), ("party-store-flour", 25),
                      ("tanuki-rice", 30), ("brine-salt", 20), ("spring-water", 15), ("shrine-greens", 35)],
    "wren-9": [("heather-lumber", 90), ("lamp-wick", 40), ("light-crystal", 180), ("spring-water", 15)],
    "pellet-reader-moss": [("reed-lumber", 100), ("spring-water", 15), ("dark-crystal", 180)],
    "roost-keeper-orla": [("dark-crystal", 180), ("lamp-wick", 40)],
    "eastwind-quartermaster-sable": [("ridge-pine-lumber", 140), ("ice-crystal", 130), ("thunder-crystal", 130)],
    "moor-footnoter-hester": [("moor-oak-lumber", 170), ("dark-crystal", 180), ("light-crystal", 180)],
    "stacks-archivist-quill": [("ink-vial", 90), ("lamp-wick", 40)],
}

def main():
    load_stable()
    build_materials()
    build_crystals()
    build_raw()
    blacksmith(); goldsmith(); woodworking(); leathercraft(); clothcraft(); bonecraft(); alchemy(); cooking()

    # items.json
    existing = json.load(open(P("data/items.json")))
    existing = [i for i in existing if not (OWN_LO <= i["id"] < OWN_HI)]
    have_names = {i["name"] for i in existing}
    # Dark Crystal might already exist; never duplicate a name outside our range
    items_out = [i for i in items if i["name"] not in have_names]
    json.dump(existing + items_out, open(P("data/items.json"), "w"), indent=2, ensure_ascii=False)
    open(P("data/items.json"), "a").write("\n")

    # recipes.json (ids unique)
    seen = set()
    for r in recipes:
        assert r["id"] not in seen, r["id"]
        seen.add(r["id"])
    json.dump(recipes, open(P("data/recipes.json"), "w"), indent=2, ensure_ascii=False)
    open(P("data/recipes.json"), "a").write("\n")

    # Go: vendor stock + foods
    g = ["// Code generated by scripts/gen_lore_items.py; DO NOT EDIT.\n", "package main\n", 'import "dragonsnshit/server/food"\n',
         "// This file sorts after owl_content_gen.go so these appends run after the vendor tables exist.", "func init() {"]
    for npc, stock in VENDOR_STOCK.items():
        for s, price in stock:
            g.append(f'\tnpcVendorCatalog["{npc}"] = append(npcVendorCatalog["{npc}"], VendorItem{{ID: "{s}", Price: {price}}})')
    g.append("}\n")
    g.append("// loreFoods are the cooked dishes; registered into the food registry at world init.")
    g.append("var loreFoods = []food.Food{")
    for sl, name, st, mins in FOODS:
        fields = ", ".join(f"{k}Bonus: {v}" for k, v in sorted(st.items()))
        g.append(f'\t{{ID: "{sl}", Name: "{name}", {fields}, Duration: {mins} * time.Minute}},')
    g.append("}\n")
    src = "\n".join(g).replace('import "dragonsnshit/server/food"', 'import (\n\t"time"\n\n\t"dragonsnshit/server/food"\n)')
    open(P("apps2/mud/zz_craft_content_gen.go"), "w").write(src)
    print("items", len(items_out), "recipes", len(recipes), "foods", len(FOODS))

if __name__ == "__main__":
    main()
