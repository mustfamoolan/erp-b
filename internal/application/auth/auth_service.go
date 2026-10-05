package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	appaudit "m3aml-erp/internal/application/audit"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/repositories"
)

// ErrInvalidCredentials is returned when login credentials are wrong.
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrUserInactive is returned when a user account is not active.
var ErrUserInactive = errors.New("user account is inactive or suspended")

// ErrAccessDenied is returned when scope access is denied.
// Security is ALWAYS enforced by the backend — Roadmap Rule 8.
var ErrAccessDenied = errors.New("access denied: insufficient scope or permissions")

// Claims is the JWT payload.
type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

// LoginRequest holds credentials submitted by the client.
type LoginRequest struct {
	Username  string
	Password  string
	IPAddress string
	Device    string
}

type ScopeDTO struct {
	ID       uuid.UUID `json:"id"`
	Type     string    `json:"type"`
	Name     string    `json:"name"`
	Code     string    `json:"code"`
	AreaID   *string   `json:"area_id,omitempty"`
	AreaName string    `json:"area_name,omitempty"`
}

// LoginResponse holds the JWT token after successful auth.
type LoginResponse struct {
	Token     string
	ExpiresAt time.Time
	UserID    uuid.UUID
	Username  string
	FullName  string
	Roles     []string
	Scopes    []ScopeDTO
}

// AuthService handles authentication.
// Business rules live here (Go backend), NOT in the WPF — Rule 20.
type AuthService struct {
	userRepo  repositories.UserRepository
	scopeRepo repositories.ScopeRepository
	roleRepo  repositories.RoleRepository
	areaRepo  repositories.AreaRepository
	jwtSecret []byte
	tokenTTL  time.Duration
	auditSvc  *appaudit.AuditService
}

func NewAuthService(userRepo repositories.UserRepository, scopeRepo repositories.ScopeRepository, roleRepo repositories.RoleRepository, areaRepo repositories.AreaRepository, jwtSecret string, auditSvc *appaudit.AuditService) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		scopeRepo: scopeRepo,
		roleRepo:  roleRepo,
		areaRepo:  areaRepo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  12 * time.Hour,
		auditSvc:  auditSvc,
	}
}

// Login validates credentials and issues a JWT.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("auth: lookup failed: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check account status BEFORE any password check — Rule 8
	if user.Status != identity.UserStatusActive {
		return nil, ErrUserInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Issue JWT
	expiresAt := time.Now().Add(s.tokenTTL)
	claims := &Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("auth: token signing failed: %w", err)
	}

	// §34: Audit LOGIN
	if s.auditSvc != nil {
		uid := user.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     user.ID,
			Action:     domainaudit.AuditLogin,
			EntityType: "auth",
			EntityID:   &uid,
			IPAddress:  req.IPAddress,
			Device:     req.Device,
		})
	}

	// Load preloaded roles and scopes via FindByID
	fullUser, _ := s.userRepo.FindByID(ctx, user.ID)
	
	var roles []string
	var scopes []ScopeDTO
	if fullUser != nil {
		seenRoles := make(map[string]bool)
		for _, ur := range fullUser.Roles {
			role, err := s.roleRepo.FindByID(ctx, ur.RoleID)
			if err == nil && role != nil {
				if !seenRoles[string(role.Name)] {
					roles = append(roles, string(role.Name))
					seenRoles[string(role.Name)] = true
				}
			}
		}
	}

	// Get scope access explicit IDs
	scopeIDs, _ := s.userRepo.GetUserScopeAccess(ctx, user.ID)
	for _, sid := range scopeIDs {
		scope, _ := s.scopeRepo.FindByID(ctx, sid)
		if scope != nil {
			dto := ScopeDTO{
				ID:   scope.ID,
				Type: string(scope.Type),
				Name: scope.Name,
				Code: scope.Code,
			}
			
			if scope.AreaID != nil {
				area, _ := s.areaRepo.FindByID(ctx, *scope.AreaID)
				if area != nil {
					areaIDStr := area.ID.String()
					dto.AreaID = &areaIDStr
					dto.AreaName = area.Name
				}
			}
			
			scopes = append(scopes, dto)
		}
	}
	
	return &LoginResponse{
		Token:     signed,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		Username:  user.Username,
		FullName:  user.FullName,
		Roles:     roles,
		Scopes:    scopes,
	}, nil
}

// ParseToken validates and parses a JWT string.
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// HashPassword hashes a plain-text password using bcrypt.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(hash), err
}
