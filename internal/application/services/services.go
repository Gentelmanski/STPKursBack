package services

// Services объединяет все сервисы
type Services struct {
	EventMedia   *EventMediaService
	Auth         *AuthService
	Event        *EventService
	Comment      *CommentService
	Notification *NotificationService
	Admin        *AdminService
}
