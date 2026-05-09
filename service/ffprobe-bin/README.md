Place platform ffprobe binaries here for embedded build mode.

Expected files:

- `windows-amd64/ffprobe.exe`
- `linux-amd64/ffprobe`
- `linux-arm64/ffprobe`

Build with:

`go build -tags embed_ffprobe .`

If not using `embed_ffprobe`, the service falls back to:

1. Side-by-side ffprobe file
2. ffprobe from PATH
3. HTTP range metadata parser
