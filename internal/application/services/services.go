package services

// Services объединяет все сервисы
type Services struct {
	Auth         *AuthService
	Event        *EventService
	Comment      *CommentService
	Notification *NotificationService
	Admin        *AdminService
}
