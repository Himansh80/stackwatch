// Tier 11 Phase 3 — Self-Service Signup (PL3).
// JSON row shapes + request/response bodies for the 3 PUBLIC
// signup endpoints:
//
//	POST /api/v1/platform/signup         — CreateSignup
//	POST /api/v1/platform/signup/verify  — VerifySignup
//	POST /api/v1/platform/signup/resend  — ResendSignup
//
// All three are PUBLIC (no JWT) — a freshly-typed-in email
// address has no StackWatch credentials yet. The handlers
// themselves live in handlers_platform_signup.go (kept under
// the 400-LOC cap by splitting types out).
//
// Why these shapes:
//   - signupReq carries the four fields a user is willing to
//     type in a marketing-style signup form. We do NOT include
//     any plan-tier field on Phase 3 — every signup lands on
//     `free` (mirroring auth_signup.go::Signup).
//
//   - signupResp returns the verification token ONLY in dev
//     mode (a future PL5 build will flip a config knob to hide
//     it once Resend/SMTP is wired). `next_step` is a stable
//     machine-readable hint the frontend uses to advance the
//     wizard state machine.
//
//   - verifyResp carries a fresh JWT so the user is logged-in
//     immediately after email verification — no second
//     round-trip to /auth/login.
package handler

// signupReq is the JSON body for POST /api/v1/platform/signup.
// All four fields are required and validated again server-side
// (email regex + ValidatePassword) regardless of what the
// browser claims — the API is public so a non-browser caller
// can bypass any HTML5 validation.
type signupReq struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=10"`
	FullName        string `json:"full_name" binding:"required,min=1,max=128"`
	OrganizationName string `json:"organization_name" binding:"required,min=1,max=128"`
}

// signupResp is the JSON returned by POST /signup on success.
//
// In dev / self-hosted mode `VerificationToken` is non-nil —
// the operator pastes it into /verify manually (no SMTP).
// Once a real email service is wired (PL5 follow-up) the
// token should be hidden from the response body (return nil)
// and only delivered via the outbound email.
type signupResp struct {
	ID                string  `json:"id"`
	Email             string  `json:"email"`
	Status            string  `json:"status"`
	VerificationToken *string `json:"verification_token,omitempty"`
	NextStep          string  `json:"next_step"`
}

// verifyReq is the JSON body for POST /api/v1/platform/signup/verify.
// The token is the plaintext the user received (in dev mode) by
// email or in the response of POST /signup.
type verifyReq struct {
	Token string `json:"token" binding:"required,min=16"`
}

// verifyResp is the JSON returned by POST /signup/verify on success.
// The JWT is the same shape /auth/login returns — the frontend
// stores it the same way and the user is now logged-in to the
// newly-created tenant. `tenant_id` + `user_id` are returned as
// strings so the frontend can persist them in localStorage without
// a UUID parser.
type verifyResp struct {
	Token    string `json:"token"`
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// resendReq is the JSON body for POST /api/v1/platform/signup/resend.
// Email is the only field — the handler looks up the most-recent
// pending signup and rotates its verification_token. We never
// expose whether the email exists in the system (no enumeration
// leak); both "no such signup" and "already verified" return 200
// with an empty token so the response is identical.
type resendReq struct {
	Email string `json:"email" binding:"required,email"`
}

// resendResp mirrors signupResp but with no id (we don't want to
// leak that the row exists). The token is returned in dev mode
// only — same dev-mode rule as signupResp.
type resendResp struct {
	Status            string  `json:"status"`
	VerificationToken *string `json:"verification_token,omitempty"`
	NextStep          string  `json:"next_step"`
}

// allowedSignupStatuses enumerates the legal values of
// platform_signups.status. Today we only ever write
// 'pending_verification' and 'verified'; 'rejected' is reserved
// for the Phase 5 abuse-moderation queue. Centralising the set
// in one place keeps the partial index (status, created_at DESC)
// honest — a typo'd status string would silently break the
// dashboard's "latest pending" query.
var allowedSignupStatuses = map[string]struct{}{
	"pending_verification": {},
	"verified":             {},
	"rejected":             {},
}

// devModeEmailDelivery is the constant written into the
// `verification_token` column by the resend handler. Phase 5
// will replace the dev-mode branch with a real SMTP / Resend
// call (see auth.go::EmailSender for the equivalent pattern
// on the /auth/forgot + /auth/magic-link surface).
//
// Keeping the value as a const means a future grep lands here,
// not at every call site — same discoverability rule as
// installTokenLifetime in handlers_platform_deploy_types.go.
const signupDevMode = true
