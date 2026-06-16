package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/iterator"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

func main() {
	ctx := context.Background()

	projectID := shared.GetEnv("SPANNER_PROJECT_ID", "ims-project")
	instanceID := shared.GetEnv("SPANNER_INSTANCE_ID", "invisible-ms-instance")
	databaseID := shared.GetEnv("SPANNER_DATABASE_ID", "invisible-ms-db")

	// Create instance admin client
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create instance admin client: %v", err)
	}
	defer instanceAdmin.Close()

	// Check if instance exists
	instanceName := fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID)
	_, err = instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{
		Name: instanceName,
	})

	if err != nil {
		// Create instance
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
			log.Fatalf("Failed to create instance: %v", err)
		}
		_, err = op.Wait(ctx)
		if err != nil {
			log.Fatalf("Instance creation failed: %v", err)
		}
		log.Println("Instance created successfully")
	} else {
		log.Println("Instance already exists")
	}

	// Create database admin client
	dbAdmin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create database admin client: %v", err)
	}
	defer dbAdmin.Close()

	// Check if database exists
	dbName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)
	_, err = dbAdmin.GetDatabase(ctx, &adminpb.GetDatabaseRequest{
		Name: dbName,
	})

	if err != nil {
		// Scan migration files
		ddlStatements, dmlStatements, err := readMigrations("migrations")
		if err != nil {
			log.Fatalf("Failed to read migrations: %v", err)
		}

		log.Printf("Found %d DDL and %d DML statements across migration files",
			len(ddlStatements), len(dmlStatements))

		// Create database with DDL statements
		log.Printf("Creating database %s...", databaseID)
		op, err := dbAdmin.CreateDatabase(ctx, &adminpb.CreateDatabaseRequest{
			Parent:          instanceName,
			CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", databaseID),
			ExtraStatements: ddlStatements,
		})
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		_, err = op.Wait(ctx)
		if err != nil {
			log.Fatalf("Database creation failed: %v", err)
		}
		log.Println("Database created successfully with DDL")

		// Execute DML statements after database is created
		if len(dmlStatements) > 0 {
			log.Printf("Executing %d DML statements...", len(dmlStatements))
			spannerClient, err := spanner.NewClient(ctx, dbName)
			if err != nil {
				log.Fatalf("Failed to create Spanner client for DML: %v", err)
			}
			defer spannerClient.Close()

			for _, stmt := range dmlStatements {
				s := stmt
				log.Printf("  Executing DML: %s...", truncate(s, 60))
				_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					_, err := txn.Update(ctx, spanner.Statement{SQL: s})
					return err
				})
				if err != nil {
					log.Fatalf("Failed to execute DML '%s': %v", truncate(s, 60), err)
				}
			}
			log.Println("DML statements executed successfully!")
		}
	} else {
		log.Println("Database already exists")
	}

	// List databases to verify
	log.Println("\nDatabases in instance:")
	iter := dbAdmin.ListDatabases(ctx, &adminpb.ListDatabasesRequest{
		Parent: instanceName,
	})
	for {
		db, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to list databases: %v", err)
		}
		log.Printf("  - %s", db.Name)
	}
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

		statements := shared.SplitSQLStatements(string(content))

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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
