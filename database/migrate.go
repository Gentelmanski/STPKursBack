package database

import (
	"auth-system/models"
	"log"
)

func Migrate() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.Tag{},
		&models.EventTag{},
		&models.EventMedia{},
		&models.EventParticipant{},
		&models.Comment{},
		&models.CommentVote{},
		&models.Notification{},
		&models.AdminAction{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Println("Database migrated successfully")
}
