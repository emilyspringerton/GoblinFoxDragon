# The Bridge: Shankpit ↔ Bedrock

This document captures the bridge layer for the Shankpit ↔ Bedrock hybrid protocol.

## Goals
- Keep the Dragonfly fork (DragonsnShit) as the authoritative game server.
- Support FPS-native mechanics (hitscan weapons, Shankpit input flow).
- Stream simplified voxel data to the Shankpit client for rapid rendering.

## Protocol Extension
Add a simplified voxel payload that the Shankpit client can parse without full Bedrock chunk decoding.

```c
#define PACKET_VOXEL_DATA 4

typedef struct {
    unsigned char x;
    unsigned char y;
    unsigned char z;
    unsigned short block_id;
} VoxelBlock;

typedef struct {
    unsigned char type; // PACKET_VOXEL_DATA
    int chunk_x;
    int chunk_z;
    unsigned short block_count;
    VoxelBlock blocks[];
} NetVoxelPacket;
```

## Server Responsibilities
- Interpret BTN_ATTACK for Shankpit weapon fire (not a block interaction).
- Use yaw/pitch to build a view vector and perform a ray trace.
- Magnum (WPN_MAGNUM) range: **200 blocks**.
- Impact rules:
  - Entities (e.g., Zombies): apply 50 damage, send hit feedback.
  - Logs (block id 17): emit wood-chip particles.

## Client Responsibilities
- Parse `PACKET_VOXEL_DATA` for nearby blocks.
- Render log blocks with `texture_log.png` and leaves with `texture_leaves.png`.
- Use alpha cutout on leaves.

## Backend Direction
The server backend is moving to Go. The goal is to:
- Own authoritative weapon logic.
- Serialize simplified voxel packets.
- Maintain compatibility with existing C clients.
