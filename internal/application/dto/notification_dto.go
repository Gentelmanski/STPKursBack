package dto

type NotificationResponse struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}

type MarkAsReadRequest struct {
	NotificationID uint `json:"notification_id"`
}

type MarkAllAsReadResponse struct {
	Message string `json:"message"`
}
