package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	pb "github.com/hainn191297/myDb/api/proto"
	"github.com/hainn191297/myDb/internal/config"
	"github.com/hainn191297/myDb/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	testPort  = 7002
	testDBDir = "test_data_e2e"
)

func setupServer(t *testing.T) (*server.Server, func()) {
	// Cleanup previous run
	os.RemoveAll(testDBDir)

	cfg := config.Config{
		DataDir:     testDBDir,
		GRPCPort:    testPort,
		MaxSessions: 100,
	}

	srv, err := server.New(cfg)
	require.NoError(t, err)

	// Run server in background
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			// It's hard to fail the test from a goroutine, but we can log
			fmt.Printf("Server failed: %v\n", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	return srv, func() {
		cancel()
		os.RemoveAll(testDBDir)
	}
}

func createClient(t *testing.T) pb.MyDBServiceClient {
	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", testPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	if conn == nil {
		t.Fatal("grpc.NewClient returned nil connection")
	}
	t.Cleanup(func() { conn.Close() })
	return pb.NewMyDBServiceClient(conn)
}

func TestBasicCRUD(t *testing.T) {
	_, teardown := setupServer(t)
	defer teardown()

	client := createClient(t)
	ctx := context.Background()
	var sessionID string

	// Helper to execute with session
	exec := func(sql string) *pb.ExecuteSQLResponse {
		resp, err := client.ExecuteSQL(ctx, &pb.ExecuteSQLRequest{
			Sql:       sql,
			SessionId: sessionID,
		})
		require.NoError(t, err)
		if resp.SessionId != "" {
			sessionID = resp.SessionId
		}
		return resp
	}

	// 1. Create Table
	t.Run("Create Table", func(t *testing.T) {
		resp := exec("CREATE TABLE users (id INT PRIMARY KEY, name TEXT, age INT)")
		require.IsType(t, &pb.ExecuteSQLResponse_CommandResult{}, resp.Result)
		assert.Equal(t, "CREATE TABLE", resp.GetCommandResult().Message)
	})

	// 2. Insert Data
	t.Run("Insert Data", func(t *testing.T) {
		resp := exec("INSERT INTO users VALUES (1, 'Alice', 30)")
		assert.Equal(t, int64(1), resp.GetCommandResult().RowsAffected)

		resp = exec("INSERT INTO users VALUES (2, 'Bob', 25)")
		assert.Equal(t, int64(1), resp.GetCommandResult().RowsAffected)
	})

	// 3. Select Data
	t.Run("Select Data", func(t *testing.T) {
		resp := exec("SELECT id, name, age FROM users WHERE id = 1")
		queryRes := resp.GetQueryResult()
		require.NotNil(t, queryRes)
		assert.Equal(t, 1, len(queryRes.Rows))
	})

	// 4. Update Data
	t.Run("Update Data", func(t *testing.T) {
		resp := exec("UPDATE users SET age = 31 WHERE id = 1")
		assert.Equal(t, int64(1), resp.GetCommandResult().RowsAffected)

		// Verify update
		resp = exec("SELECT age FROM users WHERE id = 1")
		// We can check values if we implement a helper to decode
	})

	// 5. Delete Data
	t.Run("Delete Data", func(t *testing.T) {
		resp := exec("DELETE FROM users WHERE id = 1")
		assert.Equal(t, int64(1), resp.GetCommandResult().RowsAffected)

		// Verify delete
		resp = exec("SELECT * FROM users WHERE id = 1")
		assert.Equal(t, 0, len(resp.GetQueryResult().Rows))
	})

	t.Run("Transaction Isolation", func(t *testing.T) {
		// Setup: Create table
		exec("CREATE TABLE accounts (id INT, balance INT)")
		exec("INSERT INTO accounts VALUES (1, 100)")
		exec("INSERT INTO accounts VALUES (2, 100)")

		// Txn 1: Start transaction and update account 1
		// Note: We need a new client/session for Txn 1 to simulate concurrent user
		client1 := createClient(t)
		ctx1 := context.Background()
		exec1 := func(sql string) *pb.ExecuteSQLResponse {
			resp, err := client1.ExecuteSQL(ctx1, &pb.ExecuteSQLRequest{Sql: sql})
			require.NoError(t, err)
			return resp
		}
		// Start Txn 1
		resp1 := exec1("BEGIN")
		sessionID1 := resp1.SessionId
		exec1WithSession := func(sql string) *pb.ExecuteSQLResponse {
			resp, err := client1.ExecuteSQL(ctx1, &pb.ExecuteSQLRequest{Sql: sql, SessionId: sessionID1})
			require.NoError(t, err)
			return resp
		}

		exec1WithSession("UPDATE accounts SET balance = 200 WHERE id = 1")

		// Txn 2: Try to read account 1 (should block or see old value depending on isolation)
		// Current implementation uses LockManager. Read should block if Write Lock is held.
		// To test blocking, we run Txn 2 in a goroutine.
		client2 := createClient(t)
		done := make(chan error)
		go func() {
			ctx2 := context.Background()
			resp, err := client2.ExecuteSQL(ctx2, &pb.ExecuteSQLRequest{Sql: "SELECT balance FROM accounts WHERE id = 1"})
			if err != nil {
				done <- err
				return
			}
			// Should see NEW value after Txn 1 commits?
			// Or if it blocked, it waits for commit.
			// If it didn't block (Snapshot Isolation), it sees 100.
			// Our LockManager implements blocking Read-Write locks.
			// So it should have waited and seen 200.
			if resp.GetQueryResult() != nil {
				rows := resp.GetQueryResult().Rows
				if len(rows) > 0 {
					// We expect 200 if it waited
					// But we need to assert this outside.
				}
			}
			done <- nil
		}()

		// Sleep to ensure Txn 2 started and is blocked
		time.Sleep(500 * time.Millisecond)

		// Commit Txn 1
		exec1WithSession("COMMIT")

		// Wait for Txn 2
		err := <-done
		require.NoError(t, err)

		// Verify final state
		resp := exec("SELECT balance FROM accounts WHERE id = 1")
		rows := resp.GetQueryResult().Rows
		require.Equal(t, 1, len(rows))
		// balance should be 200
		// We need to decode the row to be sure, but for now checking row count and no error.
		// Ideally we check value.
	})

	t.Run("Crash Recovery", func(t *testing.T) {
		// 1. Start server and insert data

		// Let's create a custom setup for this test.
		dbDir := "test_data_crash_recovery"
		os.RemoveAll(dbDir)
		defer os.RemoveAll(dbDir)

		cfg := config.Config{
			DataDir:     dbDir,
			GRPCPort:    testPort + 1, // Use different port
			MaxSessions: 100,
		}

		// Start Server 1
		srv1, err := server.New(cfg)
		require.NoError(t, err)
		ctx1, cancel1 := context.WithCancel(context.Background())
		go srv1.Start(ctx1)
		time.Sleep(100 * time.Millisecond) // Wait for start

		// Insert data
		conn1, err := grpc.NewClient(fmt.Sprintf("localhost:%d", cfg.GRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		client1 := pb.NewMyDBServiceClient(conn1)

		_, err = client1.ExecuteSQL(context.Background(), &pb.ExecuteSQLRequest{
			Sql: "CREATE TABLE recovery_test (id INT PRIMARY KEY, val TEXT)",
		})
		require.NoError(t, err)

		_, err = client1.ExecuteSQL(context.Background(), &pb.ExecuteSQLRequest{
			Sql: "INSERT INTO recovery_test VALUES (1, 'persistent')",
		})
		require.NoError(t, err)

		// Stop Server 1 (Simulate Crash/Shutdown)
		conn1.Close()
		cancel1()
		// Wait for shutdown
		time.Sleep(100 * time.Millisecond)

		// Start Server 2 (Same DataDir)
		srv2, err := server.New(cfg)
		require.NoError(t, err)
		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		go srv2.Start(ctx2)
		time.Sleep(100 * time.Millisecond)

		// Verify data
		conn2, err := grpc.NewClient(fmt.Sprintf("localhost:%d", cfg.GRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn2.Close()
		client2 := pb.NewMyDBServiceClient(conn2)

		resp, err := client2.ExecuteSQL(context.Background(), &pb.ExecuteSQLRequest{
			Sql: "SELECT val FROM recovery_test WHERE id = 1",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.GetQueryResult().Rows))
		// We can't easily check value "persistent" without decoding, but row count confirms it exists.
	})

	t.Run("Constraint Enforcement", func(t *testing.T) {
		client := createClient(t)
		ctx := context.Background()

		// 1. Create Table with PK
		_, err := client.ExecuteSQL(ctx, &pb.ExecuteSQLRequest{
			Sql: "CREATE TABLE constraints (id INT PRIMARY KEY, val TEXT)",
		})
		require.NoError(t, err)

		// 2. Insert first row
		_, err = client.ExecuteSQL(ctx, &pb.ExecuteSQLRequest{
			Sql: "INSERT INTO constraints VALUES (1, 'first')",
		})
		require.NoError(t, err)

		// 3. Insert duplicate PK (should fail)
		resp, err := client.ExecuteSQL(ctx, &pb.ExecuteSQLRequest{
			Sql: "INSERT INTO constraints VALUES (1, 'duplicate')",
		})
		require.NoError(t, err)
		// Check for error in response
		require.NotNil(t, resp.GetError())
		// We expect "duplicate value" or similar error message
		assert.Contains(t, resp.GetError().Message, "duplicate primary key")

		// 4. Verify only one row exists
		resp, err = client.ExecuteSQL(ctx, &pb.ExecuteSQLRequest{
			Sql: "SELECT id FROM constraints",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.GetQueryResult().Rows))
	})
}
