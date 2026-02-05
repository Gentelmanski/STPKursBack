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
		// Отключаем автоматическую миграцию при открытии соединения
		SkipDefaultTransaction: true,
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

// SimpleMigrate просто проверяет подключение
func SafeMigratee(db *gorm.DB) error {
	// Просто проверяем, что можем выполнить запрос
	var result int
	if err := db.Raw("SELECT 1").Scan(&result).Error; err != nil {
		return err
	}
	log.Println("Database connection verified")
	return nil
}
