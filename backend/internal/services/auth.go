package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/utils"
	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	ldapService *LDAPService
	jwtConfig   *config.JWTConfig
	configSvc   *SystemConfigService
}

func NewAuthService(db *gorm.DB, jwtCfg *config.JWTConfig) *AuthService {
	return &AuthService{
		db:          db,
		ldapService: NewLDAPService(db),
		jwtConfig:   jwtCfg,
		configSvc:   NewSystemConfigService(db),
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	AuthType string `json:"auth_type"` // local, ldap
}

type LoginResponse struct {
	Token    string       `json:"token"`
	User     *models.User `json:"user"`
	ExpireAt time.Time    `json:"expire_at"`
}

type LoginResult struct {
	AccessToken     string
	AccessExpireAt  time.Time
	RefreshToken    string
	RefreshExpireAt time.Time
	User            *models.User
}

type RefreshResult struct {
	AccessToken     string
	AccessExpireAt  time.Time
	RefreshToken    string
	RefreshExpireAt time.Time
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req *LoginRequest, clientIP, userAgent string) (*LoginResult, error) {
	var user *models.User
	var err error

	// Default to local auth if not specified
	if req.AuthType == "" {
		req.AuthType = "local"
	}

	switch req.AuthType {
	case "local":
		user, err = s.localAuth(req.Username, req.Password)
	case "ldap":
		user, err = s.ldapAuth(req.Username, req.Password)
	default:
		return nil, errors.New("invalid auth type")
	}

	if err != nil {
		return nil, err
	}

	accessHours := s.getAccessTokenExpireHours()
	refreshHours := s.getRefreshTokenExpireHours()

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, accessHours)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshHash, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshExpireAt := time.Now().Add(time.Duration(refreshHours) * time.Hour)
	refreshRecord := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: refreshExpireAt,
	}
	if clientIP != "" {
		refreshRecord.CreatedByIP = clientIP
	}
	if userAgent != "" {
		refreshRecord.UserAgent = userAgent
	}
	if err := s.db.Create(&refreshRecord).Error; err != nil {
		return nil, err
	}

	// Update last login time
	now := time.Now()
	user.LastLogin = &now
	s.db.Save(user)

	return &LoginResult{
		AccessToken:     token,
		AccessExpireAt:  time.Now().Add(time.Duration(accessHours) * time.Hour),
		RefreshToken:    refreshToken,
		RefreshExpireAt: refreshExpireAt,
		User:            user,
	}, nil
}

func (s *AuthService) Refresh(refreshToken string, clientIP, userAgent string) (*RefreshResult, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token required")
	}

	hash := hashRefreshToken(refreshToken)

	var stored models.RefreshToken
	if err := s.db.Where("token_hash = ?", hash).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, err
	}

	if stored.RevokedAt != nil {
		return nil, errors.New("refresh token revoked")
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	var user models.User
	if err := s.db.First(&user, stored.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if !user.IsActive {
		return nil, errors.New("user is disabled")
	}

	accessHours := s.getAccessTokenExpireHours()
	refreshHours := s.getRefreshTokenExpireHours()

	newAccessToken, err := utils.GenerateToken(user.ID, user.Username, user.Role, accessHours)
	if err != nil {
		return nil, err
	}

	newRefreshToken, newRefreshHash, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newRefresh := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		ExpiresAt: now.Add(time.Duration(refreshHours) * time.Hour),
	}
	if clientIP != "" {
		newRefresh.CreatedByIP = clientIP
	}
	if userAgent != "" {
		newRefresh.UserAgent = userAgent
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newRefresh).Error; err != nil {
			return err
		}
		if err := tx.Model(&stored).Updates(map[string]interface{}{
			"revoked_at":           now,
			"replaced_by_token_id": newRefresh.ID,
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &RefreshResult{
		AccessToken:     newAccessToken,
		AccessExpireAt:  time.Now().Add(time.Duration(accessHours) * time.Hour),
		RefreshToken:    newRefreshToken,
		RefreshExpireAt: newRefresh.ExpiresAt,
	}, nil
}

func (s *AuthService) RevokeRefreshToken(refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	hash := hashRefreshToken(refreshToken)
	now := time.Now()
	if err := s.db.Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", now).Error; err != nil {
		return err
	}

	return nil
}

func (s *AuthService) getAccessTokenExpireHours() int {
	defaultHours := s.jwtConfig.ExpireHour
	value := s.configSvc.GetWithDefault("auth_access_token_expire_hours", strconv.Itoa(defaultHours))
	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return defaultHours
	}
	return hours
}

func (s *AuthService) getRefreshTokenExpireHours() int {
	value := s.configSvc.GetWithDefault("auth_refresh_token_expire_hours", "720")
	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return 720
	}
	return hours
}

func generateRefreshToken() (token string, tokenHash string, err error) {
	randomBytes := make([]byte, 32)
	if _, err = rand.Read(randomBytes); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(randomBytes)
	tokenHash = hashRefreshToken(token)
	return token, tokenHash, nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) localAuth(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ? AND auth_type = ?", username, "local").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, errors.New("user is disabled")
	}

	if !utils.CheckPassword(password, user.Password) {
		return nil, errors.New("invalid username or password")
	}

	return &user, nil
}

func (s *AuthService) ldapAuth(username, password string) (*models.User, error) {
	// Authenticate against LDAP
	ldapUser, err := s.ldapService.Authenticate(username, password)
	if err != nil {
		return nil, err
	}

	// Find or create user in database
	var user models.User
	err = s.db.Where("username = ? AND auth_type = ?", ldapUser.Username, "ldap").First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new LDAP user
		user = models.User{
			Username: ldapUser.Username,
			Email:    ldapUser.Email,
			Nickname: ldapUser.Nickname,
			Role:     "user",
			AuthType: "ldap",
			IsActive: true,
		}
		if err := s.db.Create(&user).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, errors.New("user is disabled")
	}

	// Update user info from LDAP
	user.Email = ldapUser.Email
	user.Nickname = ldapUser.Nickname
	s.db.Save(&user)

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateAdminIfNotExists creates default admin user if not exists
func (s *AuthService) CreateAdminIfNotExists() error {
	var count int64
	s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		hashedPassword, err := utils.HashPassword("admin")
		if err != nil {
			return err
		}

		admin := models.User{
			Username: "admin",
			Password: hashedPassword,
			Nickname: "Administrator",
			Role:     "admin",
			AuthType: "local",
			IsActive: true,
		}

		return s.db.Create(&admin).Error
	}

	return nil
}

func (s *AuthService) IsLDAPEnabled() bool {
	return s.ldapService.IsEnabled()
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (s *AuthService) ChangePassword(userID uint, req *ChangePasswordRequest) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}

	if user.AuthType != "local" {
		return errors.New("LDAP users cannot change password here")
	}

	if !utils.CheckPassword(req.OldPassword, user.Password) {
		return errors.New("incorrect old password")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.db.Save(&user).Error
}
