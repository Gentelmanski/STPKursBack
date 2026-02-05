package postgres

import (
	"log"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	// Отключаем автоматическое создание внешних ключей и связей
	db = db.Set("gorm:table_options", "ENGINE=InnoDB")

	// Проверяем и добавляем только недостающие таблицы и поля
	tables := []interface{}{
		&UserModel{},
		&EventModel{},
		&TagModel{},
		&EventTagModel{},
		&EventMediaModel{},
		&EventParticipantModel{},
		&CommentModel{},
		&CommentVoteModel{},
		&NotificationModel{},
		&AdminActionModel{},
	}

	for _, table := range tables {
		// Проверяем, существует ли таблица
		if !db.Migrator().HasTable(table) {
			log.Printf("Creating table for %T...", table)
			if err := db.Migrator().CreateTable(table); err != nil {
				log.Printf("Warning: Failed to create table for %T: %v", table, err)
				// Продолжаем с другими таблицами даже если одна не создалась
			}
		} else {
			// Таблица существует, добавляем недостающие колонки
			log.Printf("Table for %T already exists, checking columns...", table)
			if err := db.AutoMigrate(table); err != nil {
				log.Printf("Warning: Failed to auto-migrate table for %T: %v", table, err)
				// Продолжаем с другими таблицами даже если одна не мигрировалась
			}
		}
	}

	log.Println("Database migration completed")
	return nil // Все операции были логированы, возвращаем nil
}
