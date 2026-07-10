// Package market held the Phase 3 per-symbol gRPC pump that bridged the engine's
// StreamFills RPC into the hub. Phase 4 replaced it with a single Kafka
// consumer (see internal/kafka), which is both simpler and the event-driven
// design the architecture calls for.
//
// This file is intentionally left as a stub: the build environment cannot delete
// files. It carries no code and no imports so it is dead weight only in the
// repository tree, not in the binary. Safe to remove with `git rm`.
package market
