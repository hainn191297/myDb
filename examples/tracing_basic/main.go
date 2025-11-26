package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hainn191297/myDb/internal/logging"
)

// Simulates a request handler
func handleRequest(ctx context.Context) {
	logging.InfoContext(ctx, "Received query request")

	// Pass context to executor
	executeQuery(ctx)

	logging.InfoContext(ctx, "Request completed successfully")
}

// Simulates the executor layer
func executeQuery(ctx context.Context) {
	// Create a new span for executor
	ctx = logging.NewSpan(ctx, "executor")
	logging.DebugContext(ctx, "Starting query execution")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	// Pass context to planner
	planQuery(ctx)

	logging.DebugContext(ctx, "Execution completed")
}

// Simulates the planner layer
func planQuery(ctx context.Context) {
	// Create a new span for planner
	ctx = logging.NewSpan(ctx, "planner")
	logging.DebugContext(ctx, "Creating query plan")

	// Simulate planning work
	time.Sleep(5 * time.Millisecond)

	// Pass context to storage
	readFromStorage(ctx)

	logging.DebugContext(ctx, "Query plan created: TableScan")
}

// Simulates the storage layer
func readFromStorage(ctx context.Context) {
	// Create a new span for storage
	ctx = logging.NewSpan(ctx, "storage")
	logging.DebugContext(ctx, "Reading from heap table")

	// Simulate I/O work
	time.Sleep(15 * time.Millisecond)

	logging.DebugContext(ctx, "Read 5 rows from storage")
}

func main() {
	fmt.Println("=== Tracing Example: Single Request Flow ===")

	// Create a context with trace
	ctx := context.Background()
	ctx, meta := logging.WithTrace(ctx)

	fmt.Printf("Created trace: %s\n", meta.TraceID)
	fmt.Println("\nLogs showing trace flow:")

	// Handle the request
	handleRequest(ctx)

	fmt.Println("\n=== Analysis ===")
	fmt.Println("Từ logs trên, bạn có thể thấy:")
	fmt.Println("1. Cùng một trace ID cho toàn bộ request")
	fmt.Println("2. Các span khác nhau: root -> executor -> planner -> storage")
	fmt.Println("3. Mỗi component có span ID riêng để identify")
	fmt.Println("\nĐể grep logs theo trace ID:")
	fmt.Printf("  grep 'trace=%s' server.log\n", meta.TraceID)
}
