# Tracing Examples

This directory contains examples demonstrating the trace and span implementation in myDb.

## What is Tracing?

Tracing allows you to follow a request through different components of the system. Each request gets a unique **trace ID**, and each component/function creates a **span** (a unit of work) within that trace.

## Log Format

```
[trace=<32_hex_chars>|span=<16_hex_chars>:<span_name>]
```

- **Trace ID**: Unique identifier for the entire request (32 hex chars / 16 bytes)
- **Span ID**: Unique identifier for a specific operation (16 hex chars / 8 bytes)
- **Span Name**: Human-readable name of the component (e.g., "executor", "storage")

## Examples

### 1. Basic Flow (`tracing_example.go`)

Demonstrates a single request flowing through:
- `root` (request handler)
- `executor` (query execution)
- `planner` (query planning)
- `storage` (data access)

Run it:
```bash
go run examples/tracing_example.go
```

### 2. Concurrent Requests (`tracing_concurrent.go`)

Shows multiple concurrent requests, each with their own trace ID, processed simultaneously.

Run it:
```bash
go run examples/tracing_concurrent.go
```

## Usage in Your Code

### Creating a Trace

```go
import "github.com/hainn191297/myDb/internal/logging"

// At request entry point
ctx := context.Background()
ctx, meta := logging.WithTrace(ctx)
```

### Creating Child Spans

```go
// In each component/function
ctx = logging.NewSpan(ctx, "component_name")
logging.InfoContext(ctx, "Processing...")
```

### Logging with Context

```go
// Always pass context to logging functions
logging.InfoContext(ctx, "User logged in: %s", username)
logging.DebugContext(ctx, "Cache hit for key: %s", key)
logging.ErrorContext(ctx, "Failed to connect: %v", err)
```

## Analyzing Logs

### Find all logs for a specific request
```bash
grep 'trace=bc01aa202968bd82e58eb3e53ffac4c4' server.log
```

### Find all storage operations
```bash
grep 'storage' server.log
```

### Find all operations in a specific span type
```bash
grep 'span=.*:executor' server.log
```

## Benefits

1. **Request Tracing**: Follow a single request through the entire system
2. **Performance Analysis**: See which component takes the most time
3. **Debugging**: Isolate issues to specific components
4. **Concurrent Safety**: Multiple requests don't interfere with each other
5. **OpenTelemetry Compatible**: Uses standard trace/span ID sizes
