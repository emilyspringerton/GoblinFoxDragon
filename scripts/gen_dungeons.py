#!/usr/bin/env python3
"""Generate the multi-room dungeons.

Each of the 8 roster dungeons (data/dungeon_roster.json) becomes a chain of 6 rooms:

    0 entrance (zone 208+i, safe)  -N-> 1 hall -N-> 2 crossroads -E-> 3 side chamber (elite)
                                                         \\-N-> 4 antechamber (elite) -N-> 5 boss hall

Rooms 1-5 are zones 300 + i*10 + room. Mobs scale with a per-dungeon level band, bosses and elites
come from the dungeon roster (so /admin/gfd-dungeon-roster in IDUNA keeps working) and are shown
under their TYLER compendium character names. Drops use weapons from data/items.json.

Writes: server/mob/dungeon_rooms_gen.go, apps2/mud/dungeon_content_gen.go,
        data/mob_drops.json + data/mob_spawns.json rows (owned rows replaced).
Run from the repo root:  python3 scripts/gen_dungeons.py && gofmt -w <the two .go files>
"""
import json, os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
def P(*a): return os.path.join(ROOT, *a)
def slug(n): return n.lower().replace(" ", "-")
def gostr(s): return json.dumps(s, ensure_ascii=False)

# roster hero id -> TYLER compendium character name (what players see)
HEROES = {
    "ARENA_HERO_CART": "The Retrieval Cart", "ARENA_HERO_NOOR1": "NOOR-1, Four Days Behind",
    "ARENA_HERO_PAIMON": "Paimon, the Court Voice", "ARENA_HERO_VASSAGO": "Vassago, the Soft Foresight",
    "ARENA_HERO_BELETH": "Beleth, the Detonation", "ARENA_HERO_ZAGAN": "Zagan, the Standstill's Confessor",
    "ARENA_HERO_DOC_WHEEL": "Buer, the Wheel-Physician", "ARENA_HERO_GUNNR": "Gunnr, Who Argued With a Raven",
    "ARENA_HERO_LOKI": "Loki, Who Isn't Here", "ARENA_HERO_GARY": "Gary, Bifrost Security",
    "ARENA_HERO_COURIER": "Ratatoskr's Debt-Collector", "ARENA_HERO_HE_XIANGU": "He Xiangu, Who Stopped Eating",
    "ARENA_HERO_WEATHERMAN": "Ao Guang's Weather-Debt Collector", "ARENA_HERO_FLUTE_DEBT": "Han Xiangzi's Flute-Debt",
    "ARENA_HERO_MORRIGAN": "The Morrigan, Once Refused", "ARENA_HERO_DAGDA": "The Dagda",
    "ARENA_HERO_UNICORN": "The Unicorn, Allegedly a Robot", "ARENA_HERO_GHOST": "The Ghost, Self-Identified as Alien",
    "ARENA_HERO_FROG": "The Frog, Keeper of a Secret", "ARENA_HERO_TREE": "The Tree, Speaking French",
    "ARENA_HERO_DUCK": "A Duck, Reportedly Telekinetic", "ARENA_HERO_CAIN": "Cain, East of Eden",
    "ARENA_HERO_ABRAHAM": "Abraham of Worms, the Mage", "ARENA_HERO_ADA": "Ada Lovelace, Pilot",
    "ARENA_HERO_MNM": "MnM, the Shapeshifting Crab", "ARENA_HERO_WARRIOR": "The Proving Warden",
}

# Weapon pools (names must exist in data/items.json). low = minion drops, mid = elite drops,
# top = boss drops (best first).
POOLS = [
    dict(low=["Iron Shortsword", "Hand Axe", "Brass Staff"], mid=["Broadsword", "Sabre", "Beestinger"],
         top=["Swift Blade", "Composite Bow", "Paltry Bastard Sword"]),
    dict(low=["Sabre", "Composite Bow", "Broadsword"], mid=["Swift Blade", "Paltry Bastard Sword", "Beestinger"],
         top=["Steel Sword", "Hidden Dagger of Kali", "Banjo"]),
    dict(low=["Swift Blade", "Composite Bow", "Hidden Dagger of Kali"], mid=["Steel Sword", "Banjo", "Hidden Dagger of Kali"],
         top=["Sunken Dagger of Kali", "Death's Gambit", "Steel Sword"]),
    dict(low=["Steel Sword", "Sunken Dagger of Kali", "Banjo"], mid=["Death's Gambit", "Healing Staff", "Qi Tornado"],
         top=["Keen Edge", "Titanium Claws", "Heavy Claws"]),
    dict(low=["Death's Gambit", "Healing Staff", "Titanium Claws"], mid=["Keen Edge", "Bow of the Gorgon Outcast", "Utzu Mah Turi"],
         top=["Mythril Sword", "Light's End", "Fourth Remorse"]),
    dict(low=["Keen Edge", "Utzu Mah Turi", "Bow of the Gorgon Outcast"], mid=["Fourth Remorse", "Light's End", "Shepherd's Staff"],
         top=["Icewraith Claymore", "Fourth Remorse", "Shepherd's Staff"]),
    dict(low=["Fourth Remorse", "Light's End", "Shepherd's Staff"], mid=["Icewraith Claymore", "Mythril Sword", "Shepherd's Staff"],
         top=["Last Regret", "Icewraith Claymore", "Fourth Remorse"]),
    dict(low=["Icewraith Claymore", "Mythril Sword", "Light's End"], mid=["Last Regret", "Fourth Remorse", "Shepherd's Staff"],
         top=["Mikazuki", "Juzumaru", "Onimaru"]),
]

# rooms: (name, desc, smell) x 6; minions: three kinds (id, flavor)
DUNGEONS = [
 dict(mats=["vellum-scrap", "ink-vial"], minions=["sealed-clerk", "ink-wraith", "box-mimic"], default_elite="the-unfiled-clerk",
  rooms=[
   ("Archive Antechamber", "A cold vestibule of numbered pigeonholes. A brass plate by the door reads: PLEASE TAKE A NUMBER. Nobody has called a number in four hundred years.", "Cold vellum and a long-sealed envelope. Nothing here has been opened on purpose."),
   ("Hall of Sealed Boxes", "Shelves climb into the dark, every one crowded with identical grey boxes, each wrapped in a seal that has never been broken and is faintly warm.", "Wax, dust, and the inside of a drawer that has outlived its owner."),
   ("Reading Room of Redactions", "Long tables under green lamps. Every page left open has had its important lines struck through in a single confident black bar.", "Lamp oil and ink, and the sharp, clean smell of something being un-said."),
   ("The Unfiled Annex", "A side room of things that belong nowhere: a stopped clock, a single glove, a coin that is two dates at once.", "Mothballs, copper, and a faint note of somebody else's evening."),
   ("Stacks of Returned Things", "Aisle after aisle of items marked RETURN TO SENDER. The senders are all deceased. The returns keep arriving.", "Cardboard, brown paper, and a stale wind from a delivery that never stopped."),
   ("The Delivery Dock", "A loading dock at the archive's heart. A wheeled cart sits at the end of the rails, piled with things nobody ordered, humming at the pitch of a held breath.", "Machine oil, old rope, and the smell of something that has just arrived."),
  ]),
 dict(mats=["lens-shard", "wisp-wax"], minions=["tuning-fork-wraith", "standing-wave", "goetic-echo"], default_elite="the-seventy-first-chair",
  rooms=[
   ("Tuning Hall", "A circular room with one long note hanging in the air. Every surface has been struck, once, by someone who wanted to hear what it said.", "Hot brass and the clean ozone of a note held too long."),
   ("Hall of Seventy-Two Chairs", "Seventy-two high-backed chairs ring a table of black stone. Seventy-one are occupied by empty robes. They turn, slightly, to watch you walk.", "Beeswax, old velvet, and the dry sweetness of a summoning that went well."),
   ("The Phase Chamber", "Two standing waves cross in the middle of the floor and cancel; where they cross there is a perfect, unnerving silence.", "Static, clean glass, and the weightless smell of a held frequency."),
   ("Beleth's Detonation Gallery", "A long gallery of cracked mirrors, each reflecting a love triangle at the moment before it broke.", "Burnt sugar, rose water and the sharp edge of an argument."),
   ("Zagan's Confessional", "A narrow booth of black wood. Somewhere inside a voice has been mid-sentence for forty-seven minutes.", "Candle smoke and the thick, patient air of a monologue."),
   ("The Court of Paimon", "A high court under a tented ceiling of brass trumpets. Two hundred legions stand at attention in the dark. A voice, sweet and enormous, asks you to state your business.", "Incense, cymbal-brass and the heavy sweetness of royal attention."),
  ]),
 dict(mats=["pellet-bone", "hedge-hide"], minions=["draugr-remnant", "raven-scribe", "hall-thrall"], default_elite="a-very-patient-draugr",
  rooms=[
   ("Mead Hall Gate", "Heavy doors carved with a wolf that is eating its own argument. Inside, a long fire has burned down to the memory of heat.", "Peat smoke, spilled mead and wet fur drying by a fire."),
   ("Longhouse of Remnants", "A long hall of benches and half-finished feasts. What is left of the guests stand against the walls, still holding their cups.", "Stale ale, tallow and cold iron."),
   ("The Raven Loft", "Roof beams crowded with ravens, each holding a small scroll of everything it has ever overheard.", "Feathers, parchment and cold air through the rafters."),
   ("Bifrost Watch", "A narrow stone gallery above the hall where the off-duty guard keeps the shift Heimdall does not want.", "Cold stone, lamp oil and a thermos of something strong."),
   ("The Root of Ratatoskr's Run", "A tunnel of knotted roots. Something small and furious has been running insults up and down it since before the roots grew.", "Damp earth, sap and the sour tang of a recently delivered insult."),
   ("The High Seat", "At the end of the hall a high seat of grey oak. A woman sits in it with a raven on her shoulder, correcting it.", "Pine smoke, iron and the stubborn freshness of someone who was right."),
  ]),
 dict(mats=["spider-silk", "lizard-tail"], minions=["jade-sentinel", "peach-thief", "hopping-corpse"], default_elite="a-guest-of-the-wall",
  rooms=[
   ("Wall Gate", "A gate of red lacquer in a wall that continues, unbroken, in both directions out of sight. Beyond it, someone is not eating.", "Red lacquer, cold jade and dry grass."),
   ("Courtyard of Peaches", "A garden of peach trees, every fruit at the exact moment of ripeness, none touched. Something is guarding them very politely.", "Ripe peach, stone dust and incense."),
   ("The Storm Pavilion", "A pavilion roofed with a small, localized sky. It is owed rain, and has been collecting on the debt.", "Wet slate, ozone and the heavy smell of weather owed."),
   ("Flute Terrace", "A terrace where every wrong note ever played nearby has settled into the stone as a faint, insistent hum.", "Bamboo, lacquer and a faintly sour chord."),
   ("The Hungry Bridge", "A narrow bridge over mist. The railings are carved with feasts. The planks are carved with abstinence.", "Cold mist, jade and the clean emptiness of a long fast."),
   ("The Fast of He Xiangu", "A bare pavilion at the end of the wall. A woman sits perfectly still on an empty table, immortal on nothing at all.", "Clean air, lotus and absolutely no food."),
  ]),
 dict(mats=["down-feather", "wolf-pelt"], minions=["wake-mourner", "highland-kelpie", "cairn-wight"], default_elite="a-crow-in-the-corner",
  rooms=[
   ("Cairn Path", "A path of stacked stones climbing a heathered ridge. Every cairn is a promise somebody did not keep.", "Wet heather, cold stone and peat smoke on a far wind."),
   ("The Wake Hall", "A low hall where the wake has not ended. Mourners sit along the walls, patient, with cups gone cold.", "Whisky, tallow and rain on wool."),
   ("Crow Fen", "A black bog where crows stand in rows, silent, awaiting a verdict.", "Bog water, wet feathers and iron."),
   ("The Dagda's Cauldron Room", "A huge, battered cauldron that is never empty and never clean. A big man in a tunic too short for him is stirring it, and does not appear to mind how he looks.", "Porridge, smoke and a stew that is somehow, aggressively, all the foods."),
   ("The Standing Stones", "A circle of old stones humming a low note. Between two of them hangs a doorway of black feathers.", "Cold granite, crow-wing and the ozone of an old promise."),
   ("The Morrigan's Refusal", "A hall of black stone. A tall woman stands with her back to you, weapon lowered, having just been told no, and still deciding what to do about it.", "Wet iron, crow feathers and a quiet, bitter patience."),
  ]),
 dict(mats=["moth-dust", "down-feather"], minions=["spaghetti-wisp", "top-hat-toad", "telekinetic-mallard"], default_elite="a-second-duck",
  rooms=[
   ("Foyer of Spaghetti", "An elegant foyer in which every table is set for a meal that is, quite specifically, spaghetti. None of it has been eaten. All of it is still hot.", "Tomato, basil and the warm steam of a meal in progress."),
   ("The Piano Room", "A grand piano plays itself. The pianist, translucent and pale green, pauses to explain that he is an alien, not a ghost, and resumes.", "Rosin, lemon polish and old velvet."),
   ("The Pond Room", "A round pond under a skylight. A frog in a top hat sits on a lily pad, guarding a secret it has politely declined to share.", "Pond water, lily and a hint of top hat felt."),
   ("The Greenhouse", "Glass walls, humid air, and one very large tree that is, with great dignity, speaking French.", "Wet earth, green leaves and bergamot."),
   ("The Duck Pond", "A pond of perfectly still water. A duck floats very slightly above it, telekinetically, looking pleased.", "Cold water, breadcrumbs and static electricity."),
   ("The Founders' Table", "A table laid for six, though only a unicorn sits at it, eating spaghetti and explaining, kindly, that it is a robot.", "Warm pasta, ozone and, faintly, hot circuitry."),
  ]),
 dict(mats=["vellum-scrap", "lens-shard"], minions=["difference-engine", "worm-scribe", "unmarked-wanderer"], default_elite="a-mark-bearer",
  rooms=[
   ("Gate of the Unbound", "A gate left open on purpose. Beyond it the wing belongs to anyone who has ever been told they did not belong.", "Fresh air, fresh paint and a draught from several different elsewheres."),
   ("The Babbage Hall", "Ranked brass engines tick in the dark, running a program that was written for a machine that was never built.", "Hot brass, oil and the faint dry crackle of punched cards."),
   ("The Wormhole Library", "A library whose shelves bend through each other. A thin mage in a tattered coat is cataloguing books that have not been written yet.", "Old paper, ozone and the green tang of pages from elsewhere."),
   ("The Detroit Alley", "A rain-dark alley of a city that is not this one. A large crab in a hoodie shape-shifts nervously beside a dumpster, claiming to be local.", "Wet asphalt, fryer oil and brine, unmistakably."),
   ("The Mark Corridor", "A corridor of white doors, each marked with a single small cross at eye height. Behind each is a field.", "Turned earth, iron and a long, long walk."),
   ("Cain's Field", "A wide, quiet field east of anywhere. A man works it alone, carrying a mark that is both a curse and the only thing keeping him safe.", "Dust, wheat and the clean, painful smell of a very old apology."),
  ]),
 dict(mats=["beetle-shell", "wolf-fang"], minions=["arena-gladiator", "pit-hound", "trial-automaton"], default_elite="a-reigning-champion",
  rooms=[
   ("Gate of Trials", "A portcullis of black iron. Over it, in a hundred languages, the same sentence: PROVE IT.", "Hot sand, old blood and cold iron."),
   ("The Sand Pits", "Open pits ringed with seats, every seat a different era's idea of a crowd. The sand is raked flat. It will not stay that way.", "Sun-baked sand, sweat and oiled leather."),
   ("The Bone Gallery", "A gallery of every weapon that ever lost here, mounted and labelled with the precise date of its defeat.", "Rust, bone-meal and polish."),
   ("The Trial Antechamber", "A bare room with one lamp and one question written on the floor in chalk, which you will answer with your hands.", "Chalk, lamp oil and held breath."),
   ("The Champions' Ring", "A last ring of packed earth, the fighters' benches empty, the standing champions' names scratched out in sequence by their successors.", "Dust, steel and the cool clarity of an ending."),
   ("The Warden's Circle", "A circle of white sand under a single shaft of light. The Warden stands in the middle, not moving, waiting to be proved wrong.", "Clean sand, steel and perfect, patient stillness."),
  ]),
]

# level bands per dungeon
BANDS = [(8, 18), (17, 27), (26, 36), (35, 45), (44, 54), (53, 63), (62, 72), (71, 81)]
ROOM_T = [0.0, 0.1, 0.3, 0.55, 0.8, 1.0]

def hp(L): return round(14 * L ** 1.3)
def dmg(L): return round(2.0 * L + 4)

# room layout
EXITS = {0: {"north": 1}, 1: {"south": 0, "north": 2}, 2: {"south": 1, "east": 3, "north": 4},
         3: {"west": 2}, 4: {"south": 2, "north": 5}, 5: {"south": 4}}

def zid(i, r): return 208 + i if r == 0 else 300 + i * 10 + r

def main():
    roster = json.load(open(P("data/dungeon_roster.json")))
    items = {i["name"]: i for i in json.load(open(P("data/items.json")))}
    for pool in POOLS:
        for tier in pool.values():
            for n in tier:
                assert n in items and items[n]["category"] == "weapon", ("not a weapon in items.json", n)

    rooms_go, content = [], []
    drops, spawn_rows = [], []
    for i, d in enumerate(DUNGEONS):
        lo, hi = BANDS[i]
        r0 = roster[i]
        assert len(d["rooms"]) == 6
        pool = POOLS[i]
        mats = d["mats"]
        kinds = d["minions"]
        # drop tables ---------------------------------------------------------------
        for k in kinds:
            its = [{"id": mats[0], "name": mats[0].replace("-", " ").title(), "drop_chance": .55},
                   {"id": mats[1], "name": mats[1].replace("-", " ").title(), "drop_chance": .30}]
            for w in pool["low"]:
                its.append({"id": slug(w), "name": w, "drop_chance": .012})
            drops.append({"kind": k, "items": its})
        heroes = [r0["boss"]] + r0["elite"]
        for h in heroes:
            is_boss = h == r0["boss"]
            nm = HEROES.get(h, h.replace("ARENA_HERO_", "").replace("_", " ").title())
            if is_boss:
                its = [{"id": slug(pool["top"][0]), "name": pool["top"][0], "drop_chance": .30},
                       {"id": slug(pool["top"][1]), "name": pool["top"][1], "drop_chance": .22},
                       {"id": slug(pool["top"][2]), "name": pool["top"][2], "drop_chance": .15},
                       {"id": slug(pool["mid"][0]), "name": pool["mid"][0], "drop_chance": .45},
                       {"id": mats[0], "name": mats[0].replace("-", " ").title(), "drop_chance": 1.0}]
            else:
                its = [{"id": slug(pool["mid"][0]), "name": pool["mid"][0], "drop_chance": .10},
                       {"id": slug(pool["mid"][1]), "name": pool["mid"][1], "drop_chance": .08},
                       {"id": slug(pool["mid"][2]), "name": pool["mid"][2], "drop_chance": .08},
                       {"id": mats[1], "name": mats[1].replace("-", " ").title(), "drop_chance": .8}]
            drops.append({"kind": nm, "items": its})
        # the dungeon's own fallback elite (used when the roster lists none)
        drops.append({"kind": d["default_elite"].replace("-", " ").title(), "items": [
            {"id": slug(pool["mid"][0]), "name": pool["mid"][0], "drop_chance": .12},
            {"id": slug(pool["mid"][1]), "name": pool["mid"][1], "drop_chance": .10},
            {"id": slug(pool["mid"][2]), "name": pool["mid"][2], "drop_chance": .10},
            {"id": mats[1], "name": mats[1].replace("-", " ").title(), "drop_chance": .8}]})
        # rooms ----------------------------------------------------------------------
        for r, (rname, desc, smell) in enumerate(d["rooms"]):
            z = zid(i, r)
            L = round(lo + (hi - lo) * ROOM_T[r])
            mobs = []
            if r == 1:
                mobs = [("minion", kinds[0], 4), ("minion", kinds[1], 2)]
            elif r == 2:
                mobs = [("minion", kinds[0], 3), ("minion", kinds[1], 3), ("minion", kinds[2], 2)]
            elif r == 3:
                mobs = [("elite", 0, 1), ("minion", kinds[2], 3)]
            elif r == 4:
                mobs = [("elite", 1, 1), ("minion", kinds[1], 3), ("minion", kinds[2], 3)]
            elif r == 5:
                mobs = [("boss", 0, 1), ("minion", kinds[2], 2)]
            mdefs = []
            for role, k, count in mobs:
                mult_hp = {"minion": 1.0, "elite": 2.4, "boss": 6.0}[role]
                mult_dm = {"minion": 1.0, "elite": 1.4, "boss": 1.7}[role]
                mdefs.append(dict(role=role, kind=k if role == "minion" else k, hp=round(hp(L) * mult_hp), dmg=round(dmg(L) * mult_dm),
                                  count=count, aggro=12.0 if role == "minion" else 16.0, speed=3.4 if role == "minion" else 3.0))
                if role == "minion":
                    spawn_rows.append({"zone_id": z, "kind": k, "enabled": True})
            rooms_go.append((z, i, r, rname, L, mdefs))
            content.append((z, i, r, rname, desc, smell, L))

    # ---- data files ------------------------------------------------------------------
    cur = json.load(open(P("data/mob_drops.json")))
    owned = {t["kind"].lower() for t in drops}
    cur = [t for t in cur if t["kind"].lower() not in owned] + drops
    json.dump(cur, open(P("data/mob_drops.json"), "w"), indent=2, ensure_ascii=False); open(P("data/mob_drops.json"), "a").write("\n")

    sp = json.load(open(P("data/mob_spawns.json")))
    zones_owned = {z for z, *_ in content}
    sp = [s for s in sp if s["zone_id"] not in zones_owned]
    seen = set()
    for s in spawn_rows:
        key = (s["zone_id"], s["kind"])
        if key not in seen:
            seen.add(key); sp.append(s)
    json.dump(sp, open(P("data/mob_spawns.json"), "w"), indent=2, ensure_ascii=False); open(P("data/mob_spawns.json"), "a").write("\n")

    # ---- Go: server/mob/dungeon_rooms_gen.go -------------------------------------------
    g = ["// Code generated by scripts/gen_dungeons.py; DO NOT EDIT.\n", "package mob\n",
         "// DungeonHeroNames maps roster hero ids to the TYLER compendium character names players see.",
         "var DungeonHeroNames = map[string]string{"]
    for k, v in HEROES.items(): g.append(f"\t{gostr(k)}: {gostr(v)},")
    g.append("}\n")
    g.append("// DungeonLevelBands is the recommended (low, high) level of each roster dungeon.")
    g.append("var DungeonLevelBands = [][2]int{" + ", ".join(f"{{{a}, {b}}}" for a, b in BANDS) + "}\n")
    g.append("// DungeonRoomTable lists every room zone's mobs. Role is minion/elite/boss; for elites Kind is an\n// index into the dungeon roster's elite list (wrapping), for the boss it is the roster boss.")
    g.append("var DungeonRoomTable = []DungeonRoom{")
    for z, i, r, rname, L, mdefs in rooms_go:
        ms = ", ".join(
            f"{{Role: {gostr(m['role'])}, Kind: {gostr(m['kind'] if m['role']=='minion' else '')}, EliteIdx: {m['kind'] if m['role']=='elite' else 0}, HP: {m['hp']}, Damage: {m['dmg']}, Count: {m['count']}, AggroRange: {m['aggro']}, MoveSpeed: {m['speed']}}}"
            for m in mdefs)
        g.append(f"\t{{ZoneID: {z}, Dungeon: {i}, Index: {r}, Name: {gostr(rname)}, Level: {L}, Mobs: []DungeonRoomMob{{{ms}}}}},")
    g.append("}\n")
    # default elite fallback kinds per dungeon
    g.append("// DungeonFallbackElites is used when the roster lists fewer elites than a room needs.")
    g.append("var DungeonFallbackElites = []string{" + ", ".join(gostr(d["default_elite"].replace("-", " ").title()) for d in DUNGEONS) + "}")
    open(P("server/mob/dungeon_rooms_gen.go"), "w").write("\n".join(g) + "\n")

    # ---- Go: apps2/mud/dungeon_content_gen.go ------------------------------------------
    c = ["// Code generated by scripts/gen_dungeons.py; DO NOT EDIT.\n", "package main\n",
         "// dungeonRoomNames: display names of the dungeon room zones (rooms 1-5; room 0 is the roster dungeon itself).",
         "var dungeonRoomNames = map[int]string{"]
    for z, i, r, rname, desc, smell, L in content:
        if r > 0: c.append(f"\t{z}: {gostr(rname)},")
    c.append("}\n")
    c.append("// init wires room exits, descriptions and smells for the eight dungeons.")
    c.append("func init() {")
    for z, i, r, rname, desc, smell, L in content:
        ex = dict(EXITS[r])
        exs = ", ".join(f"{gostr(dname)}: {zid(i, tgt)}" for dname, tgt in ex.items())
        if r == 0:
            exs += ', "out": 4' # leave to New Handington
        c.append(f"\texits[{z}] = map[string]int{{{exs}}}")
        full = f"[DUNGEON: {rname.upper()}] {desc} (Recommended levels {BANDS[i][0]}-{BANDS[i][1]}.)"
        c.append(f"\tzoneDesc[{z}] = {gostr(full)}")
        c.append(f"\tzoneSmell[{z}] = {gostr(smell)}")
    c.append("}\n")
    open(P("apps2/mud/dungeon_content_gen.go"), "w").write("\n".join(c) + "\n")
    print("dungeons", len(DUNGEONS), "rooms", len(content), "drop tables", len(drops), "spawn rows", len(seen))

if __name__ == "__main__":
    main()
