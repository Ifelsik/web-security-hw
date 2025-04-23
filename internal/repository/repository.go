package repository

import (
	"context"
	"fmt"

	"github.com/Ifelsik/web-security-hw/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ORMrepository struct {
	logger *logrus.Entry
	db     *gorm.DB
}

func NewORMrepository(db *gorm.DB, log *logrus.Logger) *ORMrepository {
	db.AutoMigrate(&models.Request{}, &models.Response{})

	return &ORMrepository{
		logger: logrus.NewEntry(log),
		db:     db,
	}
}

func (rep *ORMrepository) CreateRequest(ctx context.Context, request *models.Request) (uint, error) {
	// rep.logger.Debugf("Saving request. Got: %v", request)
	result := rep.db.Create(request)

	if result.Error != nil {
		rep.logger.Errorf("Failed to save request: %v", result.Error)
		return 0, result.Error
	}

	rep.logger.Debug("Request saved")
	return request.ID,nil
}

func (rep *ORMrepository) GetRequestByID(ctx context.Context, id uint64) (*models.Request, error) {
	rep.logger.Debugf("Getting request by id: %d", id)

	request := new(models.Request)
	result := rep.db.Preload("Response").Take(request, id)
	if result.Error != nil {
		rep.logger.Errorf("Failed to get request: %v", result.Error)
		return nil, result.Error
	}

	rep.logger.Debug("Request got")
	return request, nil
}

// Returns request fields:
// id, method, path
func (rep *ORMrepository) GetRequests(ctx context.Context, limit int) ([]*models.Request, error) {
	rep.logger.Debug("Getting requests count ", limit)

	var requests []*models.Request
	result := rep.db.Select("id", "method", "path").Limit(limit).Order("id desc").Find(&requests)
	if result.Error != nil {
		rep.logger.Errorf("Failed to get requests: %v", result.Error)
		return nil, result.Error
	}

	rep.logger.Debug("Requests got")
	return requests, nil
}

func (rep *ORMrepository) CreateResponse(ctx context.Context, response *models.Response) (uint, error) {
	rep.logger.Debugf("Saving response. Got: %v", response)
	result := rep.db.Create(response)

	if result.Error != nil {
		rep.logger.Errorf("Failed to save response: %v", result.Error)
		return 0, result.Error
	}

	rep.logger.Debug("Response saved")
	return response.ID , nil
}

func ConnectPGSQL(host, user, password, dbName, port string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		password,
		dbName,
		port,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
