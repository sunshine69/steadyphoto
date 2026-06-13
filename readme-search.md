# Search & Discovery Feature

**Status:** ✅ Completed  
**Last Updated:** June 13, 2026

---

## Overview

Client-side search and filtering across the media grid with three scope options:

| Scope | Behavior | Implementation |
|-------|----------|----------------|
| **All** (default) | Searches both filename AND tags; returns items matching either criterion | Combines results using OR logic, deduplicates by ID |
| **Name Only** | Filters by `filename` containing the search term (case-insensitive substring match) | Simple string `.includes()` on lowercased filenames |
| **Tags Only** | Filters by tag values containing the search term | Splits tags field (`:`-delimited), checks each tag for substring match |

---

## Additional Features

- ✅ **Tag URL Filter**: `?tag=...` query parameter filters grid to items matching that specific tag (case-insensitive). Clear filter button resets view. Server-side implementation at `/api/v1/media/search`.

---

## Related Documents

- [Architecture Overview](readme-arch.md) — High-level system architecture
