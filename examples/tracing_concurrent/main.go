package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hainn191297/myDb/internal/logging"
)

// Simulates processing a query with full trace
func processQuery(ctx context.Context, queryID int, query string) {
	logging.InfoContext(ctx, "Query #%d: %s", queryID, query)

	// Executor stage
	ctx = logging.NewSpan(ctx, "executor")
	logging.DebugContext(ctx, "Parsing and validating query")
	time.Sleep(5 * time.Millisecond)

	// Planner stage
	ctx = logging.NewSpan(ctx, "planner")
	logging.DebugContext(ctx, "Creating execution plan")
	time.Sleep(3 * time.Millisecond)

	// Storage stage
	ctx = logging.NewSpan(ctx, "storage")
	logging.DebugContext(ctx, "Accessing data from disk")
	time.Sleep(10 * time.Millisecond)

	// Back to executor
	ctx = logging.NewSpan(ctx, "executor")
	logging.InfoContext(ctx, "Query #%d completed: 3 rows affected", queryID)
}

func main() {
	fmt.Println("=== Advanced Tracing Example: Multiple Concurrent Requests ===\n")

	queries := []string{
		"SELECT * FROM users WHERE id = 1",
		"INSERT INTO users VALUES (2, 'Alice')",
		"UPDATE users SET name = 'Bob' WHERE id = 1",
	}

	var wg sync.WaitGroup

	fmt.Println("Starting 3 concurrent queries...\n")

	for i, query := range queries {
		wg.Add(1)
		go func(queryID int, q string) {
			defer wg.Done()

			// Each request gets its own trace
			ctx := context.Background()
			ctx, meta := logging.WithTrace(ctx)

			fmt.Printf("Query #%d trace: %s\n", queryID+1, meta.TraceID)
			processQuery(ctx, queryID+1, q)
		}(i, query)

		// Stagger requests slightly
		time.Sleep(2 * time.Millisecond)
	}

	wg.Wait()

	fmt.Println("\n=== How to Use Traces ===")
	fmt.Println("\n1. Grep logs by trace ID to see full request flow:")
	fmt.Println("   grep 'trace=<trace_id>' server.log")
	fmt.Println("\n2. Filter by specific span:")
	fmt.Println("   grep 'span=.*:storage' server.log")
	fmt.Println("\n3. See all executor operations:")
	fmt.Println("   grep 'executor' server.log")
	fmt.Println("\n4. Trace format: [trace=<32_hex>|span=<16_hex>:<name>]")
	fmt.Println("   - Same trace ID = same request")
	fmt.Println("   - Different span IDs = different stages")
	fmt.Println("   - Span names identify components")
}
