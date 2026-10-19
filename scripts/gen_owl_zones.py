#!/usr/bin/env python3
"""Generate the Eastwind Owl zone cluster (zones 10-18).

Single source of truth for: zone/mob/NM/NPC/quest/shop content. Emits
  - data/items.json, mob_drops.json, mob_variants.json, mob_spawns.json (merged in place,
    idempotent: rows this script owns are replaced, everything else is left alone) -- the same
    files the IDUNA admin tools (/admin/gfd-items, gfd-mob-drops, gfd-mob-spawns) edit, so the
    content stays tunable there afterwards.
  - server/mob/owl_gen.go, server/quest/owl_quests_gen.go, apps2/mud/owl_content_gen.go

Run from the repo root:  python3 scripts/gen_owl_zones.py && gofmt -w <the three .go files>
"""
import json, math, os, re

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
def P(*a): return os.path.join(ROOT, *a)

def slug(name): return name.lower().replace(" ", "-")

MELEE = ["WAR", "MNK", "THF", "BST", "NIN", "DNC", "RNG"]
CASTER = ["WHM", "BLM", "RDM", "SMN", "SCH"]
SWORD_JOBS = ["WAR", "PLD", "DRK", "RDM", "NIN"]
DAGGER_JOBS = ["WAR", "MNK", "RDM", "THF", "BST", "NIN", "DNC"]
STAFF_JOBS = ["WHM", "BLM", "RDM", "SMN", "SCH"]

# ---- shop tiers (basic gear, sold by zone vendors) -------------------------------------------
TIERS = [  # prefix, level
    ("Fledgling", 5), ("Observer", 12), ("Fieldwing", 20), ("Archivist", 30), ("Protocol", 40),
]

def shop_items(tier, base_id):
    prefix, L = TIERS[tier]
    items, i = [], base_id
    def add(name, cat, slots, jobs, stats, model, **kw):
        nonlocal i
        d = {"id": i, "name": name, "category": cat, "equip_slots": slots, "level": L,
             "stats": stats, "stack_size": 1, "model_id": model}
        if jobs: d["jobs"] = jobs
        d.update(kw); items.append(d); i += 1
    atk = round(10 + L * 1.6)
    add(f"{prefix} Blade", "weapon", ["main", "off"], SWORD_JOBS, {"attack": atk, "str": 1 + L // 10},
        "sword_steel" if tier > 1 else "sword_iron_short", weapon_type="sword", delay=228 + tier * 6)
    add(f"{prefix} Talon", "weapon", ["main", "off"], DAGGER_JOBS, {"attack": round(atk * .8), "dex": 1 + L // 10},
        "dagger_basic" if tier < 2 else "dagger_003", weapon_type="dagger", delay=168 + tier * 3)
    add(f"{prefix} Perch", "weapon", ["main"], STAFF_JOBS, {"magic_attack_bonus": round(3 + L * .7), "mnd": 1 + L // 8},
        "staff_brass", weapon_type="staff", delay=198 + tier * 4)
    d = 5 + round(L * .75)
    add(f"{prefix} Jerkin", "armor", ["body"], MELEE, {"defense": d + 4, "vit": 1 + L // 12}, "body_rugged" if tier < 2 else "body_chain")
    add(f"{prefix} Robe", "armor", ["body"], CASTER, {"defense": d, "mnd": 1 + L // 10, "int": 1 + L // 10}, "body_robe")
    add(f"{prefix} Leggings", "armor", ["legs"], None, {"defense": d - 1, "agi": L // 14}, "legs_rugged" if tier < 2 else "legs_chain")
    add(f"{prefix} Mitts", "armor", ["hand-l", "hand-r"], None, {"defense": max(2, d - 4), "dex": L // 14}, "hands_rugged")
    add(f"{prefix} Boots", "armor", ["feet"], None, {"defense": max(2, d - 4), "agi": L // 14}, "feet_rugged")
    add(f"{prefix} Hood", "armor", ["head"], None, {"defense": max(2, d - 3), "vit": L // 14}, "head_rugged")
    return items

# ---- zones -------------------------------------------------------------------------------------
# mobs: kind, hp, dmg, speed, aggro, leash, melee, swing_ms, count, ring, drops[(name, chance)]
def M(kind, hp, dmg, speed, aggro, count, ring, drops, swing=3000):
    return dict(kind=kind, hp=hp, dmg=dmg, speed=speed, aggro=aggro, leash=max(20, aggro * 2 + 10),
                melee=2.0, swing=swing, count=count, ring=ring, drops=drops)

# nm: id, kind(display), ph (placeholder kind), hp, dmg, aggro, speed, chance, open_s, close_s,
#     respawn_min, drops [(item name, chance)] -- items defined in NM_ITEMS
def N(id, kind, ph, hp, dmg, aggro, speed, chance, wo, wc, resp, items):
    return dict(id=id, kind=kind, ph=ph, hp=hp, dmg=dmg, aggro=aggro, speed=speed, chance=chance,
                wo=wo, wc=wc, resp=resp, items=items)

ZONES = [
 dict(id=10, name="Fledgling Field", lv=(3, 8), tier=0, spawn=(-3, 2), bounds=(40, 40),
  exits={"east": 0, "north": 14}, branch=(0, "west"),
  desc="[EASTWIND: FLEDGLING FIELD / Eastwind Owls / Pasture] Trampled pasture ringed by fence posts, each topped with a numbered owlet in training. Chalk grids are scratched across the turf; fledglings practise Observation on the field mice. Nobody approaches anything. The Protocol is painted on a board by the gate.",
  smell="Warm barley dust, mouse nest, and a faint chalky sourness -- the smell of things being watched for the first time.",
  npc=("warden-pip", "Fledgling Warden Pip", "Observe. Do not approach. Do not approach *yet* -- the Protocol is very clear on yet. Mind the mice."),
  mobs=[M("field-mouse", 60, 5, 3.0, 0, 6, 16, [("Mouse Whisker", .6), ("Owl Pellet", .35)]),
        M("fledgling-owlet", 95, 9, 3.4, 9, 5, 26, [("Down Feather", .6), ("Owl Pellet", .3)]),
        M("dust-moth", 130, 13, 4.2, 12, 3, 35, [("Moth Dust", .6), ("Down Feather", .25)])],
  variants=[("field-mouse", "Overfed Mouse", 1.8), ("fledgling-owlet", "Brooding Owlet", 1.5)],
  nms=[N("nm-brood-mother-strix", "Brood Mother Strix", "fledgling-owlet", 520, 24, 14, 3.8, .25, 240, 1500, 45, ["Strix Talon", "Downy Mantle"]),
       N("nm-old-mousewright", "Old Mousewright", "field-mouse", 430, 20, 10, 3.2, .3, 180, 1200, 40, ["Mousewright Band", "Whisker Charm"])],
  quests=[("owl-10-pellets", "First Pellets", "Warden Pip needs 4 Owl Pellets for the fledgling syllabus -- 'proof you can look at something without touching it.'", {"owl-pellet": 4}, {}, 120, "fledgling-hood"),
          ("owl-10-mousewright", "The Old Mousewright", "A mouse older than the fence posts keeps unfiling the filing. Defeat the Old Mousewright.", {}, {"Old Mousewright": 1}, 260, "fledgling-blade")]),
 dict(id=11, name="Wren's Perch", lv=(8, 14), tier=1, spawn=(0, -3), bounds=(45, 45),
  exits={"south": 1, "north": 15}, branch=(1, "north"),
  desc="[EASTWIND: WREN'S PERCH / Eastwind Owls / Upland] A tilted tableland of heather and surveyor's cairns. Each cairn carries a brass plate: FIELD OBSERVER DESIGNATE WREN, and a number. Several plates have been scratched out. The wind carries nothing but the sound of someone writing.",
  smell="Cold heather, brass polished by a thumb, and wet wool. Under it, ink -- fresh, which is wrong for a place nobody visits.",
  npc=("wren-9", "Field Observer Designate WREN-9", "WREN-3 filed a complaint against himself in 1623 and never withdrew it. I am WREN-9. I have withdrawn nothing, because I have filed nothing. Buy a blade."),
  mobs=[M("wren-sentry", 170, 16, 3.0, 12, 6, 15, [("Brass Plate Shard", .6), ("Sentry Quill", .4)]),
        M("banded-kestrel", 210, 21, 5.4, 18, 5, 26, [("Kestrel Pinion", .6), ("Sentry Quill", .25)]),
        M("hedge-badger", 270, 26, 3.6, 14, 3, 35, [("Badger Claw", .6), ("Hedge Hide", .4)])],
  variants=[("wren-sentry", "Sentry of Record", 1.8), ("banded-kestrel", "Dire Kestrel", 1.5)],
  nms=[N("nm-wren-3-unretired", "WREN-3 Unretired", "wren-sentry", 1200, 38, 16, 3.6, .22, 300, 1800, 60, ["Drafted Complaint Blade", "Wren Ledger Staff"]),
       N("nm-hedgemaw", "Hedgemaw", "hedge-badger", 1050, 42, 12, 3.8, .25, 300, 1800, 60, ["Hedgemaw Hide Vest", "Burrower Belt"])],
  quests=[("owl-11-plates", "Plates Scratched Out", "Collect 5 Brass Plate Shards so WREN-9 can reconcile which Wrens still exist.", {"brass-plate-shard": 5}, {}, 220, "observer-hood"),
          ("owl-11-wren3", "Withdraw the Complaint", "WREN-3 has been dead for 300 years and is still filing. Put the Unretired to rest.", {}, {"WREN-3 Unretired": 1}, 500, "observer-blade")]),
 dict(id=13, name="Pellet Marsh", lv=(10, 16), tier=1, spawn=(0, 3), bounds=(45, 45),
  exits={"west": 3, "north": 17}, branch=(3, "east"),
  desc="[EASTWIND: PELLET MARSH / Eastwind Owls / Wetland] A flat, pewter marsh where the owls drop what they cannot digest. Every hummock is a compacted pellet the size of a cart, studded with small bones, labelled in a careful hand. Herons stand in the reeds like clerks.",
  smell="Peat water, regurgitated fur, and something faintly like old filing glue. Not rot -- processing.",
  npc=("pellet-reader-moss", "Pellet Reader Moss", "I read what the Owls couldn't keep. It's all in the pellets: the records, the bones, the arguments. Bring me bones and I'll read you something worth paying for."),
  mobs=[M("bog-vole", 210, 17, 4.0, 10, 6, 15, [("Vole Pelt", .6), ("Pellet Bone", .4)]),
        M("pellet-slug", 270, 23, 2.6, 8, 5, 25, [("Pellet Bone", .6), ("Slug Ichor", .35)]),
        M("marsh-heron", 340, 29, 4.6, 16, 3, 35, [("Heron Quill", .6), ("Pellet Bone", .3)])],
  variants=[("bog-vole", "Greedy Vole", 1.8), ("pellet-slug", "Bloated Pellet Slug", 1.5)],
  nms=[N("nm-pellet-colossus", "Pellet Colossus", "pellet-slug", 1400, 40, 10, 2.8, .22, 300, 1800, 60, ["Pellet Plate Cap", "Compacted Ring"]),
       N("nm-heron-of-record", "Heron of Record", "marsh-heron", 1250, 46, 18, 4.8, .22, 300, 1800, 60, ["Heron Quill Bow", "Waders of Record"])],
  quests=[("owl-13-bones", "Bones for the Reader", "Bring Pellet Reader Moss 6 Pellet Bones.", {"pellet-bone": 6}, {}, 260, "observer-boots"),
          ("owl-13-heron", "The Heron Who Kept Count", "The Heron of Record has been tallying visitors. Un-tally it.", {}, {"Heron of Record": 1}, 560, "observer-perch")]),
 dict(id=12, name="Roost Deeps", lv=(12, 18), tier=2, spawn=(0, 3), bounds=(25, 50),
  exits={"north": 2, "down": 16}, branch=(2, "south"),
  desc="[EASTWIND: ROOST DEEPS / Eastwind Owls / Underground] The caves widen into a cathedral of perches: stalactites hung with ropes, ledges white with old guano, and a hush that is the owls holding their breath. Lantern-moths drift. Down the east wall a stair is cut into the stone.",
  smell="Ammonia, cold stone, and feather-oil. A dry, sharp warmth -- the smell of a place where many things sleep during the day.",
  npc=("roost-keeper-orla", "Roost-Keeper Orla", "Mind your voice. The day-sleepers are at their perches and the night-watch is not at its best. If you must fight, fight downwind."),
  mobs=[M("roost-bat", 270, 21, 5.0, 14, 6, 14, [("Roost Wing", .6), ("Guano Clump", .4)]),
        M("guano-crawler", 340, 27, 2.8, 10, 5, 24, [("Guano Clump", .6), ("Crawler Mandible", .35)]),
        M("barn-owl-shade", 420, 34, 5.2, 20, 3, 34, [("Shade Feather", .6), ("Roost Wing", .25)])],
  variants=[("roost-bat", "Elder Roost Bat", 1.8), ("guano-crawler", "Corpulent Crawler", 1.5)],
  nms=[N("nm-guano-prince", "Guano Prince", "guano-crawler", 1700, 50, 12, 3.2, .22, 300, 1800, 65, ["Prince Slick Gauntlets", "Guano Tooth"]),
       N("nm-unlit-roost-mother", "Unlit Roost-Mother", "barn-owl-shade", 1900, 58, 22, 5.0, .2, 300, 1800, 70, ["Nightroost Katana", "Shade Wing Cape"])],
  quests=[("owl-12-wings", "Quiet the Night-Watch", "Roost-Keeper Orla needs 6 Roost Wings to muffle the stair.", {"roost-wing": 6}, {}, 300, "fieldwing-mitts"),
          ("owl-12-mother", "The Lights Went Out", "The Unlit Roost-Mother has put out the lanterns. Relight the Deeps by felling her.", {}, {"Unlit Roost-Mother": 1}, 700, "fieldwing-talon")]),
 dict(id=14, name="Observation Tor", lv=(20, 28), tier=2, spawn=(0, -3), bounds=(45, 45),
  exits={"south": 10, "east": 15, "north": 18}, branch=None,
  desc="[EASTWIND: OBSERVATION TOR / Eastwind Owls / Summit] A bald granite tor crowned with brass telescopes bolted into the rock, every one trained on a different horizon. Owls the size of cattle sit between them, eyes like lamps. The telescopes swing slowly. All of them are pointing at you now.",
  smell="Cold lens-glass, sun-warmed granite, and the clean ozone of attention. The air smells like being looked at.",
  npc=("tor-lensman-aldous", "Tor Lensman Aldous", "Every glass up here is pointed at something. Most of it doesn't want to be. Do you want the north glass or the west? Either way: buy a hood."),
  mobs=[M("tor-eagle-owl", 560, 38, 4.6, 20, 6, 15, [("Eagle Owl Plume", .6), ("Lens Shard", .35)]),
        M("lens-wisp", 480, 46, 5.0, 24, 5, 25, [("Lens Shard", .6), ("Wisp Glass", .4)]),
        M("tor-rimefox", 700, 54, 5.6, 22, 3, 35, [("Rime Pelt", .6), ("Lens Shard", .25)])],
  variants=[("tor-eagle-owl", "Greater Eagle Owl", 1.8), ("lens-wisp", "Focused Wisp", 1.5)],
  nms=[N("nm-the-lensmaster", "The Lensmaster", "lens-wisp", 2600, 68, 26, 5.0, .2, 400, 2100, 75, ["Lensmaster Monocle", "Prismatic Ring"]),
       N("nm-parliament-without-eyes", "Parliament Without Eyes", "tor-eagle-owl", 3000, 72, 22, 4.4, .2, 400, 2100, 80, ["Blind Parliament Robe", "Eyeless Dagger"])],
  quests=[("owl-14-lens", "Re-Grind the Glass", "Tor Lensman Aldous needs 6 Lens Shards to repair the north telescope.", {"lens-shard": 6}, {}, 450, "fieldwing-hood"),
          ("owl-14-parliament", "A Parliament Without Eyes", "The Parliament sits in session with no one to see it. Dissolve it.", {}, {"Parliament Without Eyes": 1}, 1000, "fieldwing-blade")]),
 dict(id=15, name="Eastwind Ridge", lv=(18, 26), tier=2, spawn=(0, 3), bounds=(45, 45),
  exits={"south": 11, "west": 14, "up": 18}, branch=None,
  desc="[EASTWIND: EASTWIND RIDGE / Eastwind Owls / Alpine] The long windward ridge the Owls are named for. A wind that has never once changed direction scours the rock. Snow-pale owls ride it motionless, wings out, reading the weather like a ledger.",
  smell="Thin cold air, sleet, and a clean mineral edge. The wind brings the smell of everywhere east of here, all at once, none of it yours.",
  npc=("eastwind-quartermaster-sable", "Eastwind Quartermaster Sable", "The wind comes from one place. Always. We named ourselves for it. Supplies are on the left; questions are on the right, filed in triplicate."),
  mobs=[M("sleet-hare", 480, 34, 5.6, 0, 6, 15, [("Hare Pelt", .6), ("Sleet Crystal", .35)]),
        M("ridge-snowy-owl", 600, 44, 5.0, 22, 5, 25, [("Snowfeather", .6), ("Sleet Crystal", .3)]),
        M("windwraith", 740, 54, 6.0, 26, 3, 35, [("Wraith Thread", .6), ("Sleet Crystal", .3)])],
  variants=[("sleet-hare", "Blizzard Hare", 1.8), ("ridge-snowy-owl", "Matriarch's Chick", 1.5)],
  nms=[N("nm-eastwind-matriarch", "Eastwind Matriarch", "ridge-snowy-owl", 2800, 70, 24, 5.2, .2, 400, 2100, 80, ["Matriarch Plume Mantle", "Snowfeather Sabatons"]),
       N("nm-gale-apprentice-cataloguer", "Gale Apprentice-Cataloguer", "windwraith", 2500, 76, 28, 6.0, .2, 400, 2100, 80, ["Gale Index Staff", "Windbound Sash"])],
  quests=[("owl-15-snowfeathers", "Plumage for the Quartermaster", "Bring Eastwind Quartermaster Sable 6 Snowfeathers.", {"snowfeather": 6}, {}, 420, "fieldwing-boots"),
          ("owl-15-matriarch", "Challenge the Matriarch", "The Eastwind Matriarch refuses to change direction. Make her.", {}, {"Eastwind Matriarch": 1}, 950, "fieldwing-perch")]),
 dict(id=16, name="Cataloguer's Stacks", lv=(24, 32), tier=3, spawn=(0, 0), bounds=(40, 40),
  exits={"up": 12, "east": 17}, branch=None,
  desc="[EASTWIND: CATALOGUER'S STACKS / Eastwind Owls / Archive] Shelves taller than the roost, row on row, lit by hanging lamps that never flicker. Everything is shelved. Everything is labelled. Some of the labels are watching you read them. One aisle is marked in pencil: FOOTNOTES (PRIVATE).",
  smell="Vellum, binder's paste, and lamp oil. Beneath it, a thin sharp note of something being filed away that was not ready to be.",
  npc=("stacks-archivist-quill", "Stacks Archivist Quill", "Silence is not required. It is merely observed. Quills, vellum, and the occasional bloodied index card are all for sale. Do not touch the footnotes."),
  mobs=[M("paper-moth", 620, 44, 5.2, 22, 6, 14, [("Vellum Scrap", .6), ("Moth Dust", .25)]),
        M("ink-mite-swarm", 700, 54, 4.6, 24, 5, 24, [("Ink Vial", .6), ("Vellum Scrap", .3)]),
        M("stack-warden", 950, 68, 3.8, 28, 3, 34, [("Binder's Paste", .6), ("Vellum Scrap", .35)])],
  variants=[("paper-moth", "Archive Moth", 1.8), ("ink-mite-swarm", "Indelible Swarm", 1.5)],
  nms=[N("nm-the-overdue-tome", "The Overdue Tome", "stack-warden", 3800, 84, 26, 4.2, .2, 500, 2400, 90, ["Overdue Greatsword", "Bookbinder Gloves"]),
       N("nm-footnote-seventeen", "Footnote Seventeen", "ink-mite-swarm", 3300, 80, 28, 5.0, .2, 500, 2400, 90, ["Footnote Ring", "Marginalia Cloak"])],
  quests=[("owl-16-vellum", "Reshelving", "Stacks Archivist Quill needs 8 Vellum Scraps to rebind a torn index.", {"vellum-scrap": 8}, {}, 650, "archivist-mitts"),
          ("owl-16-tome", "Return the Overdue Tome", "Three centuries overdue. The fine is you.", {}, {"The Overdue Tome": 1}, 1400, "archivist-blade")]),
 dict(id=17, name="Footnote Moor", lv=(22, 30), tier=3, spawn=(0, 3), bounds=(45, 45),
  exits={"west": 16, "south": 13, "east": 18}, branch=None,
  desc="[EASTWIND: FOOTNOTE MOOR / Eastwind Owls / Moorland] A rain-dark moor where the Cataloguer's marginal notes wander off. Numbered standing stones lean in the mist, each carved with a tiny superscript. Will-o-the-lamps hover over bog pools, murmuring the beginnings of sentences.",
  smell="Peat smoke, wet slate, and the sweet scorch of a candle someone has just snuffed out. The air tastes like an unsent apology.",
  npc=("moor-footnoter-hester", "Moor Footnoter Hester", "Each stone is a footnote the Cataloguer didn't dare put in the main text. You may read them. You may not act on them. The Protocol is firm on that, and I am firmer."),
  mobs=[M("footnote-wisp", 560, 46, 4.6, 22, 6, 14, [("Wisp Wax", .6), ("Ink Vial", .25)]),
        M("moor-hag-owl", 700, 56, 5.0, 24, 5, 24, [("Hag Feather", .6), ("Wisp Wax", .3)]),
        M("bog-lamp", 880, 66, 3.6, 28, 3, 34, [("Wisp Wax", .5), ("Lamp Oil", .6)])],
  variants=[("footnote-wisp", "Bitter Footnote", 1.8), ("moor-hag-owl", "Hag of the Fourth Stone", 1.5)],
  nms=[N("nm-draft-complaint", "Draft Complaint", "moor-hag-owl", 3400, 82, 26, 5.0, .2, 500, 2400, 90, ["Seven Drafts Blade", "Unsent Ring"]),
       N("nm-moor-cataloguer", "Moor Cataloguer", "footnote-wisp", 3600, 86, 28, 4.8, .2, 500, 2400, 95, ["Moor Cataloguer Coat", "Wisp Lantern Charm"])],
  quests=[("owl-17-wax", "Candles for the Stones", "Moor Footnoter Hester needs 8 Wisp Wax to keep the stones lit.", {"wisp-wax": 8}, {}, 600, "archivist-boots"),
          ("owl-17-complaint", "Send the Complaint", "The Draft Complaint has been unsent for seven drafts. Defeat it so the record can close.", {}, {"Draft Complaint": 1}, 1300, "archivist-perch")]),
 dict(id=18, name="Protocol Wall", lv=(36, 45), tier=4, spawn=(0, -3), bounds=(40, 40),
  exits={"south": 14, "down": 15, "west": 17, "east": 203}, branch=(203, "west"),
  desc="[EASTWIND: THE PROTOCOL WALL / Eastwind Owls / Boundary] A curtain wall of dressed stone, forty feet high, carved edge to edge with the Cairngorms Protocol in an unbroken hand. 'No inference beyond the evidence. No action derived from observation.' Gaps in the text have been blacked out. Beyond a heavy gate to the east, the Archive.",
  smell="Cold dressed stone, iron gate-oil, and a sealed-envelope smell with something underneath it like a held breath. A very long time has passed here, carefully.",
  npc=("protocol-clerk-vael", "Protocol Clerk Vael", "The Protocol prohibits action derived from observation. I observe that you are armed. I will file that. Nothing further. ...The shop is open."),
  mobs=[M("protocol-sentinel", 1250, 82, 4.2, 26, 5, 15, [("Protocol Stone", .6), ("Black Bar Shard", .3)]),
        M("redacted-owl", 1500, 96, 5.4, 30, 4, 25, [("Black Bar Shard", .6), ("Redacted Feather", .4)]),
        M("unfiled-archivist", 1800, 112, 4.6, 32, 3, 34, [("Protocol Stone", .5), ("Ink Vial", .5)])],
  variants=[("protocol-sentinel", "Sentinel Prime", 1.8), ("redacted-owl", "Fully Redacted Owl", 1.5)],
  nms=[N("nm-the-cataloguer", "The Cataloguer", "unfiled-archivist", 7500, 130, 30, 4.8, .16, 600, 2700, 120, ["Seventeen Footnote Staff", "Insufferable Choker"]),
       N("nm-redaction-itself", "Redaction Itself", "redacted-owl", 7000, 140, 32, 5.6, .16, 600, 2700, 120, ["Redacted Edge", "Black Bar Mantle"])],
  quests=[("owl-18-stone", "Weight of the Wall", "Protocol Clerk Vael requires 8 Protocol Stones to repair a breach in the text.", {"protocol-stone": 8}, {}, 1600, "protocol-mitts"),
          ("owl-18-cataloguer", "Insufferable", "The Cataloguer has footnoted you seventeen times. Respond in the only way the Protocol does not forbid.", {}, {"The Cataloguer": 1}, 3500, "protocol-blade"),
          ("owl-18-redaction", "Strike the Redaction", "Something has been blacked out that should not have been. Defeat Redaction Itself.", {}, {"Redaction Itself": 1}, 3000, "protocol-perch")]),
]

# ---- NM-only exotic items ----------------------------------------------------------------------
def X(name, cat, slots, lv, stats, model, jobs=None, wtype=None, delay=None, desc=None, ex=False):
    d = {"name": name, "category": cat, "equip_slots": slots, "level": lv, "stats": stats,
         "stack_size": 1, "model_id": model, "flags": ["rare", "ex"] if ex else ["rare"]}
    if jobs: d["jobs"] = jobs
    if wtype: d["weapon_type"] = wtype
    if delay: d["delay"] = delay
    if desc: d["description"] = desc
    return d

NM_ITEMS = [
 X("Strix Talon", "weapon", ["main", "off"], 8, {"attack": 24, "dex": 3, "accuracy": 4}, "dagger_kali_01", DAGGER_JOBS, "dagger", 160, "Shed by the Brood Mother during a very quiet moult."),
 X("Downy Mantle", "accessory", ["back"], 6, {"agi": 3, "defense": 5, "evasion": 4}, "cape_yellow_01"),
 X("Mousewright Band", "accessory", ["ringl", "ringr"], 7, {"str": 1, "dex": 2, "accuracy": 5}, "ring_copper"),
 X("Whisker Charm", "accessory", ["neck"], 6, {"agi": 2, "evasion": 6}, "neck_priestess_charm"),
 X("Drafted Complaint Blade", "weapon", ["main", "off"], 14, {"attack": 40, "str": 3, "accuracy": 6}, "sabre_01", SWORD_JOBS, "sword", 222, "Edge honed on 300 years of unsubmitted paperwork."),
 X("Wren Ledger Staff", "weapon", ["main"], 13, {"magic_attack_bonus": 16, "mnd": 3, "int": 2}, "staff_healing", STAFF_JOBS, "staff", 192),
 X("Hedgemaw Hide Vest", "armor", ["body"], 13, {"defense": 30, "vit": 4, "str": 2}, "body_leather", MELEE),
 X("Burrower Belt", "accessory", ["waist"], 12, {"vit": 3, "defense": 4, "haste": 2}, "waist_belt_life"),
 X("Pellet Plate Cap", "armor", ["head"], 15, {"defense": 16, "vit": 3, "mnd": 1}, "helm_bone_plate_mk1"),
 X("Compacted Ring", "accessory", ["ringl", "ringr"], 14, {"vit": 2, "defense": 4, "str": 2}, "ring_brass_01"),
 X("Heron Quill Bow", "weapon", ["main"], 16, {"attack": 38, "agi": 3, "accuracy": 8}, "bow_gorgon_01", ["WAR", "RNG", "NIN"], "bow", 300),
 X("Waders of Record", "armor", ["feet"], 15, {"defense": 14, "agi": 3, "evasion": 5}, "boots_winter_01"),
 X("Prince Slick Gauntlets", "armor", ["hand-l", "hand-r"], 17, {"defense": 15, "dex": 3, "attack": 6}, "bone_plate_gauntlets_m"),
 X("Guano Tooth", "accessory", ["earl", "earr"], 16, {"str": 2, "attack": 6}, "ear_spike"),
 X("Nightroost Katana", "weapon", ["main"], 18, {"attack": 52, "dex": 3, "evasion": 4}, "katana_juzumaru", ["SAM", "NIN"], "katana", 210, "Forged in a room with no light, for a hand that needed none."),
 X("Shade Wing Cape", "accessory", ["back"], 18, {"agi": 4, "evasion": 8, "defense": 6}, "threads_of_missing_gods_mantle"),
 X("Lensmaster Monocle", "armor", ["head"], 26, {"defense": 18, "int": 5, "mnd": 5, "accuracy": 8}, "head_icarus_crown", CASTER),
 X("Prismatic Ring", "accessory", ["ringl", "ringr"], 25, {"int": 4, "magic_attack_bonus": 6}, "ring_magic_a01"),
 X("Blind Parliament Robe", "armor", ["body"], 27, {"defense": 28, "mnd": 6, "int": 6}, "robe_dark", CASTER),
 X("Eyeless Dagger", "weapon", ["main", "off"], 26, {"attack": 60, "dex": 5, "evasion": 6}, "dagger_sunken_kali", DAGGER_JOBS, "dagger", 156),
 X("Matriarch Plume Mantle", "accessory", ["back"], 24, {"agi": 5, "defense": 8, "evasion": 8}, "cape_yellow_01"),
 X("Snowfeather Sabatons", "armor", ["feet"], 23, {"defense": 20, "agi": 4, "evasion": 6}, "feet_sabatons"),
 X("Gale Index Staff", "weapon", ["main"], 25, {"magic_attack_bonus": 28, "int": 4, "mnd": 4}, "staff_shepherd_01", STAFF_JOBS, "staff", 186),
 X("Windbound Sash", "accessory", ["waist"], 24, {"agi": 5, "evasion": 10, "haste": 3}, "waist_sash", ["NIN", "THF", "DNC"]),
 X("Overdue Greatsword", "weapon", ["main"], 31, {"attack": 92, "str": 6, "accuracy": 6}, "greatsword_icewraith", ["WAR", "DRK"], "greatsword", 420, "Three centuries overdue. The fine is you.", True),
 X("Bookbinder Gloves", "armor", ["hand-l", "hand-r"], 30, {"defense": 22, "dex": 5, "attack": 8}, "glove_faithful_01", None, None, None, None, True),
 X("Footnote Ring", "accessory", ["ringl", "ringr"], 29, {"int": 5, "mnd": 5, "magic_attack_bonus": 8}, "raven_ring_a", None, None, None, None, True),
 X("Marginalia Cloak", "accessory", ["back"], 30, {"agi": 6, "evasion": 12, "defense": 10}, "threads_of_missing_gods_mantle", None, None, None, None, True),
 X("Seven Drafts Blade", "weapon", ["main", "off"], 29, {"attack": 74, "str": 5, "accuracy": 8}, "weapon_sword_keen_edge", SWORD_JOBS, "sword", 216, "Revised seven times. Never submitted.", True),
 X("Unsent Ring", "accessory", ["ringl", "ringr"], 27, {"cha": 4, "mnd": 4, "evasion": 6}, "ring_glass_01", None, None, None, None, True),
 X("Moor Cataloguer Coat", "armor", ["body"], 30, {"defense": 34, "mnd": 7, "int": 7}, "robe_crimson_mdl", CASTER, None, None, None, True),
 X("Wisp Lantern Charm", "accessory", ["neck"], 28, {"int": 4, "mnd": 4, "evasion": 8}, "neck_choker_dawn", None, None, None, None, True),
 X("Seventeen Footnote Staff", "weapon", ["main"], 45, {"magic_attack_bonus": 60, "int": 10, "mnd": 10}, "staff_shepherd_01", STAFF_JOBS, "staff", 180, "Every footnote is private. Every footnote is also a weapon.", True),
 X("Insufferable Choker", "accessory", ["neck"], 44, {"str": 6, "dex": 6, "vit": 6, "agi": 6, "int": 6, "mnd": 6, "defense": 12}, "enigma_curse_neck", None, None, None, "Documented as insufferable in seventeen footnotes.", True),
 X("Redacted Edge", "weapon", ["main", "off"], 44, {"attack": 128, "str": 9, "accuracy": 12}, "ws_remorse_dark", SWORD_JOBS, "sword", 210, "The blade, in the record, is blacked out.", True),
 X("Black Bar Mantle", "accessory", ["back"], 43, {"agi": 9, "evasion": 20, "defense": 18}, "threads_of_missing_gods_mantle", None, None, None, None, True),
]
NM_ITEM_BY_NAME = {x["name"]: x for x in NM_ITEMS}

# ---- helpers -------------------------------------------------------------------------------------
def load(path): return json.load(open(P(path)))
def dump(path, obj):
    with open(P(path), "w") as f:
        json.dump(obj, f, indent=2, ensure_ascii=False); f.write("\n")

def gostr(s): return json.dumps(s, ensure_ascii=False)

def main():
    # ---- items ----
    items = load("data/items.json")
    owned = lambda i: 1100 <= i["id"] < 1400
    items = [i for i in items if not owned(i)]
    new_items, nid = [], 1100
    shop_by_tier = []
    for t in range(len(TIERS)):
        rows = shop_items(t, nid); nid += len(rows); new_items += rows; shop_by_tier.append(rows)
    nid = 1300
    for x in NM_ITEMS:
        x["id"] = nid; nid += 1
    new_items += NM_ITEMS
    items += new_items
    dump("data/items.json", items)

    # ---- drops ----
    drops = load("data/mob_drops.json")
    owned_kinds = set()
    for z in ZONES:
        for m in z["mobs"]: owned_kinds.add(m["kind"])
        for n in z["nms"]: owned_kinds.add(n["kind"].lower())
    drops = [d for d in drops if d["kind"].lower() not in owned_kinds]
    for z in ZONES:
        for m in z["mobs"]:
            drops.append({"kind": m["kind"], "items": [
                {"id": slug(n), "name": n, "drop_chance": c} for n, c in m["drops"]]})
        for n in z["nms"]:
            its = [{"id": slug(i), "name": i, "drop_chance": .45} for i in n["items"]]
            its.append({"id": "nm-owl-token", "name": "Owl Token", "drop_chance": 1.0})
            drops.append({"kind": n["kind"], "items": its})
    dump("data/mob_drops.json", drops)

    # ---- variants ----
    variants = load("data/mob_variants.json")
    vkinds = {v[0] for z in ZONES for v in z["variants"]}
    variants = [v for v in variants if v["base_kind"] not in vkinds]
    for z in ZONES:
        for base, name, mul in z["variants"]:
            variants.append({"base_kind": base, "display_name": name, "power_mul": mul})
    dump("data/mob_variants.json", variants)

    # ---- spawns (with NM blocks) ----
    spawns = load("data/mob_spawns.json")
    zids = {z["id"] for z in ZONES}
    spawns = [s for s in spawns if s["zone_id"] not in zids]
    for z in ZONES:
        nm_by_ph = {n["ph"]: n for n in z["nms"]}
        for m in z["mobs"]:
            row = {"zone_id": z["id"], "kind": m["kind"], "enabled": True}
            n = nm_by_ph.get(m["kind"])
            if n:
                row["nm"] = {"id": n["id"], "spawn_chance": n["chance"], "window_open_sec": n["wo"],
                             "window_close_sec": n["wc"], "respawn_minutes": n["resp"]}
            spawns.append(row)
    dump("data/mob_spawns.json", spawns)

    # ---- Go: server/mob/owl_gen.go ----
    g = ["// Code generated by scripts/gen_owl_zones.py; DO NOT EDIT.\n", "package mob\n"]
    g.append("// OwlZoneMobs: Eastwind Owl zone cluster (zones 10-18) base mob tables, keyed by zone ID.")
    g.append("var OwlZoneMobs = map[int][]OwlMobDef{")
    for z in ZONES:
        g.append(f"\t{z['id']}: {{")
        for m in z["mobs"]:
            g.append(f"\t\t{{Kind: {gostr(m['kind'])}, HP: {m['hp']}, Damage: {m['dmg']}, MoveSpeed: {m['speed']}, AggroRange: {float(m['aggro'])}, LeashRange: {float(m['leash'])}, MeleeRange: {m['melee']}, SwingMS: {m['swing']}, Count: {m['count']}, Ring: {float(m['ring'])}}},")
        g.append("\t},")
    g.append("}\n")
    g.append("// OwlNMDefs: notorious monsters for the Eastwind Owl zones.")
    g.append("var OwlNMDefs = []OwlNMDef{")
    for z in ZONES:
        for k, n in enumerate(z["nms"]):
            x = 8 * (1 if k == 0 else -1)
            g.append(f"\t{{ID: {gostr(n['id'])}, Kind: {gostr(n['kind'])}, ZoneID: {z['id']}, HP: {n['hp']}, Damage: {n['dmg']}, AggroRange: {float(n['aggro'])}, LeashRange: {float(n['aggro']*2+20)}, MeleeRange: 3.0, MoveSpeed: {n['speed']}, SwingMS: 2200, X: {float(x*3)}, Z: {float(-z['bounds'][1]//2)}}},")
    g.append("}\n")
    open(P("server/mob/owl_gen.go"), "w").write("\n".join(g))

    # ---- Go: server/quest/owl_quests_gen.go ----
    q = ["// Code generated by scripts/gen_owl_zones.py; DO NOT EDIT.\n", "package quest\n",
         "// OwlZoneQuests are the Eastwind Owl cluster (zones 10-18) quests.",
         "var OwlZoneQuests = []*Quest{"]
    for z in ZONES:
        for qid, title, desc, items_, kills, flow, reward in z["quests"]:
            ri = ", ".join(f"{gostr(k)}: {v}" for k, v in items_.items())
            rk = ", ".join(f"{gostr(k)}: {v}" for k, v in kills.items())
            q.append(f"\t{{ID: {gostr(qid)}, Title: {gostr(title)}, Desc: {gostr(desc)}, GiverNPCID: {gostr(z['npc'][0])}, "
                     f"RequireItems: map[string]int{{{ri}}}, RequireKills: map[string]int{{{rk}}}, RewardFlow: {flow}, RewardItem: {gostr(reward)}}},")
    q.append("}\n")
    open(P("server/quest/owl_quests_gen.go"), "w").write("\n".join(q))

    # ---- Go: apps2/mud/owl_content_gen.go ----
    c = ["// Code generated by scripts/gen_owl_zones.py; DO NOT EDIT.\n", "package main\n",
         'import "dragonsnshit/server/fame"\n',
         "// owlZoneNames: display names for the Owl cluster, used when registering the zones.",
         "var owlZoneNames = map[int]string{"]
    for z in ZONES: c.append(f"\t{z['id']}: {gostr(z['name'])},")
    c += ["}\n", "// init wires the Eastwind Owl cluster (zones 10-18): exits, descriptions, smells,",
          "// NPCs, vendor catalogs and item display names.", "func init() {"]
    # exits
    for z in ZONES:
        c.append(f"\texits[{z['id']}] = map[string]int{{{', '.join(f'{gostr(d)}: {t}' for d, t in z['exits'].items())}}}")
    for z in ZONES:
        if z["branch"]:
            src, d = z["branch"]
            c.append(f"\texits[{src}][{gostr(d)}] = {z['id']}")
    for z in ZONES:
        full = f"{z['desc']} (Recommended levels {z['lv'][0]}-{z['lv'][1]}.)"
        c.append(f"\tzoneDesc[{z['id']}] = {gostr(full)}")
        c.append(f"\tzoneSmell[{z['id']}] = {gostr(z['smell'])}")
    for z in ZONES:
        nid_, nname, greet = z["npc"]
        c.append(f"\tnpcs = append(npcs, npcDef{{ID: {gostr(nid_)}, Name: {gostr(nname)}, ZoneID: {z['id']}, Greeting: {gostr(greet)}, FameNation: fame.Neutral}})")
        tier = z["tier"]
        rows = shop_by_tier[tier]
        prices = []
        base = 40 + 50 * (tier + 1) ** 2
        cat = [("echo-drop", 50), ("antidote", 80), ("hi-potion", 250)]
        for it in rows:
            mult = {"weapon": 3.0, "armor": 2.0}[it["category"]] * (1.2 if it["equip_slots"][0] == "body" else 1.0)
            prices.append((slug(it["name"]), int(base * mult)))
        allp = cat + prices
        c.append(f"\tnpcVendorCatalog[{gostr(nid_)}] = []VendorItem{{")
        for s, pr in allp: c.append(f"\t\t{{ID: {gostr(s)}, Price: {pr}}},")
        c.append("\t}")
    names = {}
    for it in new_items: names[slug(it["name"])] = it["name"]
    for z in ZONES:
        for m in z["mobs"]:
            for n, _ in m["drops"]: names[slug(n)] = n
    names["nm-owl-token"] = "Owl Token"
    for k in sorted(names): c.append(f"\titemDisplayName[{gostr(k)}] = {gostr(names[k])}")
    c.append("}\n")
    open(P("apps2/mud/owl_content_gen.go"), "w").write("\n".join(c))
    print("zones", len(ZONES), "items", len(new_items), "nms", sum(len(z['nms']) for z in ZONES))

if __name__ == "__main__":
    main()
