package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/hainn191297/myDb/api/proto"
)

func main() {
	// Connect to server
	conn, err := grpc.NewClient("localhost:7001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewMyDBServiceClient(conn)

	// Setup data (using a fresh session)
	sessID := ""
	// Try to drop table, ignore error if not exists
	execIgnoreError(client, sessID, "DROP TABLE accounts")
	sessID = exec(client, sessID, "CREATE TABLE accounts (id INT PRIMARY KEY, balance INT)")
	sessID = exec(client, sessID, "INSERT INTO accounts VALUES (1, 1000)")

	fmt.Println("Data setup complete. Starting concurrent transactions...")

	var wg sync.WaitGroup
	wg.Add(2)

	// Txn 1: Updates balance, holds lock for 2 seconds
	go func() {
		defer wg.Done()
		fmt.Println("Txn 1: Starting")

		// Create new client connection for separate session
		conn1, _ := grpc.NewClient("localhost:7001", grpc.WithTransportCredentials(insecure.NewCredentials()))
		defer conn1.Close()
		c1 := pb.NewMyDBServiceClient(conn1)

		sid := ""
		sid = exec(c1, sid, "BEGIN")
		sid = exec(c1, sid, "UPDATE accounts SET balance = 900 WHERE id = 1")
		fmt.Println("Txn 1: Acquired lock, sleeping...")
		time.Sleep(2 * time.Second)
		sid = exec(c1, sid, "COMMIT")
		fmt.Println("Txn 1: Committed")
	}()

	// Txn 2: Tries to update same row immediately
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond) // Wait for Txn 1 to start
		fmt.Println("Txn 2: Starting")

		conn2, _ := grpc.NewClient("localhost:7001", grpc.WithTransportCredentials(insecure.NewCredentials()))
		defer conn2.Close()
		c2 := pb.NewMyDBServiceClient(conn2)

		sid := ""
		sid = exec(c2, sid, "BEGIN")
		start := time.Now()
		sid = exec(c2, sid, "UPDATE accounts SET balance = 800 WHERE id = 1")
		duration := time.Since(start)

		if duration < 1*time.Second {
			fmt.Printf("Txn 2: ERROR - Should have blocked! Duration: %v\n", duration)
		} else {
			fmt.Printf("Txn 2: Blocked as expected for %v\n", duration)
		}
		sid = exec(c2, sid, "COMMIT")
		fmt.Println("Txn 2: Committed")
	}()

	wg.Wait()
}

func execIgnoreError(c pb.MyDBServiceClient, sessID, sql string) string {
	resp, _ := c.ExecuteSQL(context.Background(), &pb.ExecuteSQLRequest{
		Sql:       sql,
		SessionId: sessID,
	})
	if resp != nil && resp.SessionId != "" {
		return resp.SessionId
	}
	return sessID
}

func exec(c pb.MyDBServiceClient, sessID, sql string) string {
	resp, err := c.ExecuteSQL(context.Background(), &pb.ExecuteSQLRequest{
		Sql:       sql,
		SessionId: sessID,
	})
	if err != nil {
		log.Fatalf("Failed to execute %s: %v", sql, err)
	}
	if resp.GetError() != nil {
		log.Fatalf("Server error executing %s: %s", sql, resp.GetError().Message)
	}
	return resp.SessionId
}
