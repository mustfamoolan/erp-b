package seeders

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminSeeder creates the initial SUPER_ADMIN user.
// Username: admin | Password: 12345678
// Idempotent — skips if admin user already exists.
type AdminSeeder struct{}

func (s *AdminSeeder) Run(db *gorm.DB) error {
	// Check if admin already exists
	var count int64
	if err := db.Table("users").Where("username = ?", "admin").Count(&count).Error; err != nil {
		return fmt.Errorf("AdminSeeder: failed to check existing admin: %w", err)
	}
	if count > 0 {
		fmt.Printf("%s Admin user already exists — skipping.\n", color.YellowString("⚠️"))
		return nil
	}

	// Get the SUPER_ADMIN role ID
	var superAdminRoleID uuid.UUID
	row := db.Table("roles").Select("id").Where("name = ?", "SUPER_ADMIN").Row()
	if err := row.Scan(&superAdminRoleID); err != nil {
		return fmt.Errorf("AdminSeeder: SUPER_ADMIN role not found (run migrations first): %w", err)
	}

	// Get the Administration scope ID
	var adminScopeID uuid.UUID
	scopeRow := db.Table("organization_scopes").Select("id").Where("code = ?", "ADM").Row()
	if err := scopeRow.Scan(&adminScopeID); err != nil {
		return fmt.Errorf("AdminSeeder: Administration scope not found (run migrations first): %w", err)
	}

	// Hash the password using bcrypt (cost 12)
	hash, err := bcrypt.GenerateFromPassword([]byte("12345678"), 12)
	if err != nil {
		return fmt.Errorf("AdminSeeder: failed to hash password: %w", err)
	}

	adminID := uuid.New()
	now := time.Now()

	// Insert the admin user
	if err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, full_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
	`, adminID, "admin", "admin@m3aml.local", string(hash), "System Administrator", now, now).Error; err != nil {
		return fmt.Errorf("AdminSeeder: failed to insert admin user: %w", err)
	}

	// Assign SUPER_ADMIN role (company-wide, no scope restriction)
	if err := db.Exec(`
		INSERT INTO user_roles (id, user_id, role_id, scope_id, created_at)
		VALUES (?, ?, ?, NULL, ?)
	`, uuid.New(), adminID, superAdminRoleID, now).Error; err != nil {
		return fmt.Errorf("AdminSeeder: failed to assign SUPER_ADMIN role: %w", err)
	}

	// Grant access to Administration scope
	if err := db.Exec(`
		INSERT INTO user_scope_access (id, user_id, scope_id, granted_at, granted_by)
		VALUES (?, ?, ?, ?, ?)
	`, uuid.New(), adminID, adminScopeID, now, adminID).Error; err != nil {
		return fmt.Errorf("AdminSeeder: failed to grant scope access: %w", err)
	}

	fmt.Printf("%s Admin user created: username=admin password=12345678 role=SUPER_ADMIN\n", color.GreenString("✅"))
	return nil
}
