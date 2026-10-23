//go:build integration

package repositories

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type integrationEntity struct {
	ID   uuid.UUID `gorm:"primaryKey;type:uuid"`
	Name string    `gorm:"not null"`
}

func TestRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}

	if err := db.AutoMigrate(&integrationEntity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(&integrationEntity{})
	})

	repo := &repository[integrationEntity, uuid.UUID]{db: db}
	entity := &integrationEntity{ID: uuid.New(), Name: "test"}

	if err := repo.Create(entity); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByPK(entity.ID)
	if err != nil {
		t.Fatalf("get by pk: %v", err)
	}
	if got.Name != entity.Name {
		t.Fatalf("name = %q, want %q", got.Name, entity.Name)
	}

	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("len(all) = %d, want 1", len(all))
	}
}
