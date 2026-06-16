package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()

	projectID := os.Getenv("SPANNER_PROJECT_ID")
	instanceID := os.Getenv("SPANNER_INSTANCE_ID")
	databaseID := os.Getenv("SPANNER_DATABASE_ID")

	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create admin client: %v", err)
	}
	defer adminClient.Close()

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)

	// Read migration file
	migrationSQL, err := os.ReadFile("migrations/004_create_company_action_types.sql")
	if err != nil {
		log.Fatalf("Failed to read migration: %v", err)
	}

	// Split into DDL and DML statements
	ddlStatements, dmlStatements := splitStatements(string(migrationSQL))

	// Run DDL statements
	if len(ddlStatements) > 0 {
		fmt.Printf("Running %d DDL statements...\n", len(ddlStatements))
		op, err := adminClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
			Database:   dbPath,
			Statements: ddlStatements,
		})
		if err != nil {
			log.Fatalf("Failed to update database DDL: %v", err)
		}
		if err := op.Wait(ctx); err != nil {
			log.Fatalf("Failed to wait for DDL update: %v", err)
		}
		fmt.Println("DDL migration completed successfully!")
	}

	// Run DML statements
	if len(dmlStatements) > 0 {
		fmt.Printf("Running %d DML statements...\n", len(dmlStatements))
		client, err := spanner.NewClient(ctx, dbPath)
		if err != nil {
			log.Fatalf("Failed to create Spanner client: %v", err)
		}
		defer client.Close()

		for _, stmt := range dmlStatements {
			fmt.Printf("  Executing: %s...\n", truncate(stmt, 60))
			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, err := txn.Update(ctx, spanner.Statement{SQL: stmt})
				return err
			})
			if err != nil {
				log.Fatalf("Failed to execute DML: %v", err)
			}
		}
		fmt.Println("DML migration completed successfully!")
	}

	fmt.Println("All migrations completed successfully!")
}

func splitStatements(sql string) ([]string, []string) {
	var ddlStatements []string
	var dmlStatements []string
	var current string

	for _, line := range splitLines(sql) {
		line = trimSpace(line)
		if line == "" || startsWith(line, "--") {
			continue
		}
		current += line + " "
		if endsWith(line, ";") {
			stmt := trimSpace(current)
			// Remove trailing semicolon
			if endsWith(stmt, ";") {
				stmt = stmt[:len(stmt)-1]
			}
			stmt = trimSpace(stmt)
			
			// Classify as DDL or DML
			upper := strings.ToUpper(stmt)
			if strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "ALTER") || strings.HasPrefix(upper, "DROP") {
				ddlStatements = append(ddlStatements, stmt)
			} else if strings.HasPrefix(upper, "INSERT") || strings.HasPrefix(upper, "UPDATE") || strings.HasPrefix(upper, "DELETE") {
				dmlStatements = append(dmlStatements, stmt)
			}
			current = ""
		}
	}
	return ddlStatements, dmlStatements
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
