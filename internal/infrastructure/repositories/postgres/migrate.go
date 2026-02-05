package postgres

import (
	"log"

	"gorm.io/gorm"
)

// Migrate проверяет соединение
func Migrate(db *gorm.DB) {
	log.Println("Checking database structure...")

	// Проверяем наличие основных таблиц
	tables := []string{"users", "events", "tags", "event_tags", "comments", "notifications"}
	for _, table := range tables {
		if db.Migrator().HasTable(table) {
			log.Printf("Table %s exists ✓", table)
		} else {
			log.Printf("WARNING: Table %s does not exist", table)
		}
	}

	log.Println("Database structure check completed")
}
