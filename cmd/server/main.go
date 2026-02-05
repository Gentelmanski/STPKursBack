package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"auth-system/internal/application/interfaces/api"
	"auth-system/internal/application/interfaces/controllers"
	"auth-system/internal/application/services"
	"auth-system/internal/config"
	"auth-system/internal/infrastructure/http"
	"auth-system/internal/infrastructure/repositories"
	"auth-system/internal/infrastructure/repositories/postgres"
	"auth-system/internal/pkg/utils"
)

func main() {
	// Загрузка конфигурации
	cfg := config.Load()

	// загрузка утилит
	jwtUtil := utils.NewJWTUtil(cfg.JWTSecret)
	passwordUtil := utils.NewPasswordUtil()

	// подключение к бд
	db := postgres.ConnectDB(cfg.DatabaseURL)

	// Простая проверка подключения (без изменения структуры)
	log.Println("Verifying database connection...")
	if err := postgres.SafeMigratee(db); err != nil {
		log.Printf("Warning: %v", err)
	}

	log.Println("Database ready")

	//загрузка репозиториев
	repos := repositories.NewRepositories(db)

	// загрузка сервисов
	svc := setupServices(repos, jwtUtil, passwordUtil)

	// загрузка контролеров
	ctrls := setupControllers(svc)

	// запуск сервера
	server := http.NewServer(cfg)

	// загрузка маршрутов
	router := setupRouter(server.GetEngine(), ctrls, jwtUtil)

	// старт сервера
	log.Printf("Server starting on %s", cfg.ServerPort)
	if err := router.Run(cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// func setupRepositories(db *gorm.DB) *postgresRepos.Repositories {
// 	return &postgresRepos.Repositories{
// 		User:         postgresRepos.NewUserRepository(db),
// 		Event:        postgresRepos.NewEventRepository(db),
// 		Comment:      postgresRepos.NewCommentRepository(db),
// 		Notification: postgresRepos.NewNotificationRepository(db),
// 		Admin:        postgresRepos.NewAdminRepository(db),
// 	}
// }

func setupServices(repos *repositories.Repositories, jwtUtil utils.JWTUtil, passwordUtil utils.PasswordUtil) *services.Services {
	return &services.Services{
		Auth:         services.NewAuthService(repos.User, jwtUtil, passwordUtil),
		Event:        services.NewEventService(repos.Event, repos.User, repos.Notification),
		Comment:      services.NewCommentService(repos.Comment, repos.User, repos.Event, repos.Notification),
		Notification: services.NewNotificationService(repos.Notification),
		Admin:        services.NewAdminService(repos.Admin, repos.Event, repos.User, repos.Comment, repos.Notification),
	}
}

func setupControllers(services *services.Services) *controllers.Controllers {
	return &controllers.Controllers{
		Auth:         controllers.NewAuthController(services.Auth),
		Event:        controllers.NewEventController(services.Event),
		Comment:      controllers.NewCommentController(services.Comment),
		Notification: controllers.NewNotificationController(services.Notification),
		Admin:        controllers.NewAdminController(services.Admin),
	}
}

func setupRouter(engine *gin.Engine, ctrls *controllers.Controllers, jwtUtil utils.JWTUtil) *gin.Engine {
	return api.SetupRoutes(engine, ctrls, jwtUtil)
}
