package postgres

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB(dsn string) *gorm.DB {
	// Конфигурация GORM для работы с существующей базой данных
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		// Отключаем автоматическое создание внешних ключей
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Настраиваем пул соединений
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Настройки пула соединений
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(0)

	DB = db
	log.Println("Database connected successfully")
	return db
}

// SafeMigrate выполняет безопасную миграцию без потери данных
func SafeMigrate(db *gorm.DB) error {
	// Сначала проверяем наличие основных таблиц
	requiredTables := []string{"users", "events", "tags", "comments"}

	for _, table := range requiredTables {
		if !db.Migrator().HasTable(table) {
			log.Printf("Warning: Table %s does not exist. Creating...", table)
		}
	}

	// Выполняем безопасную миграцию
	return Migrate(db)
}
