package adminhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	botToken   string
	httpClient *http.Client
}

type RequestError struct {
	Op           string
	StatusCode   int
	Fallbackable bool
	Err          error
}

type actorTGIDContextKeyType struct{}

var actorTGIDContextKey actorTGIDContextKeyType

func (e *RequestError) Error() string {
	if e == nil {
		return ""
	}
	switch {
	case e.Err != nil && e.StatusCode > 0:
		return fmt.Sprintf("%s: status=%d: %v", e.Op, e.StatusCode, e.Err)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	case e.StatusCode > 0:
		return fmt.Sprintf("%s: status=%d", e.Op, e.StatusCode)
	default:
		return e.Op
	}
}

func (e *RequestError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewClient(baseURL string, botToken string, timeout time.Duration) (*Client, error) {
	trimmedBaseURL := strings.TrimSpace(baseURL)
	trimmedToken := strings.TrimSpace(botToken)
	if trimmedBaseURL == "" || trimmedToken == "" {
		return nil, &RequestError{
			Op:           "create admin http client",
			Fallbackable: false,
			Err:          errors.New("admin api url or admin bot token is empty"),
		}
	}

	parsed, err := url.Parse(trimmedBaseURL)
	if err != nil {
		return nil, &RequestError{
			Op:           "parse admin api url",
			Fallbackable: false,
			Err:          err,
		}
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, &RequestError{
			Op:           "validate admin api url",
			Fallbackable: false,
			Err:          fmt.Errorf("invalid admin api url: %s", trimmedBaseURL),
		}
	}

	if timeout <= 0 {
		timeout = 8 * time.Second
	}

	return &Client{
		baseURL:  strings.TrimRight(trimmedBaseURL, "/"),
		botToken: trimmedToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func IsFallbackable(err error) bool {
	var reqErr *RequestError
	if errors.As(err, &reqErr) {
		return reqErr.Fallbackable
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func WithActorTGID(ctx context.Context, actorTGID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, actorTGIDContextKey, actorTGID)
}

func ActorTGIDFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	value, ok := ctx.Value(actorTGIDContextKey).(int64)
	if !ok {
		return 0
	}
	return value
}

func (c *Client) DoJSON(ctx context.Context, method string, path string, requestBody interface{}, responseBody interface{}) error {
	if c == nil || c.httpClient == nil {
		return &RequestError{
			Op:           "do json request",
			Fallbackable: false,
			Err:          errors.New("admin http client is not initialized"),
		}
	}

	var payload []byte
	if requestBody != nil {
		rawPayload, err := json.Marshal(requestBody)
		if err != nil {
			return &RequestError{
				Op:           "marshal request body",
				Fallbackable: false,
				Err:          err,
			}
		}
		payload = rawPayload
	}

	actorTGID := ActorTGIDFromContext(ctx)
	if actorTGID == 0 {
		actorTGID = detectActorTGID(requestBody)
	}

	statusCode, responseBytes, err := c.do(ctx, method, path, payload, actorTGID)
	if err != nil {
		return err
	}
	if responseBody == nil {
		return nil
	}
	if len(responseBytes) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBytes, responseBody); err != nil {
		return &RequestError{
			Op:           "decode http response",
			StatusCode:   statusCode,
			Fallbackable: false,
			Err:          err,
		}
	}

	return nil
}

func (c *Client) do(ctx context.Context, method string, path string, body []byte, actorTGID int64) (int, []byte, error) {
	if c == nil || c.httpClient == nil {
		return 0, nil, &RequestError{
			Op:           "do request",
			Fallbackable: false,
			Err:          errors.New("admin http client is not initialized"),
		}
	}
	if strings.TrimSpace(method) == "" {
		method = http.MethodGet
	}

	fullURL := c.baseURL + ensureLeadingSlash(path)

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return 0, nil, &RequestError{
			Op:           "create http request",
			Fallbackable: false,
			Err:          err,
		}
	}
	req.Header.Set("X-Admin-Bot-Token", c.botToken)
	req.Header.Set("X-Actor-Tg-Id", strconv.FormatInt(actorTGID, 10))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, &RequestError{
			Op:           "execute http request",
			Fallbackable: isFallbackableNetworkError(err),
			Err:          err,
		}
	}
	defer resp.Body.Close()

	responseBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		return resp.StatusCode, nil, &RequestError{
			Op:           "read http response",
			StatusCode:   resp.StatusCode,
			Fallbackable: false,
			Err:          readErr,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMessage := strings.TrimSpace(string(responseBytes))
		if errMessage == "" {
			errMessage = http.StatusText(resp.StatusCode)
		}
		return resp.StatusCode, responseBytes, &RequestError{
			Op:           "unexpected http status",
			StatusCode:   resp.StatusCode,
			Fallbackable: isFallbackableStatus(resp.StatusCode),
			Err:          errors.New(errMessage),
		}
	}

	return resp.StatusCode, responseBytes, nil
}

func isFallbackableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout and transport network failures should fallback in dual mode.
		return true
	}
	return false
}

func ensureLeadingSlash(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

func isFallbackableStatus(statusCode int) bool {
	return statusCode >= 500
}

func detectActorTGID(requestBody interface{}) int64 {
	if requestBody == nil {
		return 0
	}

	switch body := requestBody.(type) {
	case map[string]interface{}:
		return findActorTGIDInMap(body)
	case map[string]string:
		return findActorTGIDInMapString(body)
	case map[string]int64:
		return findActorTGIDInMapInt64(body)
	}

	value := reflect.ValueOf(requestBody)
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return 0
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0
	}

	fieldCandidates := []string{
		"ActorTGID",
		"ActorTgID",
		"UpdatedByTGID",
		"UpdatedByTgID",
		"GrantedBy",
	}
	for _, fieldName := range fieldCandidates {
		field := value.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}
		if tgID, ok := valueToInt64(field); ok && tgID != 0 {
			return tgID
		}
	}

	return 0
}

func findActorTGIDInMap(value map[string]interface{}) int64 {
	keyCandidates := []string{"actor_tg_id", "updated_by_tg_id", "granted_by"}
	for _, key := range keyCandidates {
		raw, ok := value[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case int:
			return int64(typed)
		case int8:
			return int64(typed)
		case int16:
			return int64(typed)
		case int32:
			return int64(typed)
		case int64:
			return typed
		case uint:
			return int64(typed)
		case uint8:
			return int64(typed)
		case uint16:
			return int64(typed)
		case uint32:
			return int64(typed)
		case uint64:
			return int64(typed)
		case float64:
			return int64(typed)
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func findActorTGIDInMapString(value map[string]string) int64 {
	keyCandidates := []string{"actor_tg_id", "updated_by_tg_id", "granted_by"}
	for _, key := range keyCandidates {
		raw, ok := value[key]
		if !ok {
			continue
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func findActorTGIDInMapInt64(value map[string]int64) int64 {
	keyCandidates := []string{"actor_tg_id", "updated_by_tg_id", "granted_by"}
	for _, key := range keyCandidates {
		raw, ok := value[key]
		if ok && raw != 0 {
			return raw
		}
	}
	return 0
}

func valueToInt64(value reflect.Value) (int64, bool) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(value.Uint()), true
	default:
		return 0, false
	}
}
