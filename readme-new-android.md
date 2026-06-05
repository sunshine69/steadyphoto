Android / Kotlin layer

The UI exposes a sync button and a progress screen. On tap, it checks permissions (READ_MEDIA_IMAGES, READ_MEDIA_VIDEO, POST_NOTIFICATIONS) then starts the foreground service. The foreground service holds a persistent notification ("Syncing 12 of 84 files…") and prevents the process from being killed while active. When the user presses home, it keeps running.
ContentObserver registers against MediaStore.Images.Media.EXTERNAL_CONTENT_URI and MediaStore.Video.Media.EXTERNAL_CONTENT_URI. When Android fires a change event (new photo taken, downloaded, etc.), the observer enqueues a OneTimeWorkRequest via WorkManager with NetworkType.CONNECTED and BatteryNotLow constraints. This way the sync respects battery and connectivity without you managing it manually.
WorkManager also runs a periodic job (e.g. every 15 minutes or on WiFi reconnect) to catch anything missed.

gomobile bridge
You compile the Go package with gomobile bind -target android. This generates an .aar you drop into the Gradle project. Kotlin calls into it like a regular library. The Go side exposes a clean API:
gofunc StartSync(rootPath string, token string, callback SyncCallback)
func CancelSync()

type SyncCallback interface {
    OnProgress(done int64, total int64)
    OnError(path string, err string)
    OnComplete()
}
Kotlin implements SyncCallback and the Go engine calls back on progress.

Go sync engine
The pipeline is: scan → hash → compare → queue → upload.

Scanner walks the paths provided by Kotlin (resolved from MediaStore cursors), filtering by MIME type and last-modified timestamp.
Hasher computes xxHash64 (fast) or SHA-256 (stronger) per file. The hash is the source of truth for "already uploaded."
Metadata extractor reads EXIF (using a Go EXIF library) for date, GPS, camera model — useful for server-side deduplication.
Local index is a SQLite DB (via mattn/go-sqlite3 or modernc.org/sqlite for pure Go) storing (path, hash, remote_id, state, last_synced).
Sync comparator diffs the local index against the remote manifest (fetched as JSON). Files missing from remote or with different hashes go into the upload queue.
Conflict handler: if hash differs but mtime is close, the server wins (or user preference). If the local file is newer, it queues an overwrite.
Upload queue is a buffered channel feeding worker goroutines (e.g. 3 concurrent uploads).
Resumable uploads use the TUS protocol — the server remembers the offset, so a dropped WiFi connection resumes where it left off rather than restarting.
Retry/backoff uses exponential backoff with jitter: 100ms * 2^attempt + rand(0..100ms), max 5 retries before marking a file as failed.


Flow when you press sync

Kotlin calls SyncEngine.StartSync(paths, token, callback)
Go scans, hashes, builds a diff against the remote manifest
Uploads begin; progress callbacks update the notification
On home press — foreground service keeps the process alive; Go continues uploading
On completion, service posts a summary notification and stops itself

Flow on new media event

ContentObserver.onChange() fires
New OneTimeWorkRequest enqueued with network/battery constraints
When constraints are met, WorkManager starts a worker that calls SyncEngine.StartSync() for just the new file(s)


Key dependencies
LayerLibraryGo mobilegolang.org/x/mobile/cmd/gomobileSQLite (Go)modernc.org/sqlite (pure Go, no CGO issues)TUS client (Go)github.com/tus/tusd/pkg/handler or customEXIF (Go)github.com/rwcarlsen/goexif/exifWorkManagerandroidx.work:work-runtime-ktx
This is a solid, well-separated architecture. The Go engine is fully testable on desktop and the Android layer stays thin
