package auth

import "time"

const (
	SessionUserIDKey           = "auth.user_id"
	SessionMFAPendingUserIDKey = "auth.mfa_pending_user_id"
	SessionMFAPendingRemember  = "auth.mfa_pending_remember"
	SessionMFASetupSecretKey   = "auth.mfa_setup_secret"
	SessionMFASetupCodesKey    = "auth.mfa_setup_codes"
	TokenableTypeToko          = `App\Models\Toko`
)

type User struct {
	ID           int64
	Username     string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
	MFASecret    *string
	MFARecovery  *string
}

type PublicUser struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	IsActive   bool   `json:"isActive"`
	MFAEnabled bool   `json:"mfaEnabled"`
}

type Toko struct {
	ID          int64
	UserID      int64
	Name        string
	CallbackURL *string
	Token       *string
	IsActive    bool
}

type AccessToken struct {
	ID            int64
	TokenableType string
	TokenableID   int64
	Name          string
	ExpiresAt     *time.Time
}

type RegisterInput struct {
	Username string
	Name     string
	Email    string
	Password string
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:         u.ID,
		Username:   u.Username,
		Name:       u.Name,
		Email:      u.Email,
		Role:       u.Role,
		IsActive:   u.IsActive,
		MFAEnabled: u.MFAEnabled(),
	}
}

func (u User) MFAEnabled() bool {
	return u.MFASecret != nil && *u.MFASecret != ""
}
