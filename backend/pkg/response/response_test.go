package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(handler gin.HandlerFunc) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	handler(c)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return resp
}

func TestSuccess(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		Success(c, map[string]string{"name": "test"})
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got %q", resp.Message)
	}
}

func TestCreated(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		Created(c, map[string]int{"id": 1})
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestBadRequest(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		BadRequest(c, "invalid input")
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 400 {
		t.Errorf("expected code 400, got %d", resp.Code)
	}
	if resp.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", resp.Message)
	}
}

func TestUnauthorized(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		Unauthorized(c, "token expired")
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 401 {
		t.Errorf("expected code 401, got %d", resp.Code)
	}
}

func TestForbidden(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		Forbidden(c, "admin required")
	})

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 403 {
		t.Errorf("expected code 403, got %d", resp.Code)
	}
}

func TestNotFound(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		NotFound(c, "resource not found")
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 404 {
		t.Errorf("expected code 404, got %d", resp.Code)
	}
}

func TestServerError(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		ServerError(c, "internal error")
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 500 {
		t.Errorf("expected code 500, got %d", resp.Code)
	}
}

func TestError_WithAppError(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		err := NewBadRequest("validation failed")
		Error(c, err)
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 400 {
		t.Errorf("expected code 400, got %d", resp.Code)
	}
	if resp.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", resp.Message)
	}
}

func TestError_WithGenericError(t *testing.T) {
	w := performRequest(func(c *gin.Context) {
		Error(c, errors.New("something went wrong"))
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Code != 500 {
		t.Errorf("expected code 500, got %d", resp.Code)
	}
}

func TestAppError_ErrorInterface(t *testing.T) {
	err := NewNotFound("user not found")
	if err.Error() != "user not found" {
		t.Errorf("expected 'user not found', got %q", err.Error())
	}
}
