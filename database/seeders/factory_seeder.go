package seeders

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type FactorySeeder struct{}

func (s *FactorySeeder) Run(db *gorm.DB) error {
	var count int64
	db.Table("organization_scopes").Where("code = ?", "FAC-A").Count(&count)
	if count > 0 {
		fmt.Printf("%s Factory A scope already exists — skipping.\n", color.YellowString("⚠️"))
		return nil
	}

	now := time.Now()
	factoryScopeID := uuid.New()

	// 1. Create Factory A Scope
	if err := db.Exec(`
		INSERT INTO organization_scopes (id, type, name, code, status, created_at, updated_at)
		VALUES (?, 'FACTORY', 'مصنع أ', 'FAC-A', 'ACTIVE', ?, ?)
	`, factoryScopeID, now, now).Error; err != nil {
		return err
	}

	// 2. Create Factory Accountant User
	hash, _ := bcrypt.GenerateFromPassword([]byte("12345678"), 12)
	userID := uuid.New()
	
	if err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, full_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
	`, userID, "factory_user", "factory@m3aml.local", string(hash), "محاسب مصنع أ", now, now).Error; err != nil {
		return err
	}

	// 3. Assign Role (FACTORY_ACCOUNTANT)
	var roleID uuid.UUID
	db.Table("roles").Select("id").Where("name = ?", "FACTORY_ACCOUNTANT").Row().Scan(&roleID)

	if err := db.Exec(`
		INSERT INTO user_roles (id, user_id, role_id, scope_id, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, uuid.New(), userID, roleID, factoryScopeID, now).Error; err != nil {
		return err
	}

	// 4. Grant Scope Access
	// We need the admin ID for granted_by
	var adminID uuid.UUID
	db.Table("users").Select("id").Where("username = ?", "admin").Row().Scan(&adminID)

	if err := db.Exec(`
		INSERT INTO user_scope_access (id, user_id, scope_id, granted_at, granted_by)
		VALUES (?, ?, ?, ?, ?)
	`, uuid.New(), userID, factoryScopeID, now, adminID).Error; err != nil {
		return err
	}

	// 5. Create Factory A Cashbox
	var accountID uuid.UUID
	db.Table("accounts").Select("id").Where("code = ?", "1102").Row().Scan(&accountID)

	if err := db.Exec(`
		INSERT INTO cashboxes (id, scope_id, name, account_id, currency, created_at, updated_at)
		VALUES (?, ?, 'صندوق مصنع أ', ?, 'SAR', ?, ?)
	`, uuid.New(), factoryScopeID, accountID, now, now).Error; err != nil {
		return err
	}

	// 6. Give Main Cashbox some initial funds via an opening balance journal entry
	var mainCashboxID uuid.UUID
	db.Table("cashboxes").Select("id").Where("name = ?", "الصندوق الرئيسي للإدارة").Row().Scan(&mainCashboxID)

	var adminScopeID uuid.UUID
	db.Table("organization_scopes").Select("id").Where("code = ?", "ADM").Row().Scan(&adminScopeID)

	var mainAccountID uuid.UUID
	db.Table("accounts").Select("id").Where("code = ?", "1101").Row().Scan(&mainAccountID)

	var equityAccountID uuid.UUID
	db.Table("accounts").Select("id").Where("code = ?", "2200").Row().Scan(&equityAccountID)

	var periodID uuid.UUID
	db.Table("accounting_periods").Select("id").Where("status = ?", "OPEN").Order("start_date desc").Limit(1).Row().Scan(&periodID)

	jeID := uuid.New()
	if err := db.Exec(`
		INSERT INTO journal_entries (id, period_id, entry_number, entry_date, description, scope_id, status, created_by, posted_by, posted_at, created_at)
		VALUES (?, ?, 'OP-001', ?, 'رصيد افتتاحي للصندوق الرئيسي', ?, 'POSTED', ?, ?, ?, ?)
	`, jeID, periodID, now, adminScopeID, adminID, adminID, now, now).Error; err != nil {
		return err
	}

	// Debit Main Cashbox
	if err := db.Exec(`
		INSERT INTO journal_lines (id, journal_entry_id, account_id, debit, credit, description, scope_id)
		VALUES (?, ?, ?, 500000, 0, 'رصيد افتتاحي', ?)
	`, uuid.New(), jeID, mainAccountID, adminScopeID).Error; err != nil {
		return err
	}

	// Credit Equity
	if err := db.Exec(`
		INSERT INTO journal_lines (id, journal_entry_id, account_id, debit, credit, description, scope_id)
		VALUES (?, ?, ?, 0, 500000, 'رصيد افتتاحي', ?)
	`, uuid.New(), jeID, equityAccountID, adminScopeID).Error; err != nil {
		return err
	}

	// Cash transaction
	if err := db.Exec(`
		INSERT INTO cash_transactions (id, cashbox_id, amount, currency, direction, source_type, source_id, journal_entry_id, description, performed_by, transaction_date, status, created_at)
		VALUES (?, ?, 500000, 'SAR', 'IN', 'OPENING_BALANCE', ?, ?, 'رصيد افتتاحي', ?, ?, 'COMPLETED', ?)
	`, uuid.New(), mainCashboxID, jeID, jeID, adminID, now, now).Error; err != nil {
		return err
	}

	fmt.Printf("%s Factory A and Initial Funds created successfully.\n", color.GreenString("✅"))
	return nil
}
