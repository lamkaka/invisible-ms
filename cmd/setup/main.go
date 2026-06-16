package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()

	projectID := getEnv("SPANNER_PROJECT_ID", "ims-project")
	instanceID := getEnv("SPANNER_INSTANCE_ID", "ims-instance")
	databaseID := getEnv("SPANNER_DATABASE_ID", "ims-db")

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
		// Read migration files
		migrations := []string{
			"migrations/001_create_companies.sql",
			"migrations/002_create_workers.sql",
			"migrations/003_create_activity_logs.sql",
		}

		var statements []string
		for _, file := range migrations {
			content, err := os.ReadFile(file)
			if err != nil {
				log.Fatalf("Failed to read migration file %s: %v", file, err)
			}
			// Split by semicolons and filter empty statements
			stmts := splitSQLStatements(string(content))
			statements = append(statements, stmts...)
		}

		// Create database with migrations
		log.Printf("Creating database %s...", databaseID)
		op, err := dbAdmin.CreateDatabase(ctx, &adminpb.CreateDatabaseRequest{
			Parent:          instanceName,
			CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", databaseID),
			ExtraStatements: statements,
		})
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		_, err = op.Wait(ctx)
		if err != nil {
			log.Fatalf("Database creation failed: %v", err)
		}
		log.Println("Database created successfully with migrations")
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitSQLStatements(content string) []string {
	var statements []string
	var current string
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(content); i++ {
		c := content[i]

		// Handle string literals
		if (c == '\'' || c == '"') && (i == 0 || content[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = c
			} else if c == stringChar {
				inString = false
			}
		}

		// Check for semicolon outside of strings
		if c == ';' && !inString {
			stmt := strings.TrimSpace(current)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current = ""
		} else {
			current += string(c)
		}
	}

	// Add any remaining statement
	stmt := strings.TrimSpace(current)
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}
