# DataPushGateway AI Coding Instructions

## Project Overview

DataPushGateway is a Go HTTP server that receives monitoring data via HTTP POST, organizes it into Markdown files based on configurable tag-based routing, and syncs the results to a Perforce Helix Core server. It's a companion to Prometheus Pushgateway, designed for tracking Perforce server environment data over time using version control.

**Key Flow**: Client POST → Basic Auth → JSON/Data Processing → Markdown Generation → Perforce Sync/Submit

## Architecture

### Core Components

1. **HTTP Server** ([main.go](main.go)): Three endpoints:
   - `/` - Health check
   - `/json/` - Structured monitoring data (JSON array with `monitor_tag`, `description`, `output`)
   - `/data/` - Simple data endpoint (query params: `customer`, `instance`)

2. **Authentication** ([functions/auth.go](functions/auth.go)): 
   - Basic auth with plaintext passwords from `auth.yaml`
   - Users defined as array with `username`, `password`, and `url_prefix` fields

3. **Data Processing** ([functions/data.go](functions/data.go)):
   - Reads `config.yaml` `file_configs` to route data by `monitor_tags`
   - Groups JSON items by tag, creates Markdown files with base64-decoded outputs
   - Supports `%INSTANCE%` placeholder in file names and directories

4. **Perforce Integration** ([functions/p4cmds.go](functions/p4cmds.go)):
   - Uses P4CONFIG file for connection settings (configured in `config.yaml`)
   - Workflow: `p4 rec` → `p4 sync` → `p4 resolve -ay` → `p4 submit` (if changes exist)
   - Auto-login with ticket validation

## Configuration Pattern

**Critical**: Configuration uses TWO files that reference each other:

1. **config.yaml**: 
   - `applicationConfig.P4CONFIG` → points to `.p4config` file location
   - `applicationConfig.p4bin` → p4 executable path
   - `file_configs[]` → array of tag-based routing rules

2. **.p4config** (referenced in config.yaml):
   - Standard Perforce environment: `P4PORT`, `P4USER`, `P4CLIENT`, `P4TICKETS`, `P4TRUST`

3. **auth.yaml**: `users` array with `username`, `password`, and `url_prefix` fields

## Development Workflows

### Building
```bash
make build              # Local build
make dist              # Cross-platform builds to bin/
```

### Running Locally
```bash
./datapushgateway \
  --auth.file auth.yaml \
  --config config.yaml \
  --port :9092 \
  --debug \
  --log /var/log/datapushgateway.log \
  --data data
```

### Testing Data Submission

**JSON endpoint** (for structured monitoring):
```bash
curl -X POST -u test:test \
  'http://localhost:9092/json/?customer=acme&instance=prod' \
  -H 'Content-Type: application/json' \
  -d '[{"monitor_tag":"p4 configure","description":"P4 Settings","output":"BASE64_ENCODED_DATA"}]'
```

**Data endpoint** (for simple text):
```bash
curl -X POST -u test:test \
  'http://localhost:9092/data/?customer=acme&instance=prod' \
  -d 'Server info data here'
```

## Code Conventions

- **Logging**: Use `logrus.Logger` passed as function parameter; debug logs behind `logger.Level == logrus.DebugLevel` check
- **Error Handling**: Return errors up the stack; log at point of HTTP response
- **Validation**: Customer/instance names validated with regex: `^[a-zA-Z0-9_-]+$`
- **P4 Commands**: Always set `P4CONFIG` env var before exec; use `RunP4CommandWithEnvAndDir` wrapper
- **Base64**: JSON endpoint expects `output` field base64-encoded; decoded when writing Markdown

## Critical Patterns

### Tag-Based Routing
The `config.yaml` `file_configs` array determines which monitor tags go into which Markdown files:
```yaml
- file_name: support
  directory: servers/%INSTANCE%
  monitor_tags:
    - p4 configure
    - p4 ztag
```

When processing JSON with `"monitor_tag": "p4 configure"`, it routes to `data/{customer}/servers/{instance}/support.md`.

### Perforce State Management
- Ticket validity checked with `HasValidTicket()` before operations
- Trust handled automatically with `p4 trust -y`
- Submit only if `p4 opened` shows pending changes
- All commands use `-d` flag to specify workspace root: `filepath.Join(dataDir, customer)`

### Middleware Pattern
Connection logging is a middleware wrapper around handlers:
```go
ConnectionLoggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        // Log connection details
        next.ServeHTTP(w, req)
    }
}
```

## Integration Points

- **External Clients**: `command-runner` or `report_instance_data.sh` from p4prometheus ecosystem
- **Perforce Server**: Requires bot user (e.g., `bot_HRA_instance_monitor`) with unlimited timeout
- **File System**: `--data` directory becomes workspace root; must match P4CLIENT Root setting

## Common Gotchas

1. **P4CONFIG Path**: Must be absolute path in `config.yaml`; relative paths fail silently
2. **Instance Placeholder**: Use `%INSTANCE%` in config, gets replaced at runtime in both filenames and directories
3. **Empty Markdown**: Files with no content are automatically removed after creation
4. **Order Preservation**: Monitor tags processed in config.yaml order for consistent file structure
5. **Plaintext Passwords**: `auth.yaml` uses plaintext passwords - ensure file permissions are restrictive (600)

## Version Information

Build versioning uses git tags and `p4prometheus/version` module via ldflags injection in Makefile.
