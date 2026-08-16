package kernel

import (
	"errors"
	"net/http"
	"net/http/httptest"
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
