package services

import (
	"errors"
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
}

func NewAuthService(db *gorm.DB, jwtCfg *config.JWTConfig) *AuthService {
	return &AuthService{
		db:          db,
		ldapService: NewLDAPService(db),
		jwtConfig:   jwtCfg,
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

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
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

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, s.jwtConfig.ExpireHour)
	if err != nil {
		return nil, err
	}

	// Update last login time
	now := time.Now()
	user.LastLogin = &now
	s.db.Save(user)

	return &LoginResponse{
		Token:    token,
		User:     user,
		ExpireAt: time.Now().Add(time.Duration(s.jwtConfig.ExpireHour) * time.Hour),
	}, nil
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
