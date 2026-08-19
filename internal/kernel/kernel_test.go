package kernel

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRespondError_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	RespondError(c, ErrNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestRespondError_InternalWithDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	RespondError(c, errors.New("boom"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "boom") {
		t.Fatalf("internal error message leaked to client: %s", body)
	}
}

func TestRespondError_ConfigInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	RespondError(c, fmt.Errorf("auth_method=key but no ssh_key_id configured: %w", ErrConfigInvalid))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

func TestRespondError_Upstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	RespondError(c, fmt.Errorf("ssh dial 127.0.0.1:22: %w", ErrUpstream))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d", w.Code)
	}
}

func TestNewPage(t *testing.T) {
	p := NewPage()
	if p.Limit != 50 || p.Offset != 0 {
		t.Fatalf("unexpected default: %+v", p)
	}
}

func TestUserIsSuperAdmin(t *testing.T) {
	u := &User{Role: RoleSuperAdmin}
	if !u.IsSuperAdmin() {
		t.Fatal("want super admin")
	}
	u2 := &User{Role: RoleAdmin}
	if u2.IsSuperAdmin() {
		t.Fatal("don't want super admin")
	}
	var nilUser *User
	if nilUser.IsSuperAdmin() {
		t.Fatal("nil user can't be super admin")
	}
}
