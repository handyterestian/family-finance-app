package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"family-finance/server/db"
)

var (
	backupDir = "./backups"
	cronOnce  sync.Once
)

// InitAutoBackupScheduler menjalankan background ticker otomatis 1 minggu sekali
// File auto backup disimpan di server (./backups/auto_backup_<family_id>.json)
// dan di-replace setiap minggunya untuk menghemat ruang disk (storage).
func InitAutoBackupScheduler(ctx context.Context) {
	cronOnce.Do(func() {
		if envDir := os.Getenv("BACKUP_DIR"); envDir != "" {
			backupDir = envDir
		}
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			log.Printf("[AutoBackup] Gagal membuat direktori backup: %v", err)
		}

		go func() {
			log.Println("[AutoBackup] Scheduler aktif (interval 1 minggu). Melakukan backup pertama...")
			runAutoBackupAllFamilies(ctx)

			ticker := time.NewTicker(7 * 24 * time.Hour)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					log.Println("[AutoBackup] Scheduler berhenti.")
					return
				case <-ticker.C:
					log.Println("[AutoBackup] Menjalankan auto backup mingguan rutin...")
					runAutoBackupAllFamilies(ctx)
				}
			}
		}()
	})
}

// runAutoBackupAllFamilies melakukan query ke semua family_id dan menyimpan backup-nya
func runAutoBackupAllFamilies(ctx context.Context) {
	if db.Pool == nil {
		log.Println("[AutoBackup] Database belum terhubung.")
		return
	}

	rows, err := db.Pool.Query(ctx, `SELECT id FROM families`)
	if err != nil {
		log.Printf("[AutoBackup] Gagal query families: %v", err)
		return
	}
	defer rows.Close()

	var familyIDs []string
	for rows.Next() {
		var fid string
		if err := rows.Scan(&fid); err == nil {
			familyIDs = append(familyIDs, fid)
		}
	}

	for _, fid := range familyIDs {
		if err := BackupFamilyToFile(ctx, fid); err != nil {
			log.Printf("[AutoBackup] Gagal backup family %s: %v", fid, err)
		} else {
			log.Printf("[AutoBackup] Sukses update/replace auto backup untuk family %s", fid)
		}
	}
}

// BackupFamilyToFile membuat data JSON backup keluarga dan menimpa (replace) file mingguan di server
func BackupFamilyToFile(ctx context.Context, familyID string) error {
	bd, err := CollectFamilyBackupData(ctx, familyID)
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(bd, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal encode JSON: %w", err)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("gagal memastikan direktori backup: %w", err)
	}

	// Nama file tetap/konstan per keluarga agar di-replace setiap minggunya (menghemat disk storage server)
	filePath := filepath.Join(backupDir, fmt.Sprintf("auto_backup_%s.json", familyID))
	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("gagal menulis file backup: %w", err)
	}

	return nil
}

// CollectFamilyBackupData mengumpulkan semua data keluarga dari database
func CollectFamilyBackupData(ctx context.Context, fid string) (*backupData, error) {
	bd := &backupData{
		ExportedAt: time.Now().Format(time.RFC3339),
		FamilyID:   fid,
	}

	// Categories
	rows, err := db.Pool.Query(ctx,
		`SELECT name, type FROM categories WHERE family_id = $1 ORDER BY name`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query categories: %w", err)
	}
	for rows.Next() {
		var c backupCategory
		rows.Scan(&c.Name, &c.Type)
		bd.Categories = append(bd.Categories, c)
	}
	rows.Close()

	// Wallets
	rows, err = db.Pool.Query(ctx,
		`SELECT id, name, type, balance::float8, color, note FROM wallets WHERE family_id = $1 ORDER BY name`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query wallets: %w", err)
	}
	for rows.Next() {
		var w backupWallet
		rows.Scan(&w.ID, &w.Name, &w.Type, &w.Balance, &w.Color, &w.Note)
		bd.Wallets = append(bd.Wallets, w)
	}
	rows.Close()

	// Transactions
	rows, err = db.Pool.Query(ctx,
		`SELECT id, type, amount::float8, category_name, member, date::text, note,
		        debt_payment_id::text, wallet_id::text
		 FROM transactions WHERE family_id = $1 ORDER BY date`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query transactions: %w", err)
	}
	for rows.Next() {
		var t backupTransaction
		rows.Scan(&t.ID, &t.Type, &t.Amount, &t.CategoryName, &t.Member, &t.Date, &t.Note, &t.DebtPaymentID, &t.WalletID)
		bd.Transactions = append(bd.Transactions, t)
	}
	rows.Close()

	// Budgets
	rows, err = db.Pool.Query(ctx,
		`SELECT category_name, month, amount::float8 FROM budgets WHERE family_id = $1 ORDER BY month, category_name`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query budgets: %w", err)
	}
	for rows.Next() {
		var b backupBudget
		rows.Scan(&b.CategoryName, &b.Month, &b.Amount)
		bd.Budgets = append(bd.Budgets, b)
	}
	rows.Close()

	// Debts
	rows, err = db.Pool.Query(ctx,
		`SELECT id, name, total_amount::float8, remaining_amount::float8,
		        monthly_payment::float8, due_date::text, is_paid
		 FROM debts WHERE family_id = $1 ORDER BY due_date`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query debts: %w", err)
	}
	for rows.Next() {
		var d backupDebt
		rows.Scan(&d.ID, &d.Name, &d.TotalAmount, &d.RemainingAmount, &d.MonthlyPayment, &d.DueDate, &d.IsPaid)
		bd.Debts = append(bd.Debts, d)
	}
	rows.Close()

	// Debt payments
	rows, err = db.Pool.Query(ctx,
		`SELECT dp.id, dp.debt_id, dp.paid_by, dp.amount::float8, dp.date::text
		 FROM debt_payments dp
		 JOIN debts d ON d.id = dp.debt_id
		 WHERE d.family_id = $1 ORDER BY dp.date`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query debt_payments: %w", err)
	}
	for rows.Next() {
		var p backupDebtPayment
		rows.Scan(&p.ID, &p.DebtID, &p.PaidBy, &p.Amount, &p.Date)
		bd.DebtPayments = append(bd.DebtPayments, p)
	}
	rows.Close()

	// Savings
	rows, err = db.Pool.Query(ctx,
		`SELECT id, name, target_amount::float8, current_balance::float8, target_date::text
		 FROM savings WHERE family_id = $1 ORDER BY created_at`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query savings: %w", err)
	}
	for rows.Next() {
		var sv backupSaving
		rows.Scan(&sv.ID, &sv.Name, &sv.TargetAmount, &sv.CurrentBalance, &sv.TargetDate)
		bd.Savings = append(bd.Savings, sv)
	}
	rows.Close()

	// Saving deposits
	rows, err = db.Pool.Query(ctx,
		`SELECT sd.saving_id, sd.amount::float8, sd.date::text, sd.note
		 FROM saving_deposits sd
		 JOIN savings sv ON sv.id = sd.saving_id
		 WHERE sv.family_id = $1 ORDER BY sd.date`, fid)
	if err != nil {
		return nil, fmt.Errorf("gagal query saving_deposits: %w", err)
	}
	for rows.Next() {
		var dep backupDeposit
		rows.Scan(&dep.SavingID, &dep.Amount, &dep.Date, &dep.Note)
		bd.Deposits = append(bd.Deposits, dep)
	}
	rows.Close()

	// Emergency fund
	var ef backupEmergencyFund
	db.Pool.QueryRow(ctx,
		`SELECT current_balance::float8, monthly_expense_avg::float8 FROM emergency_fund WHERE family_id = $1`, fid,
	).Scan(&ef.CurrentBalance, &ef.MonthlyExpenseAvg)
	bd.Emergency = &ef

	return bd, nil
}
