# Buffer Pool Flow Explanation

The PlantUML sequence in `buffer_flow.puml` captures how the storage engine's buffer manager interacts with its internal data structures and the disk layer. This document walks through each phase so the diagram is easier to follow.

## Actors & Structures

- **Client Query** – executor or access path requesting pages, marking dirty, and releasing pages.
- **Buffer Manager (BM)** – hands out per-table buffer pools by combining TableManager (for data files) and WAL Manager (for log files).
- **Buffer Pool (BP)** – orchestrates lookups, frame allocation, dirty tracking, and eviction. Each pool operates on the FileManager it was constructed with and appends to its WAL logger before flushing pages.
- **Page Table (PT)** – maps `PageID -> frameID` so hits avoid disk I/O.
- **Frame / Slot** – container holding a page, pin count, and dirty flag.
- **Free List** – pool of preallocated empty frames used only for the first `capacity` misses. Once depleted, it stays empty.
- **LRU List** – replacement structure keeping frames ordered by recency.
- **Page Manager (PM)** – abstracts file layout and issues disk reads/writes.
- **Disk** – data files providing durable storage.

## Request Page Phase

1. **GetPage** – client calls `GetPage(schema, table, PageID)` into the Buffer Manager. BM reuses or creates a buffer pool: opens the table’s FileManager, obtains the WAL logger, and then BP asks the page table for a mapping.
2. **Cache Hit** – PT returns a frame ID. The frame's `pinCnt` increments to prevent eviction. BP returns the cached page.
3. **Cache Miss** – PT has no mapping, so BP must allocate a frame:
   - If the free list still has a preallocated entry, pop it and use that slot.
   - Otherwise consult the LRU list and repeatedly pick the least-recent frame until an unpinned victim appears. The evicted frame is reused immediately rather than returning to the free list.
4. **Dirty Victim Handling** – when the victim is dirty, BP flushes it via PM before removing the page table mapping and repurposing the frame.
5. **Read From Disk** – BP asks the table’s FileManager to load the requested page, which issues an OS read against the table's data file.
6. **Initialize Frame** – BP copies data into the frame, sets `pinCnt = 1`, `Dirty = false`, inserts it at the LRU front, and registers the mapping in the page table. The page is then returned to the client.

## Update Phase

- When a client modifies the in-memory page, it marks the frame dirty through BP (`MarkDirty`). Only the dirty flag flips; the page stays cached.

## Release Phase

- Clients call `Unpin(PageID)` once finished. BP decrements `pinCnt`.
- If the count hits zero, the frame becomes evictable, so it moves toward the LRU tail. Frames still pinned remain near the head to avoid eviction.

## Background Flush Phase

- Periodic or threshold-based processes (or explicit `FlushTable`) walk all frames, append each dirty page to the WAL, sync the WAL, then flush data pages via the FileManager so future evictions do not stall on writes while preserving WAL-before-data ordering.

This flow ensures the buffer manager balances fast access (cache hits) with correctness (pin counts, dirty tracking) and durability (flushes through the page manager) while keeping internal responsibilities explicit.

## Heap Table Layer

The heap table (`heap_flow.puml`) builds on top of the buffer manager:

- Inserts scan existing pages via BufferManager, initialize slotted pages when first touched, then write the tuple (key/value payload) and append to WAL before marking the frame dirty.
- Deletes simply clear the slot entry, allowing future inserts to reuse the directory entry while the free space can be reclaimed later.
- Scans iterate every page/slot, copying row bytes before returning data to higher layers.
