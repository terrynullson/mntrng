package domain

import "time"

const (
	RoleSuperAdmin   = "super_admin"
	RoleCompanyAdmin = "company_admin"
	RoleViewer       = "viewer"
)

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

const (
	RegistrationStatusPending  = "pending"
	RegistrationStatusApproved = "approved"
	RegistrationStatusRejected = "rejected"
)

type AuthUser struct {
	ID        int64     `json:"id"`
	CompanyID *int64    `json:"company_id,omitempty"`
	Email     string    `json:"email"`
	Login     string    `json:"login"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRecord struct {
	AuthUser
	PasswordHash string
}

type RegistrationRequest struct {
	ID                int64      `json:"id"`
	CompanyID         int64      `json:"company_id"`
	Email             string     `json:"email"`
	Login             string     `json:"login"`
	RequestedRole     string     `json:"requested_role"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	ProcessedByUserID *int64     `json:"processed_by_user_id,omitempty"`
	DecisionReason    *string    `json:"decision_reason,omitempty"`
}

type AuthSession struct {
	ID               int64
	UserID           int64
	CompanyID        *int64
	AccessTokenHash  string
	RefreshTokenHash string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AuthSessionUser struct {
	Session AuthSession
	User    AuthUser
}

type AuthTokensResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int64    `json:"expires_in"`
	User         AuthUser `json:"user"`
}

type AuthContext struct {
	UserID    int64
	CompanyID *int64
	Role      string
	SessionID int64
}

type LoginRequest struct {
	LoginOrEmail string `json:"login_or_email"`
	Password     string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken *string `json:"refresh_token"`
}

type RegistrationRequestCreate struct {
	CompanyID     int64  `json:"company_id"`
	Email         string `json:"email"`
	Login         string `json:"login"`
	Password      string `json:"password"`
	RequestedRole string `json:"requested_role"`
}

type ApproveRegistrationRequest struct {
	CompanyID int64  `json:"company_id"`
	Role      string `json:"role"`
}

type RejectRegistrationRequest struct {
	Reason string `json:"reason"`
}

type ChangeUserRoleRequest struct {
	Role      string `json:"role"`
	CompanyID *int64 `json:"company_id"`
}

type ChangeUserStatusRequest struct {
	Status string `json:"status"`
}

type AdminUserListResponse struct {
	Items      []AuthUser `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type AdminUserListFilter struct {
	CompanyID *int64
	Role      *string
	Status    *string
	Limit     int
}
