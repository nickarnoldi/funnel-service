package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "embed"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickarnoldi/funnel-service/internal/env"
)

const MaxConnections int32 = 10

//go:embed config/schema.sql
var createSchemaSql string

//go:embed config/migration.sql
var migrationSchemaSql string

type ConnectionParameters struct {
	Username string
	Password string
	Host     string
	Port     int
	Database string
}

func (conn *ConnectionParameters) CreateConnectionPool(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(conn.getDSN())
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}
	config.MaxConns = MaxConnections

	return pgxpool.NewWithConfig(ctx, config)
}

func (conn *ConnectionParameters) CreateSingleConnection(ctx context.Context, useMaintenanceMode ...bool) (*pgx.Conn, error) {
	connStr := conn.getDSN()
	if len(useMaintenanceMode) > 0 && useMaintenanceMode[0] {
		connStr = conn.getMaintenanceDSN()
	}

	config, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}
	config.ConnectTimeout = time.Second * 2

	return pgx.ConnectConfig(ctx, config)
}

func (conn *ConnectionParameters) TestConnection(ctx context.Context) error {
	var testConn *pgx.Conn
	var err error

	loopCount := 0

	for {
		if loopCount > 2 {
			return fmt.Errorf("unable to connect to database after multiple attempts")
		}
		loopCount++

		testConn, err = conn.CreateSingleConnection(ctx)

		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf("database \"%s\" does not exist", conn.Database)) {
				conn.createDatabase(ctx)
				conn.createSchema(ctx)

				continue
			} else {
				return fmt.Errorf("unable to connect to database: %w", err)
			}
		}

		break
	}

	defer testConn.Close(ctx)

	if err := testConn.Ping(ctx); err != nil {
		return err
	}

	schemaCheck := testConn.QueryRow(ctx, "SELECT id FROM abandoned_leads LIMIT 1")
	if err := schemaCheck.Scan(new(int)); err != nil && err != pgx.ErrNoRows {
		if err := conn.createSchema(ctx); err != nil {
			return fmt.Errorf("unable to create schema: %w", err)
		}
	}

	migrationCheck := testConn.QueryRow(ctx, `
		SELECT character_maximum_length FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name='abandoned_leads'
		AND column_name='country_code'
		AND character_maximum_length=2
		LIMIT 1`)
	if err := migrationCheck.Scan(new(int)); err == nil {
		if err := conn.migrateSchema(ctx); err != nil {
			return fmt.Errorf("unable to migrate schema: %w", err)
		}
	}

	return nil
}

func (conn *ConnectionParameters) getDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		conn.Username, conn.Password, conn.Host, conn.Port, conn.Database)
}

func (conn *ConnectionParameters) getMaintenanceDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		conn.Username, conn.Password, conn.Host, conn.Port, "postgres")
}

func (conn *ConnectionParameters) createDatabase(ctx context.Context) error {
	if conn.Database == "" {
		return fmt.Errorf("database name is empty, please provide a valid database name")
	}

	// Create a temporary connection to the default "postgres" database
	db, err := conn.CreateSingleConnection(ctx, true)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close(ctx)

	log.Printf("Database %s does not exist, attempting to create it...\n", conn.Database)

	if _, err := db.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{conn.Database}.Sanitize())); err != nil {
		log.Fatalf("Unable to create database: %v", err)
	}
	log.Printf("Database %s created successfully\n", conn.Database)

	return nil
}

func (conn *ConnectionParameters) createSchema(ctx context.Context) error {
	db, err := conn.CreateSingleConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}
	defer db.Close(ctx)

	log.Printf("Attempting to create schema for %s...\n", conn.Database)

	if _, err := db.Exec(ctx, createSchemaSql); err != nil {
		return fmt.Errorf("unable to create schema: %w", err)
	}

	log.Printf("Schema created successfully for %s\n", conn.Database)

	return nil
}

func (conn *ConnectionParameters) migrateSchema(ctx context.Context) error {
	db, err := conn.CreateSingleConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}
	defer db.Close(ctx)

	log.Printf("Attempting to migrate schema for %s...\n", conn.Database)

	if _, err := db.Exec(ctx, migrationSchemaSql); err != nil {
		return fmt.Errorf("unable to migrate schema: %w", err)
	}

	log.Printf("Schema migrated successfully for %s\n", conn.Database)

	return nil
}

func getConnectionParameters(ctx context.Context) ConnectionParameters {
	if env.Get("ENVIRONMENT", "local") == "production" {
		log.Printf("[notice] Using AWS Secrets Manager to retrieve connection parameters\n")

		region := "ca-central-1"
		config, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			log.Fatalf("unable to load AWS config: %v", err)
		}

		secretName := env.Get("DB_SECRET_ID", "prod/funnel-service")
		svc := secretsmanager.NewFromConfig(config)
		result, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
		if err != nil {
			log.Fatalf("unable to retrieve secret %s: %v", secretName, err)
		}

		var secretString string = *result.SecretString

		var conn ConnectionParameters
		err = json.Unmarshal([]byte(secretString), &conn)
		if err != nil {
			log.Fatalf("unable to unmarshal secret string: %v", err)
		}

		return conn
	}

	log.Printf("[notice] Using environment variables to retrieve connection parameters\n")
	dbuser, found := os.LookupEnv("POSTGRES_USER")
	if !found {
		log.Fatalf("unable to locate POSTGRES_USER env variable")
	}

	dbpasswd, found := os.LookupEnv("POSTGRES_PASSWORD")
	if !found {
		log.Fatalf("unable to locate POSTGRES_PASSWORD env variable")
	}

	dbname, found := os.LookupEnv("POSTGRES_DATABASE")
	if !found {
		log.Fatalf("unable to locate POSTGRES_DATABASE env variable")
	}

	dbhost := env.Get("POSTGRES_HOST", "localhost")
	dbport := env.Get("POSTGRES_PORT", 5432)

	return ConnectionParameters{
		Username: dbuser,
		Password: dbpasswd,
		Host:     dbhost,
		Port:     dbport,
		Database: dbname,
	}
}
