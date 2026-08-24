package rules

import "github.com/divmora/code-reviewer-ai-agent/pkg/model"

// DefaultRules contains built-in best practices for security, performance, and style.
var DefaultRules = []model.CustomRule{
	{
		ID:          "sec-sql-parameterization",
		Name:        "Enforce Parameterized SQL Queries",
		Severity:    model.SeverityHigh,
		Category:    model.CategorySecurity,
		Description: "All SQL queries MUST use parameterized placeholders (? or $1) rather than string concatenation or formatting (fmt.Sprintf, raw string interpolation) to prevent SQL injection.",
		Languages:   []string{"go", "python", "javascript", "typescript", "php", "java"},
	},
	{
		ID:          "perf-n-plus-one",
		Name:        "Prevent N+1 Queries in Loops",
		Severity:    model.SeverityMedium,
		Category:    model.CategoryPerformance,
		Description: "Avoid executing database queries or expensive API calls inside loops. Batch fetch with WHERE IN, JOIN, or eager loading.",
		Languages:   []string{"go", "python", "javascript", "typescript", "php", "java"},
	},
	{
		ID:          "sec-no-hardcoded-secrets",
		Name:        "Disallow Hardcoded Secrets and Credentials",
		Severity:    model.SeverityHigh,
		Category:    model.CategorySecurity,
		Description: "Never commit API keys, JWT secrets, private keys, database passwords, or auth tokens directly in code. Use environment variables or secret managers.",
	},
	{
		ID:          "perf-binary-assets-in-git",
		Name:        "Avoid Binary Blobs in Git",
		Severity:    model.SeverityMedium,
		Category:    model.CategoryPerformance,
		Description: "Do not commit binary assets (large images, PDFs, videos, zip files) directly to git repositories. Upload to CDN / S3 and reference by URL.",
	},
	{
		ID:          "php-laravel-transactions",
		Name:        "Strict DB Transactions and Error Reporting",
		Severity:    model.SeverityMedium,
		Category:    model.CategoryBestPractice,
		Description: "In PHP/Laravel, wrap DB operations in DB::beginTransaction() inside try, DB::commit() at end, and DB::rollBack() + report($e) in catch.",
		Languages:   []string{"php"},
	},
}
