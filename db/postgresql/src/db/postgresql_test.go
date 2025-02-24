package db

import (
	"testing"

	"kyle-postgresql/src/configs"
)

// go test -v -run TestInit
func TestInit(t *testing.T) {

	configs.SetDevEnv()

	// if err := InitConnection(); err != nil {
	// 	t.Error(err)
	// }
	if err := InitDB(); err != nil {
		t.Error(err)
	}

	t.Log("🔍 Checking database connection...")
	if err := CheckDBConnection(); err != nil {
		t.Error(err)
	}
}

// go test -v -run TestCreateDB
func TestCreateDB(t *testing.T) {

	configs.SetDevEnv()

	CreateDatabaseIfNotExists()
}

// go test -v -run TestCreateTables
func TestCreateTables(t *testing.T) {

	configs.SetDevEnv()

	CreateBlockchainSchemaAndTables()
}

// go test -v -run TestAutoMigrate
func TestAutoMigrate(t *testing.T) {

	configs.SetDevEnv()

	if err := AutoMigrate(); err != nil {
		t.Error(err)
	}
}
