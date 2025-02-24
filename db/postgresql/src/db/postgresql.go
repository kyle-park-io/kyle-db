package db

import (
	"database/sql"
	"fmt"
	"strings"

	"kyle-postgresql/src/logger"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitConnection() error {
	connStr := "user=kyle dbname=blockchain sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		logger.Log.Fatal("Cannot connect to database:", err)
	}

	logger.Log.Infoln("Connected to PostgreSQL!")
	return nil
}

func InitDB() error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		viper.GetString("database.host"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"),
		viper.GetInt("database.port"),
		viper.GetString("database.sslmode"),
		viper.GetString("database.timezone"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Log.Error(err)
		return err
	}

	logger.Log.Infoln("✅ Successfully connected to PostgreSQL!")
	return nil
}

func CheckDBConnection() error {
	sqlDB, err := DB.DB()
	if err != nil {
		logger.Log.Fatalf("Error getting database instance: %v", err)
	}

	// Ping check
	err = sqlDB.Ping()
	if err != nil {
		logger.Log.Infoln("❌ Database connection failed.")
		return err
	} else {
		logger.Log.Infoln("✅ Database connection is active.")
	}

	return nil
}

func CreateDatabaseIfNotExists() {
	dbname := viper.GetString("database.dbname")
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		viper.GetString("database.host"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		"postgres",
		viper.GetInt("database.port"),
		viper.GetString("database.sslmode"),
		viper.GetString("database.timezone"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Log.Fatal(err)
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbname).Scan(&exists)
	if err != nil {
		logger.Log.Fatal(err)
	}

	if !exists {
		_, err = db.Exec("CREATE DATABASE " + dbname)
		if err != nil {
			logger.Log.Fatal(err)
		}
		logger.Log.Infoln("Database created successfully")
	} else {
		logger.Log.Infoln("Database already exists")
	}
}

func CreateBlockchainSchemaAndTables() {
	dbname := viper.GetString("database.dbname")
	schemaName := viper.GetString("blockchain.schema")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		viper.GetString("database.host"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		dbname,
		viper.GetInt("database.port"),
		viper.GetString("database.sslmode"),
		viper.GetString("database.timezone"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	var schemaExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)", schemaName).Scan(&schemaExists)
	if err != nil {
		logger.Log.Fatal("Failed to check schema existence:", err)
	}

	// schema
	if !schemaExists {
		_, err = db.Exec("CREATE SCHEMA " + schemaName)
		if err != nil {
			logger.Log.Fatal("Failed to create schema:", err)
		}
		logger.Log.Infoln("Schema created successfully")
	} else {
		logger.Log.Infoln("Schema already exists")
	}

	// tables
	tables := viper.Get("blockchain.tables").([]interface{})
	for _, t := range tables {
		table := t.(map[string]interface{})
		tableName := table["name"].(string)

		var tableExists bool
		err = db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)",
			schemaName, tableName,
		).Scan(&tableExists)
		if err != nil {
			logger.Log.Fatal("Failed to check table existence:", err)
		}

		if !tableExists {
			columns := table["columns"].([]interface{})
			var columnDefs []string
			for _, c := range columns {
				column := c.(map[string]interface{})
				columnName := column["name"].(string)

				// "from", "to"
				if columnName == "from" || columnName == "to" {
					columnName = fmt.Sprintf("\"%s\"", columnName)
				}

				columnDef := fmt.Sprintf("%s %s", columnName, column["type"].(string))
				if column["primary_key"] == true {
					columnDef += " PRIMARY KEY"
				}
				if column["not_null"] == true {
					columnDef += " NOT NULL"
				}
				if column["unique"] == true {
					columnDef += " UNIQUE"
				}
				if column["default"] != nil {
					columnDef += fmt.Sprintf(" DEFAULT %v", column["default"])
				}
				columnDefs = append(columnDefs, columnDef)
			}

			createTableSQL := fmt.Sprintf("CREATE TABLE %s.%s (%s)", schemaName, tableName, strings.Join(columnDefs, ", "))
			fmt.Println(createTableSQL)
			_, err = db.Exec(createTableSQL)
			if err != nil {
				logger.Log.Infof("Failed to create table %s: %v\n", tableName, err)
			} else {
				logger.Log.Infof("Table %s created successfully\n", tableName)
			}
		} else {
			logger.Log.Infof("Table %s already exists\n", tableName)
		}
	}
}

func AutoMigrate() error {
	// init
	CreateDatabaseIfNotExists()
	InitDB()

	if err := DB.AutoMigrate(&Transaction{}, &TransactionReceipt{}, &Log{}); err != nil {
		logger.Log.Error(err)
		return err
	}

	return nil
}
