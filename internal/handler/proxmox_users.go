package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateUserRequest is the JSON body for POST /access/users.
type CreateUserRequest struct {
	UserID    string `json:"userid" binding:"required,min=1,max=64"`
	Password  string `json:"password" binding:"required,min=5,max=64"`
	Email     string `json:"email"`
	Comment   string `json:"comment"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}

// UpdateUserRequest is the JSON body for PUT /access/users/{userid}.
type UpdateUserRequest struct {
	Email     string `json:"email,omitempty"`
	Comment   string `json:"comment,omitempty"`
	FirstName string `json:"firstname,omitempty"`
	LastName  string `json:"lastname,omitempty"`
	Enable    *int   `json:"enable,omitempty"`
	Password  string `json:"password,omitempty"`
}

// CreateAPITokenRequest is the JSON body for POST token.
type CreateAPITokenRequest struct {
	TokenID string `json:"tokenid" binding:"required,min=1,max=64"`
	Comment string `json:"comment"`
	Privsep int    `json:"privsep"`
	Expire  int    `json:"expire"`
}

// ListUsers returns all Proxmox users.
func (h *ProxmoxHandler) ListUsers(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	users, err := cli.ListUsers(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"users": users, "total": len(users)})
}

// CreateUser creates a new Proxmox user.
func (h *ProxmoxHandler) CreateUser(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	// Validate userid format: name@realm
	parts := strings.SplitN(req.UserID, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.CreateUser(c.Request.Context(), req.UserID, req.Password, req.Email, req.Comment, req.FirstName, req.LastName); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{"created": true, "userid": req.UserID})
}

// GetUser returns a single Proxmox user.
func (h *ProxmoxHandler) GetUser(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	userid := c.Param("userid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	user, err := cli.GetUser(c.Request.Context(), userid)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	// Proxmox GET /access/users/{userid} does NOT include userid in response.
	// Inject it so callers know which user this is.
	if user.UserID == "" {
		user.UserID = userid
	}
	// Strip secret value of tokens for safety
	if user.Tokens != nil {
		for k, t := range user.Tokens {
			t.Value = "[REDACTED]"
			user.Tokens[k] = t
		}
	}
	kernel.RespondOK(c, user)
}

// UpdateUser modifies a user.
func (h *ProxmoxHandler) UpdateUser(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	userid := c.Param("userid")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	fields := map[string]string{}
	if req.Email != "" {
		fields["email"] = req.Email
	}
	if req.Comment != "" {
		fields["comment"] = req.Comment
	}
	if req.FirstName != "" {
		fields["firstname"] = req.FirstName
	}
	if req.LastName != "" {
		fields["lastname"] = req.LastName
	}
	if req.Password != "" {
		fields["password"] = req.Password
	}
	if req.Enable != nil {
		if *req.Enable > 0 {
			fields["enable"] = "1"
		} else {
			fields["enable"] = "0"
		}
	}
	if len(fields) == 0 {
		kernel.RespondError(c, fmt.Errorf("no fields to update"))
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.UpdateUser(c.Request.Context(), userid, fields); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"updated": true, "userid": userid})
}

// DeleteUser removes a user.
func (h *ProxmoxHandler) DeleteUser(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	userid := c.Param("userid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteUser(c.Request.Context(), userid); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true, "userid": userid})
}

// ListAPITokens returns all tokens for a user.
func (h *ProxmoxHandler) ListAPITokens(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	userid := c.Param("userid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	tokens, err := cli.ListAPITokens(c.Request.Context(), userid)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	// Strip secret values
	for k, t := range tokens {
		t.Value = "[REDACTED]"
		tokens[k] = t
	}
	kernel.RespondOK(c, gin.H{"tokens": tokens, "total": len(tokens)})
}

// CreateAPIToken creates a new token for a user.
func (h *ProxmoxHandler) CreateAPIToken(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	userid := c.Param("userid")
	var req CreateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	tok, err := cli.AddAPIToken(c.Request.Context(), userid, req.TokenID, req.Comment, req.Privsep, req.Expire)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	// Return FULL token value here — caller must save it NOW (only time we show it).
	fullToken := fmt.Sprintf("%s@%s!%s", req.TokenID, extractRealm(userid), tok.Value)
	kernel.RespondCreated(c, gin.H{
		"created":     true,
		"userid":      userid,
		"tokenid":     req.TokenID,
		"full_token":  fullToken,
		"value":       tok.Value, // secret part, only returned here
	})
}

// DeleteAPIToken removes a token from a user.
func (h *ProxmoxHandler) DeleteAPIToken(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	userid := c.Param("userid")
	tokenid := c.Param("tokenid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteAPIToken(c.Request.Context(), userid, tokenid); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true, "userid": userid, "tokenid": tokenid})
}

// extractRealm extracts "name@realm" -> "realm".
func extractRealm(userid string) string {
	parts := strings.SplitN(userid, "@", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}