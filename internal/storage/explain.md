# Buffer Pool Flow Explanation

The PlantUML sequence in `flow.puml` captures how the storage engine's buffer manager interacts with its internal data structures and the disk layer. This document walks through each phase so the diagram is easier to follow.

## Actors & Structures

- **Client Query** – executor or access path requesting pages, marking dirty, and releasing pages.
- **Buffer Pool (BP)** – orchestrates lookups, frame allocation, dirty tracking, and eviction.
- **Page Table (PT)** – maps `PageID -> frameID` so hits avoid disk I/O.
- **Frame / Slot** – container holding a page, pin count, and dirty flag.
- **Free List** – list of unused frames. When empty, BP must evict via LRU.
- **LRU List** – replacement structure keeping frames ordered by recency.
- **Page Manager (PM)** – abstracts file layout and issues disk reads/writes.
- **Disk** – data files providing durable storage.

## Request Page Phase

1. **GetPage** – client calls `GetPage(PageID)`. BP asks the page table for a mapping.
2. **Cache Hit** – PT returns a frame ID. The frame's `pinCnt` increments to prevent eviction. BP returns the cached page.
3. **Cache Miss** – PT has no mapping, so BP must allocate a frame:
   - If the free list has entries, pop one.
   - Otherwise consult the LRU list and repeatedly pick the least-recent frame until an unpinned victim appears. Pinned frames are skipped to avoid evicting in-use pages.
4. **Dirty Victim Handling** – when the victim is dirty, BP flushes it via PM before removing the page table mapping.
5. **Read From Disk** – BP asks PM to load the requested page, which issues an OS read against the data file.
6. **Initialize Frame** – BP copies data into the frame, sets `pinCnt = 1`, `Dirty = false`, inserts it at the LRU front, and registers the mapping in the page table. The page is then returned to the client.

## Update Phase

- When a client modifies the in-memory page, it marks the frame dirty through BP (`MarkDirty`). Only the dirty flag flips; the page stays cached.

## Release Phase

- Clients call `Unpin(PageID)` once finished. BP decrements `pinCnt`.
- If the count hits zero, the frame becomes evictable, so it moves toward the LRU tail. Frames still pinned remain near the head to avoid eviction.

## Background Flush Phase

- Periodic or threshold-based processes can walk the LRU list to flush cold dirty pages proactively.
- Candidates that are dirty and unpinned get written back via PM → Disk so future evictions do not stall on writes.

This flow ensures the buffer manager balances fast access (cache hits) with correctness (pin counts, dirty tracking) and durability (flushes through the page manager) while keeping internal responsibilities explicit.
