package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Ifelsik/web-security-hw/internal/models"
	"github.com/Ifelsik/web-security-hw/internal/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

type HistoryUseCase struct {
	logger *logrus.Entry
	repository repository.Repository
}

func NewHistoryUseCase(repo repository.Repository, log *logrus.Logger) *HistoryUseCase {
	return &HistoryUseCase{
		logger: logrus.NewEntry(log),
		repository: repo,
	}
}

func (u *HistoryUseCase) ParseRequest(r *http.Request) (*models.Request, error) {
	u.logger.Debugf("Parsing request. Got: %v", r)

	getParams := map[string][]string(r.URL.Query())
	getParamsJson, err := json.Marshal(getParams)
	if err != nil {
		return nil, err
	}

	headers := map[string][]string{}
	for k, v := range r.Header {
		if k != "Cookie" {
			headers[k] = v
		}
	}
	headersJson, err := json.Marshal(headers)
	if err != nil {
		return nil, err
	}

	cookies := map[string]string{}
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	cookiesJson, err := json.Marshal(cookies)
	if err != nil {
		return nil, err
	}

	var postParams map[string][]string
	contentType := r.Header.Get("Content-Type")
	u.logger.Debugf("Content-Type: %v", contentType)
	if contentType == "application/x-www-form-urlencoded" {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}
		postParams = map[string][]string(r.PostForm)
	} else {
		u.logger.Debug("Unsupported Content-Type")
	}
	postParamsJson, err := json.Marshal(postParams)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	requestModel := &models.Request{
		Method:     r.Method,
		Path:       r.URL.Path,
		GetParams:  datatypes.JSON(getParamsJson),
		PostParams: datatypes.JSON(postParamsJson),
		Headers:    datatypes.JSON(headersJson),
		Cookies:    datatypes.JSON(cookiesJson),
		Body:       string(body),
	}
	u.logger.Debugf("Request parsed %+v", requestModel)
	return requestModel, nil
}

func (u *HistoryUseCase) GetRequestsHistory(ctx context.Context) ([]*RequestsListHistory, error) {
	u.logger.Debugln("Getting requests history")

	const limit = 25
	requestModels, err := u.repository.GetRequests(ctx, limit)
	if err != nil {
		u.logger.Errorf("Failed to get requests history: %v", err)
		return nil, err
	}

	requestsHistory := make([]*RequestsListHistory, len(requestModels))
	for i, requestModel := range requestModels {
		requestsHistory[i] = &RequestsListHistory{
			ID:         requestModel.ID,
			Method:     requestModel.Method,
			Path:       requestModel.Path,
		}
	}

	u.logger.Debugf("Requests history formed successfully: %v", requestsHistory)
	return requestsHistory, nil
}

func (u *HistoryUseCase) GetRequestByID(ctx context.Context, id uint64) (*RequestHistory, error) {
	u.logger.Debugf("Getting request by id: %d", id)

	requestModel, err := u.repository.GetRequestByID(ctx, id)
	if err != nil {
		u.logger.Errorf("Failed to get request: %v", err)
		return nil, err
	}


	var getParams map[string]interface{}
	// TODO: Упаковать в отдельную функцию
	if err := json.Unmarshal(requestModel.GetParams, &getParams); err != nil {
		u.logger.Errorf("Failed to unmarshal get params: %v", err)
		return nil, err
	}

	var postParams map[string]interface{}
	// TODO: Упаковать в отдельную функцию
	if err := json.Unmarshal(requestModel.PostParams, &postParams); err != nil {
		u.logger.Errorf("Failed to unmarshal post params: %v", err)
		return nil, err
	}

	var headers map[string]interface{}
	// TODO: Упаковать в отдельную функцию
	if err := json.Unmarshal(requestModel.Headers, &headers); err != nil {
		u.logger.Errorf("Failed to unmarshal headers: %v", err)
		return nil, err
	}

	var cookies map[string]interface{}
	// TODO: Упаковать в отдельную функцию
	if err := json.Unmarshal(requestModel.Cookies, &cookies); err != nil {
		u.logger.Errorf("Failed to unmarshal cookies: %v", err)
		return nil, err
	}

	result := &RequestHistory{
		ID:         requestModel.ID,
		Method:     requestModel.Method,
		Path:       requestModel.Path,
		GetParams:  getParams,
		PostParams: postParams,
		Headers:    headers,
		Cookies:    cookies,
		Body:       requestModel.Body,
	}
	u.logger.Debugf("Request got: %v", result)
	return result, nil
}

func (u *HistoryUseCase) SaveRequest(r *http.Request) error {
	u.logger.Debug("Saving request")

	model, err := u.ParseRequest(r)
	if err != nil {
		return err
	}

	err = u.repository.CreateRequest(r.Context(), model)
	if err != nil {
		u.logger.Errorf("Failed to save request: %v", err)
		return err
	}

	u.logger.Debug("Request saved")
	return nil
}
