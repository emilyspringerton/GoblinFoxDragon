# Correctness Checklist

Use this checklist before merging changes that touch the Shankpit ↔ Bedrock bridge or Go backend.

## Protocol & Serialization
- [ ] Packet constants match between C and Go definitions.
- [ ] Struct field ordering and padding match the C wire layout.
- [ ] Any new packets are documented with field sizes and semantics.

## Weapon & Hitscan Logic
- [ ] Weapon range values match design (e.g., Magnum = 200 blocks).
- [ ] Raycast uses player eye position, not feet position.
- [ ] Entity damage and feedback events are emitted on impact.

## Client Rendering
- [ ] Voxel block IDs map to the correct textures.
- [ ] Alpha cutout enabled only for leaf blocks.

## Testing
- [ ] Unit tests cover math helpers, packet parsing, and hitscan handling.
- [ ] `go test ./...` passes locally and in CI.
