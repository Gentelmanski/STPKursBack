package controllers

import (
	"net/http"
	"strconv"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/interfaces"

	"github.com/gin-gonic/gin"
)

type CommentController struct {
	commentService interfaces.CommentService
}

func NewCommentController(commentService interfaces.CommentService) *CommentController {
	return &CommentController{commentService: commentService}
}

func (c *CommentController) GetComments(ctx *gin.Context) {
	eventID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	comments, err := c.commentService.GetComments(ctx.Request.Context(), uint(eventID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, comments)
}

func (c *CommentController) CreateComment(ctx *gin.Context) {
	eventID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var req dto.CreateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("user_id")
	comment, err := c.commentService.CreateComment(ctx.Request.Context(), req, uint(eventID), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, comment)
}

func (c *CommentController) UpdateComment(ctx *gin.Context) {
	commentID, err := strconv.ParseUint(ctx.Param("commentId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req dto.UpdateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("user_id")
	comment, err := c.commentService.UpdateComment(ctx.Request.Context(), uint(commentID), req, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, comment)
}

func (c *CommentController) DeleteComment(ctx *gin.Context) {
	commentID, err := strconv.ParseUint(ctx.Param("commentId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	userID, _ := ctx.Get("user_id")
	err = c.commentService.DeleteComment(ctx.Request.Context(), uint(commentID), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

func (c *CommentController) VoteComment(ctx *gin.Context) {
	commentID, err := strconv.ParseUint(ctx.Param("commentId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req dto.VoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("user_id")
	comment, err := c.commentService.VoteComment(ctx.Request.Context(), uint(commentID), userID.(uint), req.VoteType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, comment)
}
