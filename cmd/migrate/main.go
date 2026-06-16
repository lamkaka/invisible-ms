package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()

	projectID := os.Getenv("SPANNER_PROJECT_ID")
	instanceID := os.Getenv("SPANNER_INSTANCE_ID")
	databaseID := os.Getenv("SPANNER_DATABASE_ID")
	emulatorHost := os.Getenv("SPANNER_EMULATOR_HOST")

	if projectID == "" || instanceID == "" || databaseID == "" {
		log.Fatal("SPANNER_PROJECT_ID, SPANNER_INSTANCE_ID, and SPANNER_DATABASE_ID must be set")
	}

	instanceName := fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID)
	dbName := fmt.Sprintf("%s/databases/%s", instanceName, databaseID)

	// Step 1: Wait for Spanner emulator to be ready
	log.Printf("Waiting for Spanner emulator at %s...", emulatorHost)
	if err := waitForEmulator(ctx, emulatorHost, 30*time.Second); err != nil {
		log.Fatalf("Emulator not ready: %v", err)
	}
	log.Println("Spanner emulator is ready!")

	// Step 2: Create instance if it doesn't exist
	log.Println("Setting up Spanner instance...")
	if err := ensureInstance(ctx, projectID, instanceID, instanceName); err != nil {
		log.Fatalf("Failed to ensure instance: %v", err)
	}

	// Step 3: Read and parse migration files
	log.Println("Reading migration files...")
	ddlStatements, dmlStatements, err := readMigrations("migrations")
	if err != nil {
		log.Fatalf("Failed to read migrations: %v", err)
	}

	log.Printf("Found %d DDL statements and %d DML statements across all migration files",
		len(ddlStatements), len(dmlStatements))

	// Step 4: Create or update database
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create database admin client: %v", err)
	}
	defer adminClient.Close()

	dbExists, err := databaseExists(ctx, adminClient, dbName)
	if err != nil {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if dbExists {
		log.Printf("Database %s already exists", databaseID)
		if len(ddlStatements) > 0 {
			log.Printf("Applying %d DDL statements via UpdateDatabaseDdl...", len(ddlStatements))
			if err := applyDDL(ctx, adminClient, dbName, ddlStatements); err != nil {
				log.Fatalf("Failed to apply DDL: %v", err)
			}
			log.Println("DDL statements applied successfully!")
		} else {
			log.Println("No DDL statements to apply")
		}
	} else {
		log.Printf("Creating database %s...", databaseID)
		if err := createDatabase(ctx, adminClient, instanceName, databaseID, ddlStatements); err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Println("Database created successfully with DDL!")
	}

	// Step 5: Execute DML statements
	if len(dmlStatements) > 0 {
		log.Printf("Executing %d DML statements...", len(dmlStatements))
		if err := executeDML(ctx, dbName, dmlStatements); err != nil {
			log.Fatalf("Failed to execute DML: %v", err)
		}
		log.Println("DML statements executed successfully!")
	} else {
		log.Println("No DML statements to execute")
	}

	log.Println("All migrations completed successfully!")
}

// waitForEmulator polls the emulator host with a TCP dial until it responds or timeout.
func waitForEmulator(ctx context.Context, host string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := net.DialTimeout("tcp", host, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		log.Printf("  Emulator not ready yet, retrying in 2s... (%v)", err)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out waiting for emulator at %s after %v", host, timeout)
}

// ensureInstance creates the Spanner instance if it doesn't already exist.
func ensureInstance(ctx context.Context, projectID, instanceID, instanceName string) error {
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create instance admin client: %w", err)
	}
	defer instanceAdmin.Close()

	// Check if instance already exists
	_, err = instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{
		Name: instanceName,
	})
	if err == nil {
		log.Printf("Instance %s already exists, skipping creation", instanceID)
		return nil
	}

	// Create the instance
	log.Printf("Creating instance %s...", instanceID)
	op, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", projectID),
		InstanceId: instanceID,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", projectID),
			DisplayName: instanceID,
			NodeCount:   1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("instance creation failed: %w", err)
	}

	log.Printf("Instance %s created successfully!", instanceID)
	return nil
}

// databaseExists checks whether the database already exists.
func databaseExists(ctx context.Context, client *database.DatabaseAdminClient, dbName string) (bool, error) {
	_, err := client.GetDatabase(ctx, &adminpb.GetDatabaseRequest{
		Name: dbName,
	})
	if err == nil {
		return true, nil
	}
	// If the error indicates not found, return false
	if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "not found") {
		return false, nil
	}
	return false, err
}

// createDatabase creates a new database with the given DDL statements.
func createDatabase(ctx context.Context, client *database.DatabaseAdminClient, instanceName, databaseID string, ddlStatements []string) error {
	createStmt := fmt.Sprintf("CREATE DATABASE `%s`", databaseID)
	req := &adminpb.CreateDatabaseRequest{
		Parent:          instanceName,
		CreateStatement: createStmt,
		ExtraStatements: ddlStatements,
	}

	op, err := client.CreateDatabase(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("database creation failed: %w", err)
	}

	return nil
}

// applyDDL applies DDL statements to an existing database via UpdateDatabaseDdl.
// Statements are applied individually so that if a table or index already exists,
// only that statement fails and the rest can proceed.
func applyDDL(ctx context.Context, client *database.DatabaseAdminClient, dbName string, ddlStatements []string) error {
	for _, stmt := range ddlStatements {
		log.Printf("  Applying DDL: %s...", truncate(stmt, 80))
		op, err := client.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
			Database:   dbName,
			Statements: []string{stmt},
		})
		if err != nil {
			// If object already exists, log warning and continue
			errStr := err.Error()
			if strings.Contains(errStr, "AlreadyExists") || strings.Contains(errStr, "already exists") || strings.Contains(errStr, "Duplicate") {
				log.Printf("  Warning: DDL statement skipped (already exists): %v", err)
				continue
			}
			return fmt.Errorf("failed to apply DDL '%s': %w", truncate(stmt, 80), err)
		}
		if err := op.Wait(ctx); err != nil {
			// Same check on Wait errors
			errStr := err.Error()
			if strings.Contains(errStr, "AlreadyExists") || strings.Contains(errStr, "already exists") || strings.Contains(errStr, "Duplicate") {
				log.Printf("  Warning: DDL statement skipped (already exists): %v", err)
				continue
			}
			return fmt.Errorf("DDL operation failed for '%s': %w", truncate(stmt, 80), err)
		}
		log.Printf("  DDL applied successfully")
	}
	return nil
}

// executeDML runs DML statements via a Spanner client using ReadWriteTransaction.
func executeDML(ctx context.Context, dbName string, statements []string) error {
	client, err := spanner.NewClient(ctx, dbName)
	if err != nil {
		return fmt.Errorf("failed to create Spanner client: %w", err)
	}
	defer client.Close()

	for _, stmt := range statements {
		s := stmt // local copy
		log.Printf("  Executing DML: %s...", truncate(s, 60))
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, err := txn.Update(ctx, spanner.Statement{SQL: s})
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to execute DML '%s': %w", truncate(s, 60), err)
		}
	}

	return nil
}

// readMigrations reads all .sql files from the given directory sorted by name,
// parses each into DDL and DML statements.
func readMigrations(dir string) ([]string, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read migrations directory %s: %w", dir, err)
	}

	// Filter and sort .sql files
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	var allDDL []string
	var allDML []string

	for _, fileName := range files {
		filePath := filepath.Join(dir, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		statements := splitSQLStatements(string(content))

		var fileDDL, fileDML int
		for _, stmt := range statements {
			upper := strings.ToUpper(stmt)
			if strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "ALTER") || strings.HasPrefix(upper, "DROP") {
				allDDL = append(allDDL, stmt)
				fileDDL++
			} else if strings.HasPrefix(upper, "INSERT") || strings.HasPrefix(upper, "UPDATE") || strings.HasPrefix(upper, "DELETE") {
				allDML = append(allDML, stmt)
				fileDML++
			}
		}

		log.Printf("  %s: %d DDL, %d DML statements", fileName, fileDDL, fileDML)
	}

	return allDDL, allDML, nil
}

// splitSQLStatements splits SQL content by semicolons, respecting string literals.
func splitSQLStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	var stringChar byte

	for i := 0; i < len(content); i++ {
		c := content[i]

		// Handle string literals (single or double quotes)
		if (c == '\'' || c == '"') && (i == 0 || content[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = c
			} else if c == stringChar {
				inString = false
			}
		}

		// Skip semicolons inside string literals
		if c == ';' && !inString {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}

	// Catch any remaining statement without trailing semicolon
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
