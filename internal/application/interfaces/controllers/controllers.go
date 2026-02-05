package controllers

// Controllers объединяет все контроллеры для удобной передачи в роутер
type Controllers struct {
	Auth         *AuthController
	Event        *EventController
	Comment      *CommentController
	Notification *NotificationController
	Admin        *AdminController
}
