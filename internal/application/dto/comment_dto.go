package dto

type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type VoteRequest struct {
	VoteType string `json:"vote_type" binding:"required,oneof=upvote downvote"`
}

type CommentResponse struct {
	ID        uint              `json:"id"`
	Content   string            `json:"content"`
	EventID   uint              `json:"event_id"`
	UserID    uint              `json:"user_id"`
	ParentID  *uint             `json:"parent_id"`
	Score     int               `json:"score"`
	IsDeleted bool              `json:"is_deleted"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
	User      UserShort         `json:"user"`
	Replies   []CommentResponse `json:"replies"`
}

type CommentVoteResponse struct {
	UserID    uint   `json:"user_id"`
	CommentID uint   `json:"comment_id"`
	VoteType  string `json:"vote_type"`
	VotedAt   string `json:"voted_at"`
}
