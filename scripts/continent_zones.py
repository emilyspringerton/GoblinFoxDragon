"""The Kuroshio Reach: a second continent, reached by ship from New Handington.

Consumed by gen_owl_zones.py (same pipeline as the Eastwind Owl cluster). build() returns the new
zones, shop tiers, exotic NM items, spell pools and extra NPCs/quests for the existing owl zones.

Layout (difficulty climbs with distance from the Open Balance Quay, mirroring the home continent):

   Handington Docks (19) --ship--> Open Balance Quay (20)  Lv 40-44    (landfall, vendors, ship home)
     ring 1: Salt Flats (21) E, Pilgrim's Terraces (22) N, Tidepool Fields (23) W        Lv 42-50
     ring 2: Cormorant Marsh (24), Shrine Stair of a Thousand Gates (25),
             Sunken Chapel Coast (26) [dungeon below], Geode Mines mouth (27) [dungeon]   Lv 52-62
     ring 3: Tanuki Council Road (28), Web Cliffs (29), Foreclosure Anchorage (30)        Lv 66-80
     capstone: The Wellspring of Tides (31) [five-room vault]                              Lv 88-99
   Interior dungeons: Geode Mines (401-403), Sunken Chapel (410-412), Tide Vault (421-425).
"""

MELEE = ["WAR", "MNK", "THF", "BST", "NIN", "DNC", "RNG", "ASN", "COR", "PUP", "BLU"]
HEAVY = ["WAR", "PLD", "DRK", "SAM", "DRG", "RUN"]
CASTER = ["WHM", "BLM", "RDM", "SMN", "SCH", "GEO", "BRD"]
SWORD = ["WAR", "PLD", "DRK", "RDM", "NIN", "RUN", "BLU", "COR"]
DAGGER = ["WAR", "MNK", "RDM", "THF", "BST", "NIN", "DNC", "ASN", "COR"]
STAFF = ["WHM", "BLM", "RDM", "SMN", "SCH", "GEO", "PUP"]
BOW = ["WAR", "RNG", "NIN", "COR"]
KATANA = ["SAM", "NIN"]
GREAT = ["WAR", "DRK", "RUN"]
H2H = ["MNK", "PUP"]


def hp(L): return round(14 * L ** 1.3)
def dmg(L): return round(2.0 * L + 4)


def build(M, N, X):
    def mob(kind, L, aggro, count, ring, drops, hpmul=1.0, dmgmul=1.0, speed=4.4):
        return M(kind, round(hp(L) * hpmul), round(dmg(L) * dmgmul), speed, aggro, count, ring, drops)

    def nm(id_, name, ph, L, items, aggro=24, speed=5.0, chance=.18, wo=600, wc=2700, resp=120):
        return N(id_, name, ph, round(hp(L) * 4.5), round(dmg(L) * 1.5), aggro, speed, chance, wo, wc, resp, items)

    zones = []

    def Z(**kw):
        kw.setdefault("variants", [])
        kw.setdefault("nms", [])
        kw.setdefault("quests", [])
        kw.setdefault("branch", None)
        zones.append(kw)

    # ---------------------------------------------------------------- docks and the ship
    Z(id=19, name="Handington Docks", lv=(1, 99), spawn=(0, 0), bounds=(30, 30), exits={"west": 4}, branch=(4, "east"), mobs=[],
      desc="[EASTWIND: HANDINGTON DOCKS / Heikegani Customs / Harbour] Slate-grey piers below the town, crowded with crates stamped with a single shell-mark. The Heikegani cargo ship OPEN BALANCE rides at the end of the longest pier, her crew counting something that will not stay counted. Type 'sail' to take passage to the Kuroshio Reach -- the far shore is for adventurers of level 35 and up.",
      smell="Tar, hemp rope and cold brine, with the dry papery smell of an invoice stamped paid.")
    Z(id=32, name="Aboard the Open Balance", lv=(1, 99), spawn=(0, 0), bounds=(20, 20), exits={}, mobs=[],
      desc="[EASTWIND: AT SEA / The Open Balance / Deck] The long grey deck of the Heikegani cargo ship, rolling slowly through water the colour of old pewter. A shell-marked ledger is chained to the mast, and an accountant in a sou'wester is reconciling the horizon. There is nowhere to go but onward.",
      smell="Wet rope, hot ledger-ink and a wind that tastes of a very long way from home.")

    # ---------------------------------------------------------------- the landing
    Z(id=20, name="Open Balance Quay", lv=(40, 44), tier=5, spawn=(0, 3), bounds=(40, 40),
      exits={"east": 21, "north": 22, "west": 23}, branch=None,
      desc="[REACH: OPEN BALANCE QUAY / Heikegani Customs / Landfall] A bustling customs quay where the Heikegani stamp every arrival in triplicate. Cranes unload crates of sealed shell-marks; gulls patrol the stacks like clerks. A signpost offers the Salt Flats east, the Pilgrim's Terraces north and the Tidepool Fields west. The ship home -- type 'sail' -- waits at the berth.",
      smell="Tar, brine and wet rope; underneath, the sharp stationery smell of a customs stamp coming down on your name.",
      npc=("customs-clerk-obe", "Customs Clerk Obe", "Name, purpose of visit, quantity of debt carried. Don't tell me you have none; everyone has some. Gear is on the left, forms on the right."),
      npcs2=[("harbourmaster-tsuru", "Harbourmaster Tsuru", "I count ships in and ships out. The numbers have never once matched, and the Parliament calls that a healthy balance.")],
      mobs=[mob("quay-rat", 40, 0, 6, 15, [("Tarred Rope", .55), ("Rat Tail", .35)], speed=4.0),
            mob("dock-crab", 42, 14, 5, 25, [("Crab Claw", .55), ("Shell Fragment", .35)], hpmul=1.2),
            mob("customs-gull", 44, 18, 3, 33, [("Gull Feather", .55), ("Stale Ledger Page", .30)], speed=5.4)],
      variants=[("quay-rat", "Stowaway Rat", 1.8), ("dock-crab", "Impounded Crab", 1.5)],
      quests=[("reach-20-ropes", "Rope for the Rigging", "Customs Clerk Obe needs 6 lengths of Tarred Rope to re-moor the quay.", {"tarred-rope": 6}, {}, 900, "pilgrim-hood"),
              ("reach-20-gulls", "Gull Audit", "Harbourmaster Tsuru wants 6 Stale Ledger Pages the gulls have stolen back.", {"stale-ledger-page": 6}, {}, 1100, "pilgrim-boots", "harbourmaster-tsuru")])
    Z(id=21, name="Salt Flats", lv=(42, 48), tier=5, spawn=(0, 3), bounds=(45, 45), exits={"west": 20, "east": 24, "down": 27},
      desc="[REACH: SALT FLATS / Free Reach / Desert] A white plain of cracked brine under a pale sun, shimmering so hard the horizon doubles. Salt-crusted ribs of old wrecks lie half-sunk in the glaze. A mine shaft yawns in the ground to the south.",
      smell="Hot salt, dry kelp and a faint, metallic note of the sea retreating.",
      npc=("salt-warden-rimu", "Salt Warden Rimu", "The flats are the sea's old floor. It will want it back. Carry water; everything out here is thirsty."),
      mobs=[mob("salt-hare", 42, 0, 6, 15, [("Salt Hare Pelt", .55), ("Salt Crystal", .40)], speed=5.4),
            mob("brine-scorpion", 45, 20, 5, 26, [("Scorpion Stinger", .55), ("Salt Crystal", .30)]),
            mob("glass-lizard", 48, 22, 3, 34, [("Glass Scale", .55), ("Salt Crystal", .35)], hpmul=1.25)],
      variants=[("salt-hare", "Dune Hare", 1.8), ("brine-scorpion", "Emperor Brine Scorpion", 1.5)],
      nms=[nm("nm-dame-halite", "Dame Halite", "glass-lizard", 48, ["Halite Crown", "Salt-Glass Dagger"])],
      quests=[("reach-21-crystals", "Salt for the Brine Works", "Salt Warden Rimu needs 8 Salt Crystals.", {"salt-crystal": 8}, {}, 1300, "pilgrim-mitts"),
              ("reach-21-halite", "Dame Halite", "A crystallised noblewoman is holding court on the flats. End her court.", {}, {"Dame Halite": 1}, 3200, "swift-blade")])
    Z(id=22, name="Pilgrim's Terraces", lv=(44, 50), tier=5, spawn=(0, -3), bounds=(45, 45), exits={"south": 20, "north": 25},
      desc="[REACH: PILGRIM'S TERRACES / Yokai Guild / Hillside] Terraced paths climb a green hill, each level with its own tiny shrine and its own queue of silent figures. They have been waiting for the bell for longer than the hill has had paths.",
      smell="Incense smoke, wet cedar and the warm, patient smell of a great many people waiting.",
      npc=("pilgrim-guide-kaede", "Pilgrim Guide Kaede", "Walk slowly. Everyone who hurries ends up queuing with the rest. Take a prayer slip if you need to talk to someone."),
      mobs=[mob("pilgrim-shade", 44, 14, 6, 15, [("Pilgrim Bell", .45), ("Prayer Slip", .55)]),
            mob("prayer-wheel-mite", 47, 20, 5, 26, [("Brass Wheel Pin", .55), ("Prayer Slip", .35)], speed=5.0),
            mob("temple-monkey", 50, 22, 3, 34, [("Monkey Fur", .55), ("Temple Incense", .35)], speed=5.6)],
      variants=[("pilgrim-shade", "Devout Shade", 1.8), ("temple-monkey", "Alpha Temple Monkey", 1.5)],
      nms=[nm("nm-abbess-cinderbell", "Abbess Cinderbell", "pilgrim-shade", 50, ["Cinder Rosary", "Bell-Metal Staff"])],
      quests=[("reach-22-slips", "Slips of Paper", "Pilgrim Guide Kaede needs 8 Prayer Slips for the terrace queue.", {"prayer-slip": 8}, {}, 1500, "pilgrim-robe"),
              ("reach-22-abbess", "Silence the Bell", "Abbess Cinderbell has been ringing the same bell for six hundred years. It is time to stop it.", {}, {"Abbess Cinderbell": 1}, 3500, "fourth-remorse")])
    Z(id=23, name="Tidepool Fields", lv=(44, 50), tier=5, spawn=(0, 3), bounds=(45, 45), exits={"east": 20, "west": 26},
      desc="[REACH: TIDEPOOL FIELDS / Ichthyosapien Marches / Shore] Black rock shelves pocked with mirror-still pools, each one a little sea of its own. Things in the pools turn to look at you as you pass, politely, and turn back.",
      smell="Rotting kelp, cold stone and the clean mineral bite of a tide pool at low water.",
      npc=("tide-reader-nami", "Tide Reader Nami", "The pools remember every tide. I read them so I do not have to read the sea. Bring me kelp, and mind the hag."),
      mobs=[mob("tidepool-crab", 44, 14, 6, 15, [("Pearl Grit", .45), ("Crab Claw", .40)], hpmul=1.2),
            mob("anemone-stalker", 47, 20, 5, 26, [("Anemone Tendril", .55), ("Kelp Fiber", .45)]),
            mob("sea-hag", 50, 24, 3, 34, [("Hag Tooth", .55), ("Kelp Fiber", .35)], hpmul=1.35, dmgmul=1.15)],
      variants=[("tidepool-crab", "Giant Tidepool Crab", 1.8), ("anemone-stalker", "Venomous Stalker", 1.5)],
      nms=[nm("nm-granny-anemone", "Granny Anemone", "sea-hag", 50, ["Anemone Shawl", "Tidepool Rod"])],
      quests=[("reach-23-kelp", "Kelp for Nami", "Tide Reader Nami needs 8 lengths of Kelp Fiber.", {"kelp-fiber": 8}, {}, 1500, "pilgrim-leggings"),
              ("reach-23-granny", "Granny Anemone", "The sea hag of the pools keeps pulling people under. Hold her under for once.", {}, {"Granny Anemone": 1}, 3500, "boots-of-winter")])

    # ---------------------------------------------------------------- ring 2
    Z(id=24, name="Cormorant Marsh", lv=(52, 60), tier=6, spawn=(0, 3), bounds=(45, 45), exits={"west": 21, "north": 29},
      desc="[REACH: CORMORANT MARSH / Free Reach / Wetland] Brown water between reed islands, strung with the stilted nests of cormorants who fish on commission. A kappa magistrate presides, somewhere, over all of it.",
      smell="Fish oil, reed rot and the green, cold smell of a cucumber just cut open.",
      npc=("marsh-factor-gen", "Marsh Factor Gen", "The cormorants fish for the Parliament, the kappa fine them for it, and I count the difference. It is a good living."),
      mobs=[mob("cormorant", 52, 16, 6, 15, [("Cormorant Feather", .55), ("Fish Oil", .45)], speed=5.2),
            mob("mud-skipper", 56, 20, 5, 26, [("Skipper Scale", .55), ("Kelp Fiber", .30)], speed=5.0),
            mob("bog-kappa", 60, 24, 3, 34, [("Kappa Shell", .50), ("Cucumber Brine", .45)], hpmul=1.25)],
      variants=[("cormorant", "Master Cormorant", 1.8), ("bog-kappa", "Elder Kappa", 1.5)],
      nms=[nm("nm-magistrate-kappa", "Magistrate Kappa", "bog-kappa", 60, ["Magistrate's Dish", "Kappa-Hide Greaves"])],
      quests=[("reach-24-oil", "Oil for the Lamps", "Marsh Factor Gen needs 8 vials of Fish Oil.", {"fish-oil": 8}, {}, 2200, "tidewarden-mitts"),
              ("reach-24-kappa", "A Court of Kappa", "Magistrate Kappa fines everyone who crosses the marsh. Dismiss the court.", {}, {"Magistrate Kappa": 1}, 4200, "light's-end")])
    Z(id=25, name="Shrine Stair of a Thousand Gates", lv=(54, 62), tier=6, spawn=(0, -3), bounds=(45, 45), exits={"south": 22, "north": 28},
      desc="[REACH: SHRINE STAIR OF A THOUSAND GATES / Yokai Guild / Mountain] A staircase of red torii marches up the mountain, gate after gate after gate. Fox-fire drifts between the pillars and every stone fox on the way has turned its head to follow you.",
      smell="Red lacquer, camphor and the clean smoke of fox-fire burning without fuel.",
      npc=("gatekeeper-ume", "Gatekeeper Ume", "A thousand gates and each one asks a question. Answer politely, answer briefly, and never answer 'technically'."),
      mobs=[mob("gate-tengu", 54, 20, 6, 15, [("Tengu Feather", .55), ("Torii Lacquer", .40)], speed=5.4),
            mob("fox-fire-wisp", 58, 22, 5, 26, [("Foxfire Wick", .55), ("Torii Lacquer", .30)], speed=5.0),
            mob("komainu", 62, 26, 3, 34, [("Guardian Stone", .50), ("Torii Lacquer", .35)], hpmul=1.3)],
      variants=[("gate-tengu", "Crow Tengu Captain", 1.8), ("fox-fire-wisp", "Blue Foxfire", 1.5)],
      nms=[nm("nm-tengu-warlord-hayate", "Tengu Warlord Hayate", "gate-tengu", 62, ["Hayate's Fan", "Warlord's Feathered Cape"])],
      quests=[("reach-25-lacquer", "Fresh Paint", "Gatekeeper Ume needs 8 pots of Torii Lacquer to repaint the lower gates.", {"torii-lacquer": 8}, {}, 2400, "tidewarden-hood"),
              ("reach-25-hayate", "The Tengu Warlord", "Hayate guards the three-hundredth gate. He will not move aside by asking.", {}, {"Tengu Warlord Hayate": 1}, 4500, "shepherd's-staff")])
    Z(id=26, name="Sunken Chapel Coast", lv=(54, 62), tier=6, spawn=(0, 3), bounds=(45, 45), exits={"east": 23, "west": 30, "down": 410},
      desc="[REACH: SUNKEN CHAPEL COAST / Ichthyosapien Marches / Coast] A drowned chapel leans out of the surf, its bell tower dipping into the waves like a spoon into soup. Stairs descend into the flooded nave. Acolytes in barnacled robes sing the tide in.",
      smell="Wet plaster, cold candle-wax and brine: the smell of a sermon long since underwater.",
      npc=("sexton-hiro", "Sexton Hiro", "The Church of the Sleepers never closed. It just went down. Mind the stairs; the congregation is still in."),
      mobs=[mob("brine-acolyte", 54, 18, 6, 15, [("Acolyte Rosary", .50), ("Brine Pearl", .45)]),
            mob("drowned-choirboy", 58, 22, 5, 26, [("Choir Sheet", .55), ("Brine Pearl", .30)], speed=5.0),
            mob("shell-sexton", 62, 24, 3, 34, [("Sexton Key", .45), ("Brine Pearl", .35)], hpmul=1.3)],
      variants=[("brine-acolyte", "Senior Acolyte", 1.8), ("drowned-choirboy", "Choirmaster's Favourite", 1.5)],
      quests=[("reach-26-pearls", "Pearls for the Sexton", "Sexton Hiro needs 8 Brine Pearls to ballast the bell.", {"brine-pearl": 8}, {}, 2400, "tidewarden-boots"),
              ("reach-26-cardinal", "Cardinal Brinewright", "In the crypt below, the Church's last cardinal still presides. Dismiss him.", {}, {"Cardinal Brinewright": 1}, 5000, "choker-of-dawn")])
    Z(id=27, name="Geode Mines", lv=(52, 60), tier=6, spawn=(0, 3), bounds=(35, 35), exits={"up": 21, "down": 401}, branch=None,
      desc="[REACH: GEODE MINES / Free Reach / Mine] The mouth of an old salt-and-crystal mine, timbered with whale-ribs. The air is cool and glitters faintly. Lantern-bats hang from the lintel like unlit lamps.",
      smell="Cold stone, mineral dust and the faint ozone of crystal growing in the dark.",
      mobs=[mob("geode-beetle", 52, 16, 6, 14, [("Geode Shard", .55), ("Beetle Plate", .40)], hpmul=1.2),
            mob("quartz-crawler", 56, 20, 5, 24, [("Quartz Sliver", .55), ("Geode Shard", .30)]),
            mob("lantern-bat", 60, 24, 3, 32, [("Lantern Wing", .55), ("Geode Shard", .30)], speed=5.8)],
      variants=[("geode-beetle", "Prismatic Beetle", 1.8), ("quartz-crawler", "Cave-in Crawler", 1.5)])

    # interior: Geode Mines rooms (indoor)
    Z(id=401, name="Geode Mines: The Shaft", lv=(54, 58), spawn=(0, 3), bounds=(30, 30), exits={"up": 27, "east": 402},
      desc="[REACH: GEODE MINES -- THE SHAFT] A narrow shaft braced with whale-ribs, rope ladders descending into a dark that glitters. Miners' lanterns still hang where they were left.",
      smell="Lamp oil, cold rock and rope that has been wet and dry a hundred times.",
      mobs=[mob("geode-beetle", 54, 16, 5, 14, [("Geode Shard", .55), ("Beetle Plate", .40)], hpmul=1.2),
            mob("quartz-crawler", 56, 20, 4, 24, [("Quartz Sliver", .55), ("Geode Shard", .30)])])
    Z(id=402, name="Geode Mines: The Crystal Cavern", lv=(56, 60), spawn=(0, 3), bounds=(30, 30), exits={"west": 401, "down": 403},
      desc="[REACH: GEODE MINES -- THE CRYSTAL CAVERN] A cavern lined floor to ceiling with pale quartz, every facet holding a small reflection of you that is slightly out of step.",
      smell="Clean glass, wet limestone and a bright, cold smell like the inside of a bell.",
      mobs=[mob("quartz-crawler", 57, 20, 5, 24, [("Quartz Sliver", .55), ("Geode Shard", .30)]),
            mob("lantern-bat", 60, 24, 4, 32, [("Lantern Wing", .55), ("Geode Shard", .30)], speed=5.8)])
    Z(id=403, name="Geode Mines: The Heart Geode", lv=(58, 62), spawn=(0, 3), bounds=(30, 30), exits={"up": 402},
      desc="[REACH: GEODE MINES -- THE HEART GEODE] A hollow the size of a cathedral, its walls a single enormous geode. At its centre a boulder of living quartz is slowly, patiently, turning its attention toward you.",
      smell="Hot crystal, ozone and the heavy sweetness of something very old waking.",
      mobs=[mob("quartz-crawler", 60, 22, 3, 20, [("Quartz Sliver", .55), ("Geode Shard", .30)]),
            mob("geode-beetle", 62, 22, 3, 28, [("Geode Shard", .55), ("Beetle Plate", .40)], hpmul=1.3)],
      nms=[nm("nm-old-quartz-ma", "Old Quartz Ma", "quartz-crawler", 62, ["Quartz Heart Pendant", "Geode-Edge Katana"], aggro=20, resp=150)],
      quests=[])

    # interior: Sunken Chapel rooms
    Z(id=410, name="Sunken Chapel: The Nave", lv=(56, 60), spawn=(0, 3), bounds=(30, 30), exits={"up": 26, "north": 411},
      desc="[REACH: SUNKEN CHAPEL -- THE NAVE] Knee-deep water fills a nave of grey pews. Where the choir loft should be, a school of silver fish sings in perfect fourths.",
      smell="Standing water, old wood and candle-wax gone soft as butter.",
      mobs=[mob("brine-acolyte", 56, 18, 5, 14, [("Acolyte Rosary", .50), ("Brine Pearl", .45)]),
            mob("drowned-choirboy", 59, 22, 4, 24, [("Choir Sheet", .55), ("Brine Pearl", .30)], speed=5.0)])
    Z(id=411, name="Sunken Chapel: The Baptistry of Brine", lv=(58, 62), spawn=(0, 3), bounds=(30, 30), exits={"south": 410, "north": 412},
      desc="[REACH: SUNKEN CHAPEL -- THE BAPTISTRY] A round font full of green brine, wide as a pond. Every candle in the room is lit and none of them are wet.",
      smell="Brine, wax and the green coolness of a baptism that never quite ended.",
      mobs=[mob("drowned-choirboy", 60, 22, 4, 14, [("Choir Sheet", .55), ("Brine Pearl", .30)], speed=5.0),
            mob("shell-sexton", 62, 24, 4, 24, [("Sexton Key", .45), ("Brine Pearl", .35)], hpmul=1.3)])
    Z(id=412, name="Sunken Chapel: The Crypt of the Sleepers", lv=(60, 64), spawn=(0, 3), bounds=(30, 30), exits={"south": 411},
      desc="[REACH: SUNKEN CHAPEL -- THE CRYPT OF THE SLEEPERS] Niches of shell-marked stone hold sleeping figures, breathing very slowly. At the end of the vault a cardinal in barnacled red presides over the dream.",
      smell="Cold stone, salt and the heavy sweetness of sleep.",
      mobs=[mob("shell-sexton", 62, 24, 3, 20, [("Sexton Key", .45), ("Brine Pearl", .35)], hpmul=1.3),
            mob("drowned-choirboy", 62, 22, 3, 28, [("Choir Sheet", .55), ("Brine Pearl", .30)], speed=5.0)],
      nms=[nm("nm-cardinal-brinewright", "Cardinal Brinewright", "shell-sexton", 64, ["Cardinal's Brine Ring", "Sleeper's Crozier"], aggro=22, resp=150)])

    # ---------------------------------------------------------------- ring 3
    Z(id=28, name="Tanuki Council Road", lv=(66, 76), tier=7, spawn=(0, -3), bounds=(45, 45), exits={"south": 25, "north": 31},
      desc="[REACH: TANUKI COUNCIL ROAD / Yokai Guild / Highway] A broad road of pale stone, lined with teahouses that are, on inspection, entirely made of leaves. Bailiffs in round hats escort travellers to the council at the top, courteously and without any possibility of refusal.",
      smell="Green tea, hot rice and a leaf-mould smell that is almost, but not quite, the smell of money.",
      npc=("tea-mistress-fumi", "Tea Mistress Fumi", "Hospitality binds both parties, traveller. You will find the host is bound to be gracious -- and so, I'm afraid, are you."),
      npcs2=[("road-bailiff-kuma", "Bailiff Kuma", "The council is in session. The council is always in session. Please keep your hands where the leaves can see them.")],
      mobs=[mob("tanuki-bailiff", 66, 20, 6, 15, [("Leaf Seal", .50), ("Hospitality Token", .45)]),
            mob("leaf-bandit", 71, 24, 5, 26, [("Stolen Leaf", .55), ("Hospitality Token", .30)], speed=5.4),
            mob("hospitality-golem", 76, 28, 3, 34, [("Teapot Shard", .50), ("Hospitality Token", .35)], hpmul=1.35)],
      variants=[("tanuki-bailiff", "High Bailiff", 1.8), ("leaf-bandit", "Master of Disguise", 1.5)],
      nms=[nm("nm-bailiff-bunbuku", "Bailiff Bunbuku", "tanuki-bailiff", 76, ["Bunbuku's Teakettle Helm", "Council Gavel"], resp=180)],
      quests=[("reach-28-tokens", "Tokens of Hospitality", "Tea Mistress Fumi needs 8 Hospitality Tokens.", {"hospitality-token": 8}, {}, 3600, "reachrunner-mitts"),
              ("reach-28-bunbuku", "The Teakettle Bailiff", "Bunbuku is a kettle, a tanuki and a magistrate. Defeat all three at once.", {}, {"Bailiff Bunbuku": 1}, 6500, "icewraith-claymore", "road-bailiff-kuma")])
    Z(id=29, name="Web Cliffs", lv=(68, 78), tier=7, spawn=(0, 3), bounds=(45, 45), exits={"south": 24, "east": 31},
      desc="[REACH: WEB CLIFFS / Kuroshio Arachnids / Cliff] Sea cliffs hung with silk that catches the wind like sails. Every thread hums, very faintly, at a pitch that only becomes obvious when it stops.",
      smell="Sea spray, cold stone and the dry, clean smell of silk drying in the wind.",
      npc=("silk-factor-orochi", "Silk Factor Orochi", "Everything on these cliffs is a thread, and every thread is a message. I deal in the ones nobody is waiting on."),
      mobs=[mob("cliff-weaver", 68, 22, 6, 15, [("Weaver Silk", .55), ("Web Spinneret", .40)]),
            mob("thread-sentinel", 73, 26, 5, 26, [("Thread Spool", .55), ("Web Spinneret", .30)]),
            mob("silk-wasp", 78, 28, 3, 34, [("Wasp Wing", .55), ("Weaver Silk", .35)], speed=6.0)],
      variants=[("cliff-weaver", "Cliff Matriarch", 1.8), ("silk-wasp", "Razor Wasp", 1.5)],
      nms=[nm("nm-silkmother-ixchel", "Silkmother Ixchel", "cliff-weaver", 78, ["Ixchel's Spinneret Charm", "Silkmother's Longbow"], resp=180)],
      quests=[("reach-29-silk", "Silk for the Factor", "Silk Factor Orochi needs 8 skeins of Weaver Silk.", {"weaver-silk": 8}, {}, 3800, "reachrunner-hood"),
              ("reach-29-ixchel", "Silkmother Ixchel", "The matriarch of the cliffs has outlived four of her own webs. End the fifth.", {}, {"Silkmother Ixchel": 1}, 7000, "bow-of-the-gorgon-outcast")])
    Z(id=30, name="Foreclosure Anchorage", lv=(70, 80), tier=7, spawn=(0, 3), bounds=(45, 45), exits={"east": 26, "west": 31},
      desc="[REACH: FORECLOSURE ANCHORAGE / Shell Parliament / Harbour] A grey harbour crowded with ships that have not sailed in years, each one sealed with a red writ of seizure. Their anchors are chained to the seabed with ledger-links. The tide here is always going out.",
      smell="Rust, cold iron and the dry, official smell of ink on a writ.",
      npc=("receiver-hatsu", "Receiver Hatsu", "These ships are in receivership. So, technically, are you. Please do not use the word 'technically' near the Admiral."),
      mobs=[mob("foreclosure-bailiff", 70, 22, 6, 15, [("Foreclosure Writ", .55), ("Shell-Mark Coin", .40)]),
            mob("anchor-wight", 75, 26, 5, 26, [("Anchor Link", .55), ("Shell-Mark Coin", .30)], hpmul=1.3),
            mob("ledger-eel", 80, 28, 3, 34, [("Eel Skin", .55), ("Shell-Mark Coin", .35)], speed=5.8)],
      variants=[("foreclosure-bailiff", "Senior Bailiff", 1.8), ("anchor-wight", "Drowned Anchor-Wight", 1.5)],
      nms=[nm("nm-admiral-vesper-foreclose", "Admiral Vesper Foreclose", "anchor-wight", 80, ["Admiral's Writ Cutlass", "Foreclosure Sash"], resp=180)],
      quests=[("reach-30-writs", "Unserved Writs", "Receiver Hatsu needs 8 Foreclosure Writs returned to the office.", {"foreclosure-writ": 8}, {}, 4000, "reachrunner-boots"),
              ("reach-30-admiral", "Admiral Vesper Foreclose", "The Admiral has seized every ship in the harbour. Seize him back.", {}, {"Admiral Vesper Foreclose": 1}, 7500, "last-regret")])

    # ---------------------------------------------------------------- capstone vault
    Z(id=31, name="The Wellspring of Tides", lv=(88, 99), tier=9, spawn=(0, 3), bounds=(40, 40), exits={"south": 28, "west": 29, "east": 30, "down": 421},
      desc="[REACH: THE WELLSPRING OF TIDES / Chrono-Cube / Spring] A round pool of still, black water at the top of the world. Tides rise and fall in it that belong to no sea. On its shore stand the three roads of the Reach, and below the water a stair descends.",
      smell="Cold, clean water, a faint hum at 7.83 Hz, and a sharp sweet note like a coin spinning, just before it lands.",
      npc=("wellspring-keeper-sora", "Wellspring Keeper Sora", "The pool keeps every tide that ever turned. Whatever you carried out of the Reach, you carry back. Mind the Cube below."),
      mobs=[mob("tide-echo", 88, 24, 5, 14, [("Tidewater Vial", .50), ("Paradox Dust", .45)]),
            mob("paradox-shade", 92, 28, 4, 24, [("Paradox Dust", .55), ("Cube Splinter", .35)], hpmul=1.2),
            mob("cube-warden", 96, 30, 2, 32, [("Cube Splinter", .55), ("Chrono Filament", .30)], hpmul=1.4)],
      variants=[("tide-echo", "Resonant Echo", 1.8), ("paradox-shade", "Twice-Told Shade", 1.5)],
      quests=[("reach-31-dust", "Dust of Paradox", "Wellspring Keeper Sora needs 10 pinches of Paradox Dust.", {"paradox-dust": 10}, {}, 9000, "operator's-tunic"),
              ("reach-31-warden", "The Cube Warden", "A construct built to guard the Cube below keeps stepping out of the vault. End its patrol.", {}, {"Warden Vesper of the Cube": 1}, 14000, "helm-of-the-forgotten-kingdom")])
    Z(id=421, name="The Tide Vault: Tide Hall", lv=(88, 92), spawn=(0, 3), bounds=(30, 30), exits={"up": 31, "north": 422},
      desc="[REACH: THE TIDE VAULT -- TIDE HALL] A hall of black glass where water flows upward along the walls. Every footfall arrives a moment before you take it.",
      smell="Cold water, polished glass and the faint electric taste of a clock about to strike.",
      mobs=[mob("tide-echo", 89, 24, 5, 14, [("Tidewater Vial", .50), ("Paradox Dust", .45)]),
            mob("paradox-shade", 91, 28, 4, 24, [("Paradox Dust", .55), ("Cube Splinter", .35)], hpmul=1.2)])
    Z(id=422, name="The Tide Vault: Paradox Gallery", lv=(90, 94), spawn=(0, 3), bounds=(30, 30), exits={"south": 421, "east": 423, "north": 424},
      desc="[REACH: THE TIDE VAULT -- PARADOX GALLERY] A gallery of paintings of the same room from fifty angles, each showing you entering from a different door.",
      smell="Linseed oil, cold glass and the sweetish smell of a decision made twice.",
      mobs=[mob("paradox-shade", 92, 28, 4, 14, [("Paradox Dust", .55), ("Cube Splinter", .35)], hpmul=1.2),
            mob("cube-warden", 94, 30, 3, 24, [("Cube Splinter", .55), ("Chrono Filament", .30)], hpmul=1.4)])
    Z(id=423, name="The Tide Vault: Cube Reliquary", lv=(92, 96), spawn=(0, 3), bounds=(30, 30), exits={"west": 422},
      desc="[REACH: THE TIDE VAULT -- CUBE RELIQUARY] A shelf of empty plinths, each labelled with the date a Cube once sat there. One date is both 1623 and 1901.",
      smell="Brass polish, dust and the dry sweetness of an object that has just left.",
      mobs=[mob("paradox-shade", 93, 28, 3, 14, [("Paradox Dust", .55), ("Cube Splinter", .35)], hpmul=1.2),
            mob("cube-warden", 95, 30, 4, 22, [("Cube Splinter", .55), ("Chrono Filament", .30)], hpmul=1.4)],
      nms=[nm("nm-warden-vesper-of-the-cube", "Warden Vesper of the Cube", "cube-warden", 96, ["Cube Warden's Aegis", "Vesper's Chrono Blade"], aggro=24, resp=240)])
    Z(id=424, name="The Tide Vault: Antechamber of Coins", lv=(94, 98), spawn=(0, 3), bounds=(30, 30), exits={"south": 422, "north": 425},
      desc="[REACH: THE TIDE VAULT -- ANTECHAMBER OF COINS] Ten thousand coins stand on edge in the air, spinning in silence. None have landed. All of them are about to.",
      smell="Copper, clean water and the held breath of ten thousand almost-decisions.",
      mobs=[mob("cube-warden", 96, 30, 4, 18, [("Cube Splinter", .55), ("Chrono Filament", .30)], hpmul=1.4),
            mob("paradox-shade", 97, 28, 3, 28, [("Paradox Dust", .55), ("Cube Splinter", .35)], hpmul=1.2)])
    Z(id=425, name="The Tide Vault: The Still Point", lv=(96, 99), spawn=(0, 3), bounds=(30, 30), exits={"south": 424},
      desc="[REACH: THE TIDE VAULT -- THE STILL POINT] A room in which nothing is moving, including you. At its centre a single coin hangs in the air, heads and tails at once, waiting for someone to call it.",
      smell="Nothing: still air, still water. The kind of silence that is also a smell.",
      mobs=[mob("paradox-shade", 97, 28, 3, 12, [("Paradox Dust", .55), ("Cube Splinter", .35)], hpmul=1.2),
            mob("cube-warden", 98, 32, 2, 20, [("Cube Splinter", .55), ("Chrono Filament", .30)], hpmul=1.4)],
      nms=[nm("nm-the-coin-that-landed-both-ways", "The Coin That Landed Both Ways", "cube-warden", 99, ["Coin-Flip Blade", "Both-Sides Mantle"], aggro=22, resp=360, chance=.12)])

    nm_items = [
        # 21 Dame Halite
        X("Halite Crown", "armor", ["head"], 50, {"defense": 34, "int": 6, "mnd": 6, "magic_defense_bonus": 8}, "head_crown_madness_01", CASTER),
        X("Salt-Glass Dagger", "weapon", ["main", "off"], 49, {"attack": 84, "dex": 6, "accuracy": 10}, "dagger_kali_01", DAGGER, "dagger", 154),
        # 22 Abbess Cinderbell
        X("Cinder Rosary", "accessory", ["neck"], 52, {"mnd": 7, "magic_defense_bonus": 10, "hp_regen": 1}, "neck_priestess_charm"),
        X("Bell-Metal Staff", "weapon", ["main"], 51, {"magic_attack_bonus": 40, "mnd": 7, "int": 7}, "staff_healing", STAFF, "staff", 188),
        # 23 Granny Anemone
        X("Anemone Shawl", "accessory", ["back"], 52, {"agi": 6, "evasion": 14, "defense": 10}, "cape_yellow_01"),
        X("Tidepool Rod", "weapon", ["main"], 51, {"magic_attack_bonus": 42, "int": 8, "mnd": 6, "mp": 30}, "staff_shepherd_01", STAFF, "staff", 190),
        # 24 Magistrate Kappa
        X("Magistrate's Dish", "armor", ["head"], 61, {"defense": 38, "vit": 6, "hp": 40, "magic_defense_bonus": 8}, "helm_turtle_01", HEAVY),
        X("Kappa-Hide Greaves", "armor", ["legs"], 60, {"defense": 40, "agi": 5, "evasion": 12}, "legs_wolfskin_01", MELEE),
        # 25 Tengu Warlord Hayate
        X("Hayate's Fan", "weapon", ["main"], 62, {"attack": 108, "agi": 8, "haste": 4, "accuracy": 12}, "katana_juzumaru", KATANA, "katana", 200),
        X("Warlord's Feathered Cape", "accessory", ["back"], 62, {"agi": 8, "evasion": 18, "attack": 8, "defense": 12}, "threads_of_missing_gods_mantle"),
        # 26 Cardinal Brinewright (chapel boss)
        X("Cardinal's Brine Ring", "accessory", ["ringl", "ringr"], 63, {"mnd": 8, "magic_defense_bonus": 12, "mp": 40}, "ring_magic_a01"),
        X("Sleeper's Crozier", "weapon", ["main"], 63, {"magic_attack_bonus": 52, "mnd": 10, "int": 6, "fast_cast": 4}, "staff_brass", STAFF, "staff", 186),
        # 27 Old Quartz Ma (geode boss)
        X("Quartz Heart Pendant", "accessory", ["neck"], 61, {"vit": 7, "defense": 14, "hp": 60}, "neck_choker_dawn"),
        X("Geode-Edge Katana", "weapon", ["main"], 61, {"attack": 112, "str": 9, "crit_rate": 3}, "katana_wooden_01", KATANA, "katana", 206),
        # 28 Bailiff Bunbuku
        X("Bunbuku's Teakettle Helm", "armor", ["head"], 76, {"defense": 52, "vit": 8, "enmity": 6, "hp": 70}, "helm_dragon_king_01", HEAVY, ex=True),
        X("Council Gavel", "weapon", ["main"], 75, {"attack": 140, "str": 12, "accuracy": 14}, "weapon_sword_keen_edge", SWORD, "sword", 216, ex=True),
        # 29 Silkmother Ixchel
        X("Ixchel's Spinneret Charm", "accessory", ["neck"], 77, {"dex": 8, "agi": 8, "evasion": 16, "ranged_accuracy": 14}, "neck_priestess_charm", ex=True),
        X("Silkmother's Longbow", "weapon", ["main"], 77, {"attack": 130, "ranged_attack": 120, "agi": 12, "ranged_accuracy": 16}, "bow_gorgon_01", BOW, "bow", 296, ex=True),
        # 30 Admiral Vesper Foreclose
        X("Admiral's Writ Cutlass", "weapon", ["main", "off"], 79, {"attack": 148, "dex": 10, "str": 8, "enmity": 4}, "sabre_01", SWORD, "sword", 218, ex=True),
        X("Foreclosure Sash", "accessory", ["waist"], 78, {"vit": 8, "defense": 18, "enmity": 8, "hp": 50}, "waist_belt_life", ex=True),
        # 31 vault: Warden Vesper of the Cube
        X("Cube Warden's Aegis", "armor", ["body"], 95, {"defense": 120, "vit": 14, "str": 10, "magic_defense_bonus": 14, "hp": 120}, "body_plate_knight", HEAVY, ex=True),
        X("Vesper's Chrono Blade", "weapon", ["main"], 96, {"attack": 190, "str": 14, "dex": 10, "haste": 6, "accuracy": 18}, "ws_remorse_dark", SWORD, "sword", 206, ex=True),
        # The Coin That Landed Both Ways
        X("Coin-Flip Blade", "weapon", ["main", "off"], 99, {"attack": 210, "dex": 14, "crit_rate": 6, "accuracy": 20, "haste": 5}, "w_sword_swift_blade", SWORD, "sword", 198, ex=True),
        X("Both-Sides Mantle", "accessory", ["back"], 99, {"str": 8, "dex": 8, "vit": 8, "agi": 8, "int": 8, "mnd": 8, "evasion": 20, "defense": 20}, "threads_of_missing_gods_mantle", ex=True),
    ]

    tiers = [("Pilgrim", 50), ("Tidewarden", 60), ("Reachrunner", 70), ("Foxfire", 80), ("Wellspring", 90)]

    spells = {
        "quay-rat": ["Slow"], "dock-crab": ["Bind"], "customs-gull": ["Silence"],
        "salt-hare": ["Slow"], "brine-scorpion": ["Poison"], "glass-lizard": ["Paralyze", "Slow"],
        "pilgrim-shade": ["Silence", "Bind"], "prayer-wheel-mite": ["Slow", "Paralyze"], "temple-monkey": ["Slow"],
        "tidepool-crab": ["Bind"], "anemone-stalker": ["Poison", "Paralyze"], "sea-hag": ["Silence", "Poison"],
        "cormorant": ["Slow"], "mud-skipper": ["Poison"], "bog-kappa": ["Bind", "Silence"],
        "gate-tengu": ["Silence", "Slow"], "fox-fire-wisp": ["Silence", "Paralyze"], "komainu": ["Bind", "Paralyze"],
        "brine-acolyte": ["Silence", "Poison"], "drowned-choirboy": ["Silence"], "shell-sexton": ["Bind", "Paralyze"],
        "geode-beetle": ["Slow"], "quartz-crawler": ["Bind", "Poison"], "lantern-bat": ["Silence", "Slow"],
        "tanuki-bailiff": ["Bind", "Silence"], "leaf-bandit": ["Slow", "Paralyze"], "hospitality-golem": ["Bind", "Paralyze"],
        "cliff-weaver": ["Bind", "Poison"], "thread-sentinel": ["Paralyze", "Bind"], "silk-wasp": ["Poison", "Slow"],
        "foreclosure-bailiff": ["Silence", "Bind"], "anchor-wight": ["Bind", "Paralyze"], "ledger-eel": ["Paralyze", "Poison"],
        "tide-echo": ["Silence", "Slow"], "paradox-shade": ["Silence", "Bind", "Paralyze"], "cube-warden": ["Bind", "Paralyze", "Silence"],
    }

    # extra NPCs and quests for the existing owl zones (rewards are real items from items.json)
    owl_extras = {
        10: dict(npcs2=[("fledgling-mara", "Fledgling Mara", "I was a fledgling once. It is mostly standing still and being very, very quiet.")],
                 quests=[("owl-10-whiskers", "Whisker Count", "Fledgling Mara needs 6 Mouse Whiskers for the class tally.", {"mouse-whisker": 6}, {}, 150, "iron-shortsword", "fledgling-mara"),
                         ("owl-10-moths", "Dust the Moths", "Mara wants the dust of 5 moths for a lesson on unfiled things.", {"moth-dust": 5}, {}, 200, "hand-axe", "fledgling-mara")]),
        11: dict(npcs2=[("surveyor-pim", "Surveyor Pim", "Every cairn on this perch is a number. I'm missing three. They are not where I left them.")],
                 quests=[("owl-11-quills", "Sentry Quills", "Surveyor Pim needs 6 Sentry Quills to redraw the survey.", {"sentry-quill": 6}, {}, 380, "sabre", "surveyor-pim"),
                         ("owl-11-pinions", "Kestrel Pinions", "Pim wants 5 Kestrel Pinions for flags.", {"kestrel-pinion": 5}, {}, 420, "beestinger", "surveyor-pim")]),
        13: dict(npcs2=[("ferryman-bog", "Ferryman Bog", "Poled these pellets for forty years. Bones and reeds, mostly. Bring me bone and I'll row you anywhere you can't go.")],
                 quests=[("owl-13-voles", "Vole Pelts", "Ferryman Bog needs 6 Vole Pelts to line the punt.", {"vole-pelt": 6}, {}, 520, "swift-blade", "ferryman-bog"),
                         ("owl-13-ichor", "Slug Ichor", "Bog wants 5 vials of Slug Ichor to caulk the boat.", {"slug-ichor": 5}, {}, 560, "composite-bow", "ferryman-bog")]),
        12: dict(npcs2=[("lantern-lad-fitz", "Lantern Lad Fitz", "Only job in the Deeps is keeping the lanterns lit. Only problem is the bats keep drinking the oil.")],
                 quests=[("owl-12-guano", "Guano Clumps", "Fitz needs 6 Guano Clumps for the gardeners upstairs.", {"guano-clump": 6}, {}, 700, "paltry-bastard-sword", "lantern-lad-fitz"),
                         ("owl-12-shade", "Shade Feathers", "Fitz wants 5 Shade Feathers for a very quiet lantern shade.", {"shade-feather": 5}, {}, 760, "hidden-dagger-of-kali", "lantern-lad-fitz")]),
        14: dict(npcs2=[("lens-grinder-ro", "Lens Grinder Ro", "I grind glass until it sees something. Then I grind it until it stops.")],
                 quests=[("owl-14-wisps", "Wisp Glass", "Lens Grinder Ro needs 6 Wisp Glass fragments.", {"wisp-glass": 6}, {}, 1100, "steel-sword", "lens-grinder-ro"),
                         ("owl-14-plumes", "Plumes for the Tor", "Ro wants 5 Eagle Owl Plumes for dusting the lenses.", {"eagle-owl-plume": 5}, {}, 1200, "banjo", "lens-grinder-ro")]),
        15: dict(npcs2=[("gale-rider-ytha", "Gale Rider Ytha", "I ride the one wind that never changes. It is not interesting. It is, however, reliable.")],
                 quests=[("owl-15-hare", "Hare Pelts", "Ytha needs 6 Hare Pelts to line her cloak.", {"hare-pelt": 6}, {}, 1000, "death's-gambit", "gale-rider-ytha"),
                         ("owl-15-sleet", "Sleet Crystals", "Ytha wants 5 Sleet Crystals for the ridge beacons.", {"sleet-crystal": 5}, {}, 1100, "healing-staff", "gale-rider-ytha")]),
        16: dict(npcs2=[("bindery-apprentice-oz", "Bindery Apprentice Oz", "My master rebinds the books that rebind themselves. I fetch the glue.")],
                 quests=[("owl-16-ink", "Ink Vials", "Oz needs 6 Ink Vials for the bindery.", {"ink-vial": 6}, {}, 1500, "keen-edge", "bindery-apprentice-oz"),
                         ("owl-16-paste", "Bookbinding Paste", "Oz wants 5 pots of Bookbinding Paste.", {"bookbinding-paste": 5}, {}, 1600, "qi-tornado", "bindery-apprentice-oz")]),
        17: dict(npcs2=[("stone-polisher-agnes", "Stone Polisher Agnes", "I polish the footnote stones so the superscripts stay sharp. Nobody thanks me. Nobody should.")],
                 quests=[("owl-17-feathers", "Hag Feathers", "Agnes needs 6 Hag Feathers for dusting.", {"hag-feather": 6}, {}, 1400, "sunken-dagger-of-kali", "stone-polisher-agnes"),
                         ("owl-17-oil", "Lamp Oil", "Agnes wants 6 flasks of Lamp Oil.", {"lamp-oil": 6}, {}, 1450, "titanium-claws", "stone-polisher-agnes")]),
        18: dict(npcs2=[("wall-mason-dren", "Wall Mason Dren", "Every stone in the Wall is a clause. When one cracks, the sentence changes. I fix sentences.")],
                 quests=[("owl-18-shards", "Black Bar Shards", "Dren needs 6 Black Bar Shards to patch redactions.", {"black-bar-shard": 6}, {}, 2400, "mythril-sword", "wall-mason-dren"),
                         ("owl-18-feathers", "Redacted Feathers", "Dren wants 5 Redacted Feathers.", {"redacted-feather": 5}, {}, 2600, "fourth-remorse", "wall-mason-dren")]),
    }

    return {"zones": zones, "nm_items": nm_items, "tiers": tiers, "spells": spells, "owl_extras": owl_extras}
