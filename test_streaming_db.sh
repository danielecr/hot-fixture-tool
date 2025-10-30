#!/bin/bash

echo "=== Hot Fixture Tool - Streaming Database Rows Demo ==="
echo ""

echo "Example API calls for streaming database table rows:"
echo ""

echo "1. Traditional JSON Array Response (backward compatibility):"
echo "curl -H 'Authorization: Bearer \$JWT_TOKEN' \\"
echo "     'http://localhost:8080/db/mysql/mydb/table/users/rows'"
echo ""

echo "2. Streaming NDJSON Response (optimized for large datasets):"
echo "curl -H 'Authorization: Bearer \$JWT_TOKEN' \\"
echo "     -H 'Accept: application/x-json-stream' \\"
echo "     'http://localhost:8080/db/mysql/mydb/table/users/rows'"
echo ""

echo "3. Filtered Queries with SQL filterpart parameter:"
echo ""

echo "• Filter with WHERE clause:"
echo "curl -H 'Authorization: Bearer \$JWT_TOKEN' \\"
echo "     -H 'Accept: application/x-json-stream' \\"
echo "     'http://localhost:8080/db/mysql/mydb/table/users/rows?filterpart=WHERE age > 25'"
echo ""

echo "• Sort with ORDER BY:"
echo "curl -H 'Authorization: Bearer \$JWT_TOKEN' \\"
echo "     -H 'Accept: application/x-json-stream' \\"
echo "     'http://localhost:8080/db/postgres/mydb/table/orders/rows?filterpart=ORDER BY created_at DESC'"
echo ""

echo "• Complex filtering with LIMIT:"
echo "curl -H 'Authorization: Bearer \$JWT_TOKEN' \\"
echo "     -H 'Accept: application/x-json-stream' \\"
echo "     'http://localhost:8080/db/mysql/mydb/table/products/rows?filterpart=WHERE price > 100 ORDER BY name ASC LIMIT 50'"
echo ""

echo "• Group and aggregate data:"
echo "curl -H 'Authorization: Bearer \$JWT_TOKEN' \\"
echo "     -H 'Accept: application/x-json-stream' \\"
echo "     'http://localhost:8080/db/postgres/analytics/table/sales/rows?filterpart=GROUP BY category HAVING COUNT(*) > 10'"
echo ""

echo "=== NDJSON Streaming Response Format ==="
echo "Content-Type: application/x-json-stream"
echo ""
echo '{\"id\":1,\"name\":\"John Doe\",\"email\":\"john@example.com\",\"age\":30}'
echo '{\"id\":2,\"name\":\"Jane Smith\",\"email\":\"jane@example.com\",\"age\":25}'
echo '{\"id\":3,\"name\":\"Bob Johnson\",\"email\":\"bob@example.com\",\"age\":35}'
echo ""

echo "=== SQL Injection Protection ==="
echo "✅ Validates filterpart parameter before execution"
echo "✅ Only allows safe SQL clauses: WHERE, ORDER BY, LIMIT, HAVING, GROUP BY"
echo "✅ Blocks dangerous keywords: INSERT, UPDATE, DELETE, DROP, etc."
echo "✅ Checks for balanced quotes and suspicious patterns"
echo "✅ Rejects potentially malicious content"
echo ""

echo "=== Blocked Examples (Security Protection) ==="
echo "❌ DROP TABLE users; -- (dangerous operation)"
echo "❌ INSERT INTO users -- (data modification)"
echo "❌ SELECT * FROM information_schema -- (schema exposure)"
echo "❌ UNION SELECT password FROM -- (data injection)"
echo "❌ WHERE id = 1; DELETE FROM -- (SQL injection)"
echo ""

echo "=== Performance Benefits ==="
echo "✅ Real-time streaming: Rows appear as they're processed"
echo "✅ Memory efficient: O(1) memory usage regardless of result size"
echo "✅ Scalable: Handles millions of rows without memory constraints"
echo "✅ Early termination: Can cancel streaming requests"
echo "✅ Network efficient: Progressive loading vs large JSON arrays"
echo ""

echo "=== Supported Database Systems ==="
echo "• MySQL: Full filterpart support with MySQL-specific syntax"
echo "• PostgreSQL: Full filterpart support with PostgreSQL-specific syntax"
echo "• Multi-DBMS: Automatic provider selection based on configuration"
echo ""

echo "=== Usage Examples by Database Type ==="
echo ""

echo "MySQL Examples:"
echo "• Date filtering: WHERE DATE(created_at) >= '2024-01-01'"
echo "• Text search: WHERE name LIKE '%smith%'"
echo "• Numeric range: WHERE price BETWEEN 10 AND 100"
echo "• Complex joins: WHERE category_id IN (SELECT id FROM categories WHERE active = 1)"
echo ""

echo "PostgreSQL Examples:"
echo "• JSON queries: WHERE metadata->>'status' = 'active'"
echo "• Array operations: WHERE tags @> '[\"important\"]'"
echo "• Full-text search: WHERE to_tsvector('english', description) @@ to_tsquery('keyword')"
echo "• Window functions: ORDER BY ROW_NUMBER() OVER (PARTITION BY department_id ORDER BY salary DESC)"
echo ""

echo "=== Memory Comparison ==="
echo "Traditional JSON Array API:"
echo "  • Load entire result set into memory"
echo "  • Memory usage: O(n) where n = number of rows"
echo "  • Response time: Must wait for complete query execution"
echo "  • Risk: Memory exhaustion with large datasets"
echo ""
echo "Streaming NDJSON API:"
echo "  • Process one row at a time"
echo "  • Memory usage: O(1) constant memory"
echo "  • Response time: First row appears immediately"
echo "  • Scalability: Handles any dataset size"
echo ""

echo "🚀 Ready for production deployment with enterprise-grade database streaming!"