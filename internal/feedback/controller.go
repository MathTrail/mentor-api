package feedback

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Controller handles HTTP requests for feedback endpoints
type Controller struct {
	service Service
	logger  *zap.Logger
}

// NewController creates a new feedback controller
func NewController(service Service, logger *zap.Logger) *Controller {
	return &Controller{
		service: service,
		logger:  logger,
	}
}

// SubmitFeedback godoc
// @Summary Submit student feedback
// @Description Process student feedback about task difficulty.
// @Tags feedback
// @Accept json
// @Produce json
// @Param feedback body FeedbackRequest true "Feedback request"
// @Success 200 {object} StrategyUpdate
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/feedback [post]
func (c *Controller) SubmitFeedback(ctx *gin.Context) {
	var req FeedbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	update, err := c.service.ProcessFeedback(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to process feedback", zap.Error(err))
		ctx.JSON(500, ErrorResponse{Code: "INTERNAL_ERROR", Message: "failed to process feedback"})
		return
	}

	ctx.JSON(200, update)
}
