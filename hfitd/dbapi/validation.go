/*
 * Hot Fixture Tool Daemon - Database API Validation
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package dbapi

import (
	"fmt"
	"strings"
)

/*
* validateFilterPart validates and sanitizes the filterpart parameter to prevent SQL injection.
* Returns the sanitized filter or an error if potentially dangerous content is detected.
 */
func ValidateFilterPart(filterpart string) (string, error) {
	if filterpart == "" {
		return "", nil
	}

	// Convert to lowercase for case-insensitive checking
	lower := strings.ToLower(strings.TrimSpace(filterpart))

	// List of dangerous SQL keywords/patterns that should be blocked
	dangerousPatterns := []string{
		"insert", "update", "delete", "drop", "create", "alter", "truncate",
		"grant", "revoke", "exec", "execute", "sp_", "xp_", "union",
		"information_schema", "sys.", "pg_", "mysql.", "--", "/*", "*/",
		";", "@@", "char(", "nchar(", "varchar(", "nvarchar(",
		"waitfor", "delay", "benchmark(", "sleep(", "load_file(",
		"into outfile", "into dumpfile", "script", "javascript", "vbscript",
	}

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return "", fmt.Errorf("potentially dangerous SQL content detected: %s", pattern)
		}
	}

	// Only allow safe SQL clauses (WHERE, ORDER BY, LIMIT, HAVING, GROUP BY)
	allowedPrefixes := []string{"where ", "order by ", "limit ", "having ", "group by "}
	hasValidPrefix := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(lower, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return "", fmt.Errorf("filterpart must start with WHERE, ORDER BY, LIMIT, HAVING, or GROUP BY")
	}

	// Additional validation: check for balanced quotes
	singleQuotes := strings.Count(filterpart, "'")
	doubleQuotes := strings.Count(filterpart, "\"")
	if singleQuotes%2 != 0 || doubleQuotes%2 != 0 {
		return "", fmt.Errorf("unbalanced quotes in filterpart")
	}

	return strings.TrimSpace(filterpart), nil
}
