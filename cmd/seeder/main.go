package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"github.com/google/uuid"
	"m3aml-erp/bootstrap"
)

func main() {
	bootstrap.InitializeConfig()
	db := bootstrap.InitializeDatabase()

	fmt.Println("🚀 Starting Factory Dummy Data Seeder...")

	// 1. Find the first active Factory
	var factoryID string
	err := db.Raw("SELECT id FROM organization_scopes WHERE type = 'FACTORY' LIMIT 1").Scan(&factoryID).Error
	if err != nil || factoryID == "" {
		log.Fatalf("❌ No factory found! Please add a factory from the frontend first.")
	}
	fmt.Println("🏭 Found Factory ID:", factoryID)

	// 2. Find a user to act as the requester
	var userID string
	db.Raw("SELECT id FROM users LIMIT 1").Scan(&userID)
	if userID == "" {
		log.Fatalf("❌ No user found! Run the system once to seed the admin user.")
	}

	// 3. Find an account for the cashbox
	var accountID string
	db.Raw("SELECT id FROM accounts LIMIT 1").Scan(&accountID)
	if accountID == "" {
		log.Fatalf("❌ No account found! Make sure migrations ran.")
	}

	// --- SEED EMPLOYEES ---
	var empCount int64
	db.Raw("SELECT count(*) FROM employees WHERE scope_id = ?", factoryID).Scan(&empCount)
	if empCount == 0 {
		for i := 1; i <= 3; i++ {
			empID := uuid.New()
			err = db.Exec(`
				INSERT INTO employees (id, scope_id, employee_no, full_name, phone, hire_date, status)
				VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE')
			`, empID, factoryID, fmt.Sprintf("EMP-100%d", i), fmt.Sprintf("موظف تجريبي %d", i), fmt.Sprintf("0500000%d", i), time.Now().Format("2006-01-02")).Error
			if err != nil {
				log.Println("Error seeding employee:", err)
			}
		}
		fmt.Println("✅ Seeded 3 Employees.")
	} else {
		fmt.Println("⚠️ Employees already seeded for this factory.")
	}

	// --- SEED CASHBOX ---
	var cashboxCount int64
	db.Raw("SELECT count(*) FROM cashboxes WHERE scope_id = ?", factoryID).Scan(&cashboxCount)
	if cashboxCount == 0 {
		cashboxID := uuid.New()
		err = db.Exec(`
			INSERT INTO cashboxes (id, scope_id, name, account_id, currency, status)
			VALUES (?, ?, ?, ?, 'SAR', 'ACTIVE')
		`, cashboxID, factoryID, "صندوق المصنع التشغيلي", accountID).Error
		if err == nil {
			fmt.Println("✅ Seeded 1 Cashbox.")
			// Add a cash transaction to give it a balance of 15,000 SAR
			err2 := db.Exec(`
				INSERT INTO cash_transactions (id, cashbox_id, amount, currency, direction, source_type, performed_by, transaction_date, status)
				VALUES (?, ?, 15000.0, 'SAR', 'IN', 'INITIAL_BALANCE', ?, ?, 'COMPLETED')
			`, uuid.New(), cashboxID, userID, time.Now().Format("2006-01-02")).Error
			if err2 != nil {
				log.Println("Error seeding cash transaction:", err2)
			} else {
				fmt.Println("✅ Seeded initial cash balance of 15,000 SAR.")
			}
		} else {
			log.Println("Error seeding cashbox:", err)
		}
	} else {
		fmt.Println("⚠️ Cashbox already seeded for this factory.")
		
		// Ensure it has a transaction
		var cashboxID string
		db.Raw("SELECT id FROM cashboxes WHERE scope_id = ? LIMIT 1", factoryID).Scan(&cashboxID)
		
		var txCount int64
		db.Raw("SELECT count(*) FROM cash_transactions WHERE cashbox_id = ?", cashboxID).Scan(&txCount)
		
		if txCount == 0 {
			err2 := db.Exec(`
				INSERT INTO cash_transactions (id, cashbox_id, amount, currency, direction, source_type, performed_by, transaction_date, status)
				VALUES (?, ?, 15000.0, 'SAR', 'IN', 'INITIAL_BALANCE', ?, ?, 'COMPLETED')
			`, uuid.New(), cashboxID, userID, time.Now().Format("2006-01-02")).Error
			if err2 != nil {
				log.Println("Error seeding cash transaction:", err2)
			} else {
				fmt.Println("✅ Seeded initial cash balance of 15,000 SAR.")
			}
		}
	}

	// --- SEED WAREHOUSES ---
	var whCount int64
	db.Raw("SELECT count(*) FROM warehouses WHERE scope_id = ?", factoryID).Scan(&whCount)
	if whCount == 0 {
		db.Exec(`
			INSERT INTO warehouses (id, scope_id, code, name, type, is_active)
			VALUES (?, ?, ?, ?, 'RAW_MATERIAL', true)
		`, uuid.New(), factoryID, "WH-RAW-01", "مستودع المواد الخام")
		db.Exec(`
			INSERT INTO warehouses (id, scope_id, code, name, type, is_active)
			VALUES (?, ?, ?, ?, 'FINISHED_GOODS', true)
		`, uuid.New(), factoryID, "WH-FIN-01", "مستودع المنتج النهائي")
		fmt.Println("✅ Seeded 2 Warehouses.")
	} else {
		fmt.Println("⚠️ Warehouses already seeded for this factory.")
	}

	// --- SEED REQUESTS ---
	var reqCount int64
	db.Raw("SELECT count(*) FROM financial_requests WHERE factory_id = ?", factoryID).Scan(&reqCount)
	if reqCount == 0 {
		reqID1 := uuid.New()
		db.Exec(`
			INSERT INTO financial_requests (id, document_number, scope_id, factory_id, requested_by, type, request_date, purpose, total_amount, status)
			VALUES (?, ?, ?, ?, ?, 'FINANCIAL', ?, ?, 1500.0, 'SUBMITTED')
		`, reqID1, "FIN-2023-000001", factoryID, factoryID, userID, time.Now().Format("2006-01-02"), "مصاريف تشغيلية")

		reqID2 := uuid.New()
		db.Exec(`
			INSERT INTO financial_requests (id, document_number, scope_id, factory_id, requested_by, type, request_date, purpose, total_amount, status)
			VALUES (?, ?, ?, ?, ?, 'MATERIAL', ?, ?, 5000.0, 'ACCOUNTANT_APPROVED')
		`, reqID2, "FIN-2023-000002", factoryID, factoryID, userID, time.Now().Format("2006-01-02"), "طلب مواد تعبئة وتغليف")
		
		fmt.Println("✅ Seeded 2 Requests.")
	} else {
		fmt.Println("⚠️ Requests already seeded for this factory.")
	}

	// Check cashbox balances
	type CashboxBalance struct {
		CashboxName    string
		CurrentBalance decimal.Decimal
	}
	var balances []CashboxBalance
	db.Raw("SELECT cashbox_name, current_balance FROM cashbox_balances WHERE scope_id = ?", factoryID).Scan(&balances)
	fmt.Printf("💰 Cashbox Balances: %+v\n", balances)
}
