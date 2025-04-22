package usecase

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	"maps"

	"github.com/Ifelsik/web-security-hw/internal/models"
	"github.com/Ifelsik/web-security-hw/internal/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

type ProxyWebAPIUseCase struct {
	logger                 *logrus.Entry
	repository             repository.Repository
	voulnerabilityDictFile string
}

func NewHistoryUseCase(repo repository.Repository, log *logrus.Logger, dictPath string) *ProxyWebAPIUseCase {
	return &ProxyWebAPIUseCase{
		logger:                 logrus.NewEntry(log),
		repository:             repo,
		voulnerabilityDictFile: dictPath,
	}
}

func (u *ProxyWebAPIUseCase) ParseRequest(r *http.Request) (*models.Request, error) {
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
	u.logger.Debug("schema HTTP", r.URL.Scheme)
	headers["Host"] = []string{r.Host}

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
		Body:       body,
	}
	u.logger.Debugf("Request parsed %+v", requestModel)
	return requestModel, nil
}

func (u *ProxyWebAPIUseCase) GetRequestsHistory(ctx context.Context) ([]*RequestsListHistory, error) {
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
			ID:     requestModel.ID,
			Method: requestModel.Method,
			Path:   requestModel.Path,
		}
	}

	u.logger.Debugf("Requests history formed successfully: %v", requestsHistory)
	return requestsHistory, nil
}

func (u *ProxyWebAPIUseCase) GetRequestByID(ctx context.Context, id uint64) (*http.Request, error) {
	u.logger.Debugf("Getting request by id: %d", id)

	requestModel, err := u.repository.GetRequestByID(ctx, id)
	if err != nil {
		u.logger.Errorf("Failed to get request: %v", err)
		return nil, err
	}

	request, err := u.BuildRequest(requestModel)
	if err != nil {
		u.logger.Errorf("Failed to build request: %v", err)
		return nil, err
	}

	u.logger.Debugf("Request got: %v", request)
	return request, nil
}

func (u *ProxyWebAPIUseCase) SaveRequestResponse(r *http.Request, resp *http.Response) error {
	u.logger.Debug("Saving request")

	requestModel, err := u.ParseRequest(r)
	if err != nil {
		return err
	}

	responseModel, err := u.ParseResponse(resp)
	if err != nil {
		return err
	}

	requestModel.Response = responseModel
	_, err = u.repository.CreateRequest(r.Context(), requestModel)
	if err != nil {
		u.logger.Errorf("Failed to save request: %v", err)
		return err
	}

	u.logger.Debug("Request saved")
	return nil
}

func (u *ProxyWebAPIUseCase) BuildRequest(requestParsed *models.Request) (*http.Request, error) {
	u.logger.Debug("Building request from model")

	var parsedGet map[string][]string
	if err := json.Unmarshal(requestParsed.GetParams, &parsedGet); err != nil {
		u.logger.Errorf("Failed to unmarshal get params: %v", err)
		return nil, err
	}

	u.logger.Debugf("Get params parsed: %v", parsedGet)
	query := url.Values{}
	for k, v := range parsedGet {
		query.Add(k, v[0])
	}
	u.logger.Debugf("Query built: %v", query)

	u.logger.Debug("Building body reader")
	var body io.Reader
	if len(requestParsed.Body) != 0 {
		u.logger.Debug("Body building")
		body = bytes.NewReader(requestParsed.Body)
	}

	URL := url.URL{
		Path:     requestParsed.Path,
		RawQuery: query.Encode(),
	}
	u.logger.Debug("URL built", URL)

	r, err := http.NewRequest(requestParsed.Method, URL.String(), body)
	if err != nil {
		u.logger.Errorf("building request: %v", err)
		return nil, err
	}

	// Adding headers and cookies
	var parsedHeaders map[string][]string
	if err = json.Unmarshal(requestParsed.Headers, &parsedHeaders); err != nil {
		u.logger.Errorf("Failed to unmarshal headers: %v", err)
		return nil, err
	}
	for name, headerList := range parsedHeaders {
		for _, header := range headerList {
			r.Header.Set(name, header)
		}
	}

	u.logger.Debugf("Headers added")

	// Adding host
	if hostHeader, ok := parsedHeaders["Host"]; ok && len(hostHeader) > 0 {
		r.Host = hostHeader[0]
	}

	// Adding cookies
	var parsedCookies map[string]string
	if err = json.Unmarshal(requestParsed.Cookies, &parsedCookies); err != nil {
		u.logger.Errorf("Failed to unmarshal cookies: %v", err)
		return nil, err
	}
	for name, cookie := range parsedCookies {
		r.AddCookie(&http.Cookie{Name: name, Value: cookie})
	}
	u.logger.Debugf("Cookies added")

	return r, nil
}

func (u *ProxyWebAPIUseCase) ParseResponse(resp *http.Response) (*models.Response, error) {
	u.logger.Debug("Parsing response")

	contentType := resp.Header.Get("Content-Type")
	if contentType == "image/jpeg" ||
		contentType == "image/png" ||
		contentType == "image/gif" ||
		contentType == "image/webp" ||
		contentType == "application/x-protobuf" {
		return nil, nil
	}

	headers := make(map[string][]string)
	maps.Copy(headers, resp.Header)
	u.logger.Debugf("Headers parsed: %v", headers)

	headersJson, err := json.Marshal(headers)
	if err != nil {
		u.logger.Errorf("Failed to marshal headers to json: %v", err)
		return nil, err
	}
	u.logger.Debug("Headers json parsed")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	model := &models.Response{
		StatusCode: resp.StatusCode,
		Message:    resp.Status,
		Headers:    datatypes.JSON(headersJson),
		Body:       body,
	}

	return model, nil
}

func (u *ProxyWebAPIUseCase) DirBusterIterable(r *http.Request) func() *http.Request {
	u.logger.Debug("Building dir buster request")

	dirBusterDict, err := os.Open(u.voulnerabilityDictFile)
	if err != nil {
		u.logger.Errorf("Failed to open dir buster dict: %v", err)
		return nil
	}
	defer dirBusterDict.Close()

	scanner := bufio.NewScanner(dirBusterDict)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	i := 0
	return func() *http.Request {
		if i < len(lines) {
			r.URL.Path = "/" + lines[i]
			i++
			return r
		}
		return nil
	}
}
