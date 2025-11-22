# In-Memory B+ Tree Implementation

This package (`mem`) contains a pure in-memory implementation of a B+ Tree.

**Purpose:**

- Serves as a playground for understanding B+ Tree algorithms (insertion, splitting, deletion, merging).
- Used for initial unit testing of B+ Tree logic without the complexity of disk I/O and page management.
- **NOT** intended for production use or integration with the main storage engine.

**Relationship to Engine:**
The actual on-disk B+ Tree engine will be implemented in `internal/storage/engine/btree`, which may reuse core logic from here (potentially refactored into a shared `core` package) or implement its own page-based logic.
