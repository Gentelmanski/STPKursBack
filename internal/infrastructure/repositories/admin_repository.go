package repositories

import (
	"context"
	"time"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	"gorm.io/gorm"
)

type AdminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) interfaces.AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) LogAction(ctx context.Context, action *entities.AdminAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

func (r *AdminRepository) GetActions(ctx context.Context) ([]entities.AdminAction, error) {
	var actions []entities.AdminAction
	err := r.db.WithContext(ctx).
		Order("performed_at DESC").
		Find(&actions).Error
	return actions, err
}

func (r *AdminRepository) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	var stats = make(map[string]interface{})

	// Общее количество пользователей
	var totalUsers int64
	r.db.WithContext(ctx).Model(&entities.User{}).Count(&totalUsers)
	stats["total_users"] = totalUsers

	// Регистрации сегодня
	today := time.Now().Truncate(24 * time.Hour)
	var todayRegistrations int64
	r.db.WithContext(ctx).Model(&entities.User{}).Where("created_at >= ?", today).Count(&todayRegistrations)
	stats["today_registrations"] = todayRegistrations

	// Пользователи онлайн (последние 15 минут)
	fifteenMinutesAgo := time.Now().Add(-15 * time.Minute)
	var onlineUsers int64
	r.db.WithContext(ctx).Model(&entities.User{}).Where("last_online >= ?", fifteenMinutesAgo).Count(&onlineUsers)
	stats["online_users"] = onlineUsers

	return stats, nil
}

func (r *AdminRepository) GetEventStats(ctx context.Context) (map[string]interface{}, error) {
	var stats = make(map[string]interface{})

	// Общее количество мероприятий
	var totalEvents int64
	r.db.WithContext(ctx).Model(&entities.Event{}).Count(&totalEvents)
	stats["total_events"] = totalEvents

	// Активные мероприятия
	var activeEvents int64
	r.db.WithContext(ctx).Model(&entities.Event{}).Where("is_active = ?", true).Count(&activeEvents)
	stats["active_events"] = activeEvents

	// Верифицированные мероприятия
	var verifiedEvents int64
	r.db.WithContext(ctx).Model(&entities.Event{}).Where("is_verified = ?", true).Count(&verifiedEvents)
	stats["verified_events"] = verifiedEvents

	// Общее количество комментариев
	var totalComments int64
	r.db.WithContext(ctx).Model(&entities.Comment{}).Where("is_deleted = ?", false).Count(&totalComments)
	stats["total_comments"] = totalComments

	return stats, nil
}

func (r *AdminRepository) GetTopEvents(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Используем Raw SQL для получения топ мероприятий
	err := r.db.WithContext(ctx).Raw(`
		SELECT e.id as event_id, e.title, COUNT(ep.user_id) as participants
		FROM events e
		LEFT JOIN event_participants ep ON e.id = ep.event_id AND ep.status = 'going'
		WHERE e.is_active = true
		GROUP BY e.id, e.title
		ORDER BY participants DESC
		LIMIT ?
	`, limit).Scan(&results).Error

	return results, err
}
