package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/hainn191297/myDb/api/proto"
	"github.com/hainn191297/myDb/internal/schema"
)

const (
	version = "0.1.0"
	prompt  = "mydb> "
)

// Client wraps gRPC client and readline interface
type Client struct {
	conn      *grpc.ClientConn
	client    pb.MyDBServiceClient
	rl        *readline.Instance
	sessionID string
}

func main() {
	serverAddr := flag.String("host", "localhost:7001", "MyDB server address")
	flag.Parse()

	if err := run(*serverAddr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(serverAddr string) error {
	// Connect to server
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client := &Client{
		conn:   conn,
		client: pb.NewMyDBServiceClient(conn),
	}

	// Setup readline
	rl, err := readline.New(prompt)
	if err != nil {
		return fmt.Errorf("failed to init readline: %w", err)
	}
	defer rl.Close()
	client.rl = rl

	// Print welcome message
	fmt.Printf("MyDB v%s - Connected to %s\n", version, serverAddr)
	fmt.Println("Type 'help' for help, '\\q' to exit")
	fmt.Println()

	// Start REPL
	return client.repl()
}

// repl runs the read-eval-print loop
func (c *Client) repl() error {
	for {
		line, err := c.rl.Readline()
		if err != nil {
			if err == io.EOF || err == readline.ErrInterrupt {
				fmt.Println("\nBye!")
				return nil
			}
			return err
		}

		// Skip empty lines
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(line, "\\") {
			if err := c.handleMetaCommand(line); err != nil {
				if err == io.EOF {
					fmt.Println("Bye!")
					return nil
				}
				fmt.Printf("Error: %v\n", err)
			}
			continue
		}

		// Handle help command
		if strings.ToLower(line) == "help" {
			c.printHelp()
			continue
		}

		// Execute SQL
		if err := c.executeSQL(line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// executeSQL sends SQL to server and displays results
func (c *Client) executeSQL(sql string) error {
	resp, err := c.client.ExecuteSQL(context.Background(), &pb.ExecuteSQLRequest{
		Sql:       sql,
		SessionId: c.sessionID,
	})
	if err != nil {
		return fmt.Errorf("RPC error: %w", err)
	}

	// Update session ID if changed
	if resp.SessionId != "" {
		c.sessionID = resp.SessionId
	}

	switch result := resp.Result.(type) {
	case *pb.ExecuteSQLResponse_QueryResult:
		c.printQueryResult(result.QueryResult)
	case *pb.ExecuteSQLResponse_CommandResult:
		fmt.Println(result.CommandResult.Message)
	case *pb.ExecuteSQLResponse_Error:
		fmt.Printf("Server error [%s]: %s\n", result.Error.Code, result.Error.Message)
	default:
		return fmt.Errorf("unexpected response type")
	}

	return nil
}

// printQueryResult formats and displays query results
func (c *Client) printQueryResult(result *pb.QueryResult) {
	if len(result.Rows) == 0 {
		fmt.Println("(0 rows)")
		return
	}

	// Calculate column widths
	widths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		widths[i] = len(col)
	}
	for _, row := range result.Rows {
		for i, val := range row.Values {
			if i < len(widths) {
				width := len(formatValue(val))
				if width > widths[i] {
					widths[i] = width
				}
			}
		}
	}

	// Print header separator
	c.printSeparator(widths)

	// Print column headers
	fmt.Print("|")
	for i, col := range result.Columns {
		fmt.Printf(" %-*s |", widths[i], col)
	}
	fmt.Println()

	// Print separator
	c.printSeparator(widths)

	// Print rows
	for _, row := range result.Rows {
		fmt.Print("|")
		for i, val := range row.Values {
			if i < len(widths) {
				fmt.Printf(" %-*s |", widths[i], formatValue(val))
			}
		}
		fmt.Println()
	}

	// Print bottom separator
	c.printSeparator(widths)

	// Print row count
	fmt.Printf("(%d %s)\n", len(result.Rows), pluralize(len(result.Rows), "row", "rows"))
}

// printSeparator prints a table separator line
func (c *Client) printSeparator(widths []int) {
	fmt.Print("+")
	for _, width := range widths {
		fmt.Print(strings.Repeat("-", width+2))
		fmt.Print("+")
	}
	fmt.Println()
}

// formatValue converts a byte value to display string
func formatValue(val []byte) string {
	if len(val) == 0 {
		return "NULL"
	}
	// Try to decode as different types
	if decoded, err := schema.TypeInt64.Decode(val); err == nil {
		return fmt.Sprintf("%d", decoded)
	}
	if decoded, err := schema.TypeFloat64.Decode(val); err == nil {
		return fmt.Sprintf("%v", decoded)
	}
	if decoded, err := schema.TypeBool.Decode(val); err == nil {
		return fmt.Sprintf("%v", decoded)
	}
	// Default to string
	if decoded, err := schema.TypeText.Decode(val); err == nil {
		return fmt.Sprintf("%s", decoded)
	}
	// Fallback to raw bytes
	return string(val)
}

// pluralize returns singular or plural form based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// handleMetaCommand processes backslash commands
func (c *Client) handleMetaCommand(cmd string) error {
	switch {
	case cmd == "\\q" || cmd == "\\quit":
		return io.EOF

	case cmd == "\\dt":
		return c.listTables()

	case strings.HasPrefix(cmd, "\\d "):
		tableName := strings.TrimSpace(cmd[3:])
		return c.describeTable(tableName)

	case cmd == "\\?" || cmd == "\\help":
		c.printHelp()
		return nil

	default:
		return fmt.Errorf("unknown command: %s (try \\help)", cmd)
	}
}

// listTables displays all tables
func (c *Client) listTables() error {
	resp, err := c.client.GetMetadata(context.Background(), &pb.GetMetadataRequest{
		Schema: "public",
	})
	if err != nil {
		return fmt.Errorf("RPC error: %w", err)
	}

	if len(resp.Tables) == 0 {
		fmt.Println("No tables found.")
		return nil
	}

	fmt.Println("Tables:")
	for _, table := range resp.Tables {
		fmt.Printf("  %s.%s\n", table.Schema, table.Name)
	}
	return nil
}

// describeTable shows table structure
func (c *Client) describeTable(tableName string) error {
	resp, err := c.client.GetMetadata(context.Background(), &pb.GetMetadataRequest{
		Schema: "public",
	})
	if err != nil {
		return fmt.Errorf("RPC error: %w", err)
	}

	var found *pb.TableInfo
	for _, table := range resp.Tables {
		if table.Name == tableName {
			found = table
			break
		}
	}

	if found == nil {
		return fmt.Errorf("table not found: %s", tableName)
	}

	fmt.Printf("Table: %s.%s\n", found.Schema, found.Name)
	fmt.Println("\nColumns:")
	for _, col := range found.Columns {
		flags := ""
		if col.PrimaryKey {
			flags = " PRIMARY KEY"
		} else if !col.Nullable {
			flags = " NOT NULL"
		}
		fmt.Printf("  %-20s %-10s%s\n", col.Name, col.Type, flags)
	}

	if len(found.Indexes) > 0 {
		fmt.Println("\nIndexes:")
		for _, idx := range found.Indexes {
			unique := ""
			if idx.Unique {
				unique = " UNIQUE"
			}
			fmt.Printf("  %-20s%s (%s)\n", idx.Name, unique, strings.Join(idx.Columns, ", "))
		}
	}

	return nil
}

// printHelp displays help information
func (c *Client) printHelp() {
	fmt.Println("MyDB Interactive Client")
	fmt.Println()
	fmt.Println("SQL Commands:")
	fmt.Println("  Any valid SQL statement (CREATE TABLE, SELECT, INSERT, etc.)")
	fmt.Println()
	fmt.Println("Meta Commands:")
	fmt.Println("  \\q, \\quit       Quit the client")
	fmt.Println("  \\dt             List all tables")
	fmt.Println("  \\d <table>      Describe table structure")
	fmt.Println("  \\?, \\help       Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mydb> CREATE TABLE users (id INT PRIMARY KEY, name TEXT);")
	fmt.Println("  mydb> INSERT INTO users VALUES (1, 'Alice');")
	fmt.Println("  mydb> SELECT * FROM users;")
	fmt.Println("  mydb> \\dt")
	fmt.Println("  mydb> \\d users")
}
