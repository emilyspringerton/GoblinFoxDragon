# GFD Netcode Contract Spec

> **Canonical source:** `SHANKPIT/docs2/NETCODE_CONTRACT_SPEC.md`
>
> SHANKPIT owns the canonical UDP netcode contract. This file is a reference pointer.
> Do not edit the spec here — edit it in SHANKPIT and the change applies to GFD automatically.
>
> The GFD world backend (DragonsNShit) extends the netcode contract with voxel streaming
> packets (`PACKET_VOXEL_DATA`). Those extensions are defined in:
> - `docs2/specs/THE_BRIDGE_SPEC.md` — SHANKPIT ↔ DragonsNShit wire contract
> - `docs2/specs/SHANKPIT_DRAGONSNSHIT_SYSTEMS_SPEC.md` — full bridge systems spec
>
> If GFD needs a netcode extension that SHANKPIT does not, write the extension here as an
> addendum doc rather than forking the canonical spec.
