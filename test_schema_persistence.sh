#!/bin/bash
# Schema Persistence Test Script

echo "=== Testing Schema Persistence ==="
echo ""

# Function to execute SQL and show result
exec_sql() {
    local sql="$1"
    echo "SQL: $sql"
    echo "$sql" | ./bin/mydb-cli 2>&1 | grep -v "^MyDB" | grep -v "^mydb>" | head -20
    echo ""
}

echo "Step 1: Create table with PRIMARY KEY"
exec_sql "CREATE TABLE users (id INT PRIMARY KEY, name TEXT, age INT);"
sleep 1

echo "Step 2: Insert data"
exec_sql "INSERT INTO users VALUES (1, 'Alice', 30);"
exec_sql "INSERT INTO users VALUES (2, 'Bob', 25);"
sleep 1

echo "Step 3: Query data (before restart)"
exec_sql "SELECT * FROM users;"
sleep 1

echo "Step 4: Check catalog files"
ls -lh data/__system/
echo ""

echo "Step 5: Restart server"
echo "Stopping server..."
pkill mydbd
sleep 2

echo "Starting server..."
./bin/mydbd &
sleep 2

echo ""
echo "Step 6: Query data (after restart) - WITHOUT recreating table"
exec_sql "SELECT * FROM users;"

echo "Step 7: Verify schema metadata"
exec_sql "SELECT id, name FROM users WHERE id = 1;"

echo ""
echo "=== Test Complete ==="
echo "If you see Alice and Bob above, schema persistence WORKS! ✅"
