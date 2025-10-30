#!/bin/bash

echo "=== Hot Fixture Tool - Clean NDJSON Streaming Demo ==="
echo ""

echo "The hfit CLI now outputs pure NDJSON streams without formatting - perfect for developers!"
echo ""

echo "1. Database Table Rows Streaming:"
echo "hfit rows mysql mydb users"
echo "Output:"
echo '{"id":1,"name":"John Doe","email":"john@example.com","age":30}'
echo '{"id":2,"name":"Jane Smith","email":"jane@example.com","age":25}'
echo '{"id":3,"name":"Bob Johnson","email":"bob@example.com","age":35}'
echo ""

echo "2. Database Rows with SQL Filter:"
echo "hfit rows postgres mydb orders \"WHERE status = 'active' ORDER BY created_at DESC LIMIT 5\""
echo "Output:"
echo '{"id":101,"customer_id":5,"total":299.99,"status":"active","created_at":"2024-10-30T10:15:30Z"}'
echo '{"id":98,"customer_id":3,"total":149.50,"status":"active","created_at":"2024-10-30T09:22:15Z"}'
echo '{"id":95,"customer_id":7,"total":599.00,"status":"active","created_at":"2024-10-30T08:45:22Z"}'
echo ""

echo "3. File Listing Streaming:"
echo "hfit files volume1"
echo "Output:"
echo '{"name":"app.log","path":"logs/app.log","size":2048,"modtime":1735574400,"isdir":false}'
echo '{"name":"error.log","path":"logs/error.log","size":1024,"modtime":1735574401,"isdir":false}'
echo '{"name":"config.yaml","path":"config/config.yaml","size":512,"modtime":1735574350,"isdir":false}'
echo ""

echo "4. File Listing with Filters:"
echo "hfit files volume1 \"name:*.log\" \"size:>1000\""
echo "Output:"
echo '{"name":"app.log","path":"logs/app.log","size":2048,"modtime":1735574400,"isdir":false}'
echo '{"name":"error.log","path":"logs/error.log","size":1024,"modtime":1735574401,"isdir":false}'
echo ""

echo "=== Developer-Friendly Features ==="
echo ""
echo "✅ Pure NDJSON: No headers, counters, or formatting"
echo "✅ Streamable: Perfect for piping to jq, grep, awk, etc."
echo "✅ Real-time: Data appears as it's processed"
echo "✅ Memory efficient: O(1) memory usage"
echo "✅ Unix-friendly: Integrates with standard Unix tools"
echo ""

echo "=== Common Usage Patterns ==="
echo ""
echo "# Count rows"
echo "hfit rows mysql mydb users | wc -l"
echo ""
echo "# Extract specific fields with jq"
echo "hfit rows mysql mydb users | jq -r '.name'"
echo ""
echo "# Filter and format"
echo "hfit files volume1 \"name:*.log\" | jq -r '\"\\(.name): \\(.size) bytes\"'"
echo ""
echo "# Pipe to other tools"
echo "hfit rows postgres mydb orders \"WHERE total > 100\" | grep '\"status\":\"active\"'"
echo ""
echo "# Save to file"
echo "hfit rows mysql mydb users \"ORDER BY created_at DESC\" > users_backup.ndjson"
echo ""

echo "🚀 Perfect for automation, scripting, and integration with other tools!"