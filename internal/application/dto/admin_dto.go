package dto

type StatisticsResponse struct {
	TotalUsers         int64              `json:"total_users"`
	TotalEvents        int64              `json:"total_events"`
	ActiveEvents       int64              `json:"active_events"`
	VerifiedEvents     int64              `json:"verified_events"`
	TotalComments      int64              `json:"total_comments"`
	TodayRegistrations int64              `json:"today_registrations"`
	OnlineUsers        int64              `json:"online_users"`
	TopEvents          []TopEventResponse `json:"top_events"`
	PendingEvents      []EventResponse    `json:"pending_events"`
}

type TopEventResponse struct {
	EventID      uint   `json:"event_id"`
	Title        string `json:"title"`
	Participants int64  `json:"participants"`
}

type RejectEventRequest struct {
	Reason string `json:"reason"`
}

type AdminActionResponse struct {
	ID          uint   `json:"id"`
	AdminID     uint   `json:"admin_id"`
	ActionType  string `json:"action_type"`
	TargetID    uint   `json:"target_id"`
	TargetType  string `json:"target_type"`
	Reason      string `json:"reason"`
	PerformedAt string `json:"performed_at"`
}
