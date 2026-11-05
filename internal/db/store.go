package db

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickarnoldi/funnel-service/internal/enum"
)

var (
	dbInstance *pgxpool.Pool
	initOnce   sync.Once
)

type Store struct {
	Ctx     context.Context
	DB      *pgxpool.Pool
	Queries *Queries
}

func (s *Store) Close() {
	if s.DB != nil {
		s.DB.Close()
	}
}

func NewStore(ctx context.Context, verbose ...bool) *Store {
	initOnce.Do(func() {
		conn := getConnectionParameters(ctx)
		if err := conn.TestConnection(ctx); err != nil {
			log.Fatalf("Unable to connect to database: %v", err)
		}

		var err error
		dbInstance, err = conn.CreateConnectionPool(ctx)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Connected to %s at %s:%d\n", conn.Database, conn.Host, conn.Port)
	})

	return &Store{
		Ctx:     ctx,
		DB:      dbInstance,
		Queries: New(dbInstance),
	}
}

// GetReferenceID creates/retrieves the local ID for a given imported reference type and ID.
func (s *Store) GetReferenceID(ctx context.Context, platform enum.PlatformEnum, referenceType string, referenceID int64) (int64, error) {
	localID, err := s.Queries.GetReferenceID(ctx, GetReferenceIDParams{
		PlatformID: int32(platform),
		Type:       referenceType,
		ExternalID: referenceID,
	})
	if err == pgx.ErrNoRows {
		localID, err = s.Queries.CreateReferenceID(ctx, CreateReferenceIDParams{
			PlatformID: int32(platform),
			Type:       referenceType,
			ExternalID: referenceID,
		})
		if err != nil {
			fmt.Printf("Error creating ReferenceID for %s with imported ID %d: %v\n", referenceType, referenceID, err)
			return 0, err
		}
	} else if err != nil {
		fmt.Printf("Error fetching ReferenceID for %s with imported ID %d: %v\n", referenceType, referenceID, err)
		return 0, err
	}

	return localID, nil
}
