## Analysis of `.upload-temp/` Directory

### What It's For

The `.upload-temp/` directory lives inside your storage base directory (`storage/.upload-temp/`) and serves as the staging area for **resumable (chunked) uploads**. Here's how it works:

1. A client splits a large file into chunks
2. Each chunk is uploaded individually via `/api/v1/upload/chunk` → saved as `{uploadID}_{index}.tmp` in `.upload-temp/`
3. When all chunks arrive, the client calls `/api/v1/upload/complete` which:
   - Reads each `{uploadID}_{index}.tmp` chunk file from `.upload-temp/`
   - Assembles them into a single assembled temp file (`{uploadID}.tmp`)
   - Hashes and validates the combined file
   - Writes it to its final location in `storage/{user_id}/YYYY/MM/DD/`

The directory is **supposed** to be cleaned up after assembly completes.

---

### The Bug: Orphaned Chunk Files

There are two related bugs causing `.upload-temp/` to grow indefinitely with large items (especially videos):

#### Bug 1: `CleanupExpiredSessions()` only removes the assembled file, not chunk files
```go
// upload_handler_single.go line 247-258
func (m *UploadSessionManager) CleanupExpiredSessions() int {
    now := time.Now().Add(-24 * time.Hour)
    m.mu.Lock()
    defer m.mu.Unlock()

    count := 0
    for id, session := range m.sessions {
        if session.CreatedAt.Before(now) {
            os.Remove(session.tempPath) // ← Only removes the assembled .tmp file!
            delete(m.sessions, id)      // ← Individual chunk files ({id}_0.tmp, {id}_1.tmp...) are NEVER removed
            count++
        }
    }
    return count
}
```

After 24 hours of inactivity (e.g., a client crashes mid-upload), the session expires. The code removes `session.tempPath` which is just `{uploadID}.tmp` — but the individual chunk files like `{uploadID}_0.tmp`, `{uploadID}_1.tmp`, etc., are left behind as orphans.

#### Bug 2: `DeleteSession()` has the same problem
```go
// upload_handler_single.go line 238-245
func (m *UploadSessionManager) DeleteSession(sessionID string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    session := m.sessions[sessionID]
    if session != nil && session.tempPath != "" {
        os.Remove(session.tempPath) // ← Only removes the assembled .tmp file!
    }
    delete(m.sessions, sessionID) // ← Chunk files are NEVER removed
}
```

This is called by `HandleAbort` and after successful completion. In both cases, chunk files are orphaned.

#### Bug 3: The assembled temp path (`session.tempPath`) uses the upload ID but chunks use `{uploadID}_{index}.tmp`
The session stores only one temp path — the assembled file. But there can be many individual chunk files with different names. There is no tracking of which chunk indices belong to a session, so cleanup doesn't know what to delete.

---

### Impact

- **`.upload-temp/` grows indefinitely** as sessions expire or are aborted without cleaning up their chunks
- For video uploads (hundreds of MB to GB), this can fill up your storage volume quickly
- The `loadSessions()` function on startup will verify chunk existence and mark incomplete sessions, so recovery works — but the orphaned files from old expired sessions accumulate

## User management not displaying user role drop down selection unless relogin using disable cache

Suspect some race condition and auth - that it does not see the user is admin and then not displaying this but it allows to dispaly user mngt dialog in the first place!

