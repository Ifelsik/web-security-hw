package repository

import (
	"context"

	"github.com/Ifelsik/web-security-hw/internal/models"
)

type Repository interface {
	CreateRequest(ctx context.Context, request *models.Request) (uint, error)
	GetRequestByID(ctx context.Context, id uint64) (*models.Request, error)
	GetRequests(ctx context.Context, limit int) ([]*models.Request, error)
	CreateResponse(ctx context.Context, response *models.Response) (uint, error)
}
