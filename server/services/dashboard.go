package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/helpers"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DashboardServer mengimplementasikan finance.DashboardServiceServer.
type DashboardServer struct {
	finance.UnimplementedDashboardServiceServer
}

func (s *DashboardServer) GetDashboard(ctx context.Context, req *finance.GetDashboardRequest) (*finance.DashboardResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	thisMonth := time.Now().Format("2006-01")

	// ── 1. Ringkasan bulan ini ────────────────────────────────
	var totalIncome, totalExpense float64
	db.Pool.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN type='income'  THEN amount ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
		FROM transactions
		WHERE family_id = $1 AND TO_CHAR(date, 'YYYY-MM') = $2`,
		sess.FamilyID, thisMonth,
	).Scan(&totalIncome, &totalExpense)

	summary := &finance.MonthlySummary{
		Month:        thisMonth,
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		Balance:      totalIncome - totalExpense,
	}

	// ── 2. Pengeluaran per kategori vs anggaran ───────────────
	catRows, err := db.Pool.Query(ctx, `
		SELECT t.category_name,
		       COALESCE(SUM(t.amount), 0)::float8 AS spent,
		       COALESCE(MAX(b.amount), 0)::float8  AS budget
		FROM transactions t
		LEFT JOIN budgets b
		  ON b.family_id = t.family_id
		 AND b.category_name = t.category_name
		 AND b.month = $2
		WHERE t.family_id = $1
		  AND t.type = 'expense'
		  AND TO_CHAR(t.date, 'YYYY-MM') = $2
		GROUP BY t.category_name
		ORDER BY spent DESC`,
		sess.FamilyID, thisMonth,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query category spending")
	}
	defer catRows.Close()

	var catSpendings []*finance.CategorySpending
	var overspendCount, totalBudgetCats int
	for catRows.Next() {
		cs := &finance.CategorySpending{}
		catRows.Scan(&cs.CategoryName, &cs.Spent, &cs.Budget)
		cs.Remaining = cs.Budget - cs.Spent
		if cs.Budget > 0 {
			cs.PctUsed = cs.Spent / cs.Budget * 100
			totalBudgetCats++
			if cs.Spent > cs.Budget {
				overspendCount++
			}
		}
		catSpendings = append(catSpendings, cs)
	}

	// ── 3. Pengeluaran per anggota ────────────────────────────
	memRows, err := db.Pool.Query(ctx, `
		SELECT member, COALESCE(SUM(amount), 0)::float8
		FROM transactions
		WHERE family_id = $1
		  AND type = 'expense'
		  AND TO_CHAR(date, 'YYYY-MM') = $2
		GROUP BY member ORDER BY member`,
		sess.FamilyID, thisMonth,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query member spending")
	}
	defer memRows.Close()

	var memSpendings []*finance.MemberSpending
	for memRows.Next() {
		ms := &finance.MemberSpending{}
		memRows.Scan(&ms.Member, &ms.Spent)
		memSpendings = append(memSpendings, ms)
	}

	// ── 4. Hutang aktif bulan berjalan ────────────────────────
	// Ambil semua hutang aktif (belum lunas) — ini adalah hutang yang perlu dibayar bulan ini.
	// Sekaligus hitung total sisa hutang & total cicilan bulan berjalan.
	debtRows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, name, total_amount::float8, remaining_amount::float8,
		       monthly_payment::float8, due_date::text, is_paid, days_until_due, created_at::text
		FROM debts
		WHERE family_id = $1 AND is_paid = FALSE
		ORDER BY days_until_due ASC`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query monthly debts")
	}
	defer debtRows.Close()

	var monthlyDebts []*finance.Debt
	var totalRemainingDebt, monthlyDebtTotal float64
	for debtRows.Next() {
		d := &finance.Debt{}
		debtRows.Scan(&d.Id, &d.FamilyId, &d.Name, &d.TotalAmount, &d.RemainingAmount,
			&d.MonthlyPayment, &d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt)
		totalRemainingDebt += d.RemainingAmount
		monthlyDebtTotal += d.MonthlyPayment
		monthlyDebts = append(monthlyDebts, d)
	}

	// ── 5. Ringkasan tabungan ─────────────────────────────────
	savRows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, name, target_amount::float8, current_balance::float8,
		       target_date::text, monthly_required::float8, progress_pct::float8, created_at::text
		FROM savings WHERE family_id = $1 ORDER BY progress_pct DESC`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query savings")
	}
	defer savRows.Close()

	var savSummary []*finance.Saving
	var totalSavCurrent, totalSavTarget float64
	for savRows.Next() {
		sv := &finance.Saving{}
		savRows.Scan(&sv.Id, &sv.FamilyId, &sv.Name, &sv.TargetAmount, &sv.CurrentBalance,
			&sv.TargetDate, &sv.MonthlyRequired, &sv.ProgressPct, &sv.CreatedAt)
		totalSavCurrent += sv.CurrentBalance
		totalSavTarget += sv.TargetAmount
		savSummary = append(savSummary, sv)
	}

	// ── 6. Dana darurat ───────────────────────────────────────
	ef := &finance.EmergencyFund{}
	db.Pool.QueryRow(ctx, `
		SELECT id, family_id, current_balance::float8, monthly_expense_avg::float8,
		       target_balance::float8, months_covered::float8, updated_at::text
		FROM emergency_fund WHERE family_id = $1`,
		sess.FamilyID,
	).Scan(&ef.Id, &ef.FamilyId, &ef.CurrentBalance, &ef.MonthlyExpenseAvg,
		&ef.TargetBalance, &ef.MonthsCovered, &ef.UpdatedAt)

	// ── 7. Health Score ───────────────────────────────────────
	hs := calcHealthScore(totalSavCurrent, totalSavTarget, totalBudgetCats, overspendCount,
		ef.MonthsCovered, totalIncome, monthlyDebts)

	return &finance.DashboardResponse{
		Success:            true,
		HealthScore:        hs,
		ThisMonth:          summary,
		CategorySpending:   catSpendings,
		MemberSpending:     memSpendings,
		MonthlyDebts:       monthlyDebts,
		SavingsSummary:     savSummary,
		EmergencyFund:      ef,
		TotalRemainingDebt: totalRemainingDebt,
		MonthlyDebtTotal:   monthlyDebtTotal,
	}, nil
}

func calcHealthScore(savCurrent, savTarget float64, totalBudgetCats, overspend int,
	monthsCovered, totalIncome float64, upcomingDebts []*finance.Debt) *finance.HealthScore {

	// 1. Savings Score (Maks 30)
	var savScore float64
	if savTarget > 0 {
		savScore = math.Min(savCurrent/savTarget, 1.0) * 30.0
	} else if savCurrent > 0 {
		savScore = 15.0 // baseline jika ada tabungan tanpa target spesifik
	} else {
		savScore = 0.0
	}

	// 2. Budget Discipline Score (Maks 25)
	var budScore float64
	if totalBudgetCats > 0 {
		ratio := float64(totalBudgetCats-overspend) / float64(totalBudgetCats)
		if ratio < 0 {
			ratio = 0
		}
		budScore = ratio * 25.0
	} else {
		// Jika belum pasang anggaran sama sekali, beri baseline 15
		budScore = 15.0
	}

	// 3. Emergency Fund Score (Maks 25)
	efScore := math.Min(monthsCovered/6.0, 1.0) * 25.0

	// 4. Debt Service Ratio Score (Maks 20)
	var totalMonthlyDebt float64
	for _, d := range upcomingDebts {
		totalMonthlyDebt += d.MonthlyPayment
	}
	var debtScore float64 = 20.0
	if totalIncome > 0 {
		debtRatio := totalMonthlyDebt / totalIncome
		if debtRatio <= 0.30 {
			// Rasio cicilan <= 30% dari penghasilan dianggap sehat
			debtScore = 20.0 - (debtRatio / 0.30 * 4.0) // 16 - 20
		} else if debtRatio <= 0.50 {
			// Rasio 30% - 50% waspada
			debtScore = 16.0 - ((debtRatio - 0.30) / 0.20 * 8.0) // 8 - 16
		} else {
			// Rasio > 50% kritis
			debtScore = math.Max(0.0, 8.0-((debtRatio-0.50)/0.50*8.0))
		}
	} else if totalMonthlyDebt > 0 {
		debtScore = 5.0
	}

	total := math.Min(100.0, math.Max(0.0, savScore+budScore+efScore+debtScore))

	// Saran dinamis berdasarkan analisis menyeluruh
	advice := generateAdvice(savScore, budScore, efScore, debtScore, total, monthsCovered, totalMonthlyDebt, totalIncome, totalBudgetCats, overspend)

	return &finance.HealthScore{
		SavingsScore:       math.Round(savScore*10) / 10,
		BudgetScore:        math.Round(budScore*10) / 10,
		EmergencyFundScore: math.Round(efScore*10) / 10,
		DebtScore:          math.Round(debtScore*10) / 10,
		Total:              math.Round(total),
		Advice:             advice,
	}
}

func generateAdvice(savScore, budScore, efScore, debtScore, total, monthsCovered, totalMonthlyDebt, totalIncome float64, totalBudgetCats, overspend int) string {
	savRatio := savScore / 30.0
	budRatio := budScore / 25.0
	efRatio := efScore / 25.0
	debtRatio := debtScore / 20.0

	// Jika ada kondisi darurat mendesak
	if totalIncome > 0 && totalMonthlyDebt/totalIncome > 0.40 {
		return "⚠️ Beban cicilan melebihi 40% dari total pemasukan. Hindari menambah hutang baru dan fokus alokasikan dana untuk melunasi hutang berjalan."
	}
	if overspend > 0 && totalBudgetCats > 0 && float64(overspend)/float64(totalBudgetCats) >= 0.5 {
		return "⚠️ Lebih dari 50% pos anggaran Anda mengalami overbudget bulan ini. Disarankan melakukan rem pengeluaran non-esensial."
	}
	if monthsCovered < 1.0 {
		return "🛡️ Dana darurat Anda masih di bawah 1 bulan pengeluaran. Jadikan pengisian dana darurat sebagai prioritas utama."
	}

	// Evaluasi pilar terlemah
	minRatio := math.Min(savRatio, math.Min(budRatio, math.Min(efRatio, debtRatio)))
	switch {
	case minRatio == efRatio && efRatio < 0.6:
		return fmt.Sprintf("🛡️ Dana darurat saat ini baru mencakup %.1f bulan pengeluaran. Tingkatkan hingga idealnya 6 bulan untuk stabilitas finansial.", monthsCovered)
	case minRatio == budRatio && budRatio < 0.7:
		return "📋 Kedisiplinan anggaran perlu ditingkatkan. Periksa kembali pengeluaran pos yang melebihi batas dan buat penyesuaian."
	case minRatio == debtRatio && debtRatio < 0.7:
		return "🏦 Porsi cicilan hutang cukup menggerus arus kas bulanan. Pertimbangkan strategi debt avalanche atau debt snowball untuk percepatan pelunasan."
	case minRatio == savRatio && savRatio < 0.6:
		return "🎯 Capaian target tabungan masih belum optimal. Sisihkan minimal 10%–20% dari setiap pemasukan di awal bulan secara konsisten."
	case total >= 80:
		return "🌟 Luar biasa! Kesehatan finansial keluarga Anda sangat sehat dan terjaga dengan baik. Terus pertahankan pengelolaan anggaran yang disiplin!"
	default:
		return "💡 Kondisi keuangan keluarga cukup stabil. Jaga keseimbangan antara alokasi tabungan, dana darurat, dan konsumsi harian."
	}
}

func (s *DashboardServer) SeedData(ctx context.Context, req *finance.SeedDataRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Data transaksi 3 bulan terakhir
	type txSeed struct {
		typ      string
		amount   float64
		cat      string
		member   string
		daysAgo  int
		note     string
	}
	seeds := []txSeed{
		// Bulan ini
		{"income",  8000000, "Gaji",         "Ayah",  5,  "Gaji bulan ini"},
		{"income",  5000000, "Gaji",         "Ibu",   5,  "Gaji bulan ini"},
		{"expense", 1500000, "Makanan",      "Ibu",   3,  "Belanja bulanan"},
		{"expense",  450000, "Transport",    "Ayah",  4,  "BBM dan parkir"},
		{"expense",  350000, "Tagihan",      "Ayah",  2,  "Listrik PLN"},
		{"expense",  200000, "Tagihan",      "Ibu",   1,  "Internet"},
		{"expense",  300000, "Kesehatan",    "Ibu",   6,  "Obat-obatan"},
		{"expense",  150000, "Hiburan",      "Anak",  7,  "Langganan streaming"},
		// Bulan lalu
		{"income",  8000000, "Gaji",         "Ayah",  35, "Gaji bulan lalu"},
		{"income",  5000000, "Gaji",         "Ibu",   35, "Gaji bulan lalu"},
		{"expense", 1800000, "Makanan",      "Ibu",   32, "Belanja bulanan"},
		{"expense",  500000, "Transport",    "Ayah",  33, "BBM"},
		{"expense",  350000, "Tagihan",      "Ayah",  31, "Listrik"},
		{"expense",  200000, "Tagihan",      "Ibu",   30, "Internet"},
		{"expense",  500000, "Pendidikan",   "Anak",  28, "SPP sekolah"},
		{"expense",  250000, "Belanja",      "Ibu",   29, "Kebutuhan rumah"},
		// 2 bulan lalu
		{"income",  8000000, "Gaji",         "Ayah",  65, "Gaji 2 bulan lalu"},
		{"income",  5000000, "Gaji",         "Ibu",   65, "Gaji 2 bulan lalu"},
		{"income",  1000000, "Bonus",        "Ayah",  60, "Bonus lembur"},
		{"expense", 2000000, "Makanan",      "Ibu",   62, "Belanja bulanan"},
		{"expense",  400000, "Transport",    "Ayah",  63, "BBM"},
		{"expense",  350000, "Tagihan",      "Ayah",  61, "Listrik"},
		{"expense",  200000, "Tagihan",      "Ibu",   60, "Internet"},
		{"expense",  800000, "Kesehatan",    "Ibu",   58, "Dokter dan obat"},
	}

	for _, seed := range seeds {
		date := now.AddDate(0, 0, -seed.daysAgo).Format("2006-01-02")
		_, err = tx.Exec(ctx, `
			INSERT INTO transactions (family_id, created_by, type, amount, category_name, member, date, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8)`,
			sess.FamilyID, sess.UserID, seed.typ, seed.amount, seed.cat, seed.member, date, seed.note,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "gagal insert seed transaksi: "+err.Error())
		}
	}

	// Budget bulan ini
	type budSeed struct{ cat string; amount float64 }
	budgets := []budSeed{
		{"Makanan", 2000000}, {"Transport", 600000}, {"Tagihan", 700000},
		{"Kesehatan", 500000}, {"Pendidikan", 600000}, {"Hiburan", 300000},
		{"Belanja", 500000}, {"Lainnya", 300000},
	}
	for _, b := range budgets {
		_, err = tx.Exec(ctx, `
			INSERT INTO budgets (family_id, category_name, month, amount)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (family_id, category_name, month) DO NOTHING`,
			sess.FamilyID, b.cat, thisMonth(now), b.amount,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "gagal insert seed budget")
		}
	}

	// 1 hutang KPR (selalu insert baru, tidak pakai ON CONFLICT karena tidak ada unique constraint)
	dueDate := now.AddDate(0, 3, 0).Format("2006-01-02")
	daysUntilDue := helpers.RecalcDaysUntilDue(dueDate)
	_, err = tx.Exec(ctx, `
		INSERT INTO debts (family_id, name, total_amount, remaining_amount, monthly_payment, due_date, days_until_due)
		VALUES ($1, 'KPR Rumah', 150000000, 98000000, 2500000, $2::date, $3)`,
		sess.FamilyID, dueDate, daysUntilDue,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert seed hutang")
	}

	// 1 tabungan Dana Pendidikan
	targetDate := now.AddDate(2, 0, 0).Format("2006-01-02")
	monthly, pct := helpers.RecalcSavingFields(5000000, 30000000, targetDate)
	_, err = tx.Exec(ctx, `
		INSERT INTO savings (family_id, name, target_amount, current_balance, target_date, monthly_required, progress_pct)
		VALUES ($1, 'Dana Pendidikan Anak', 30000000, 5000000, $2::date, $3, $4)`,
		sess.FamilyID, targetDate, monthly, pct,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert seed tabungan")
	}

	// Dana darurat
	_, err = tx.Exec(ctx, `
		INSERT INTO emergency_fund (family_id, current_balance, monthly_expense_avg, target_balance, months_covered, updated_at)
		VALUES ($1, 12000000, 4500000, 27000000, 2.67, NOW())
		ON CONFLICT (family_id) DO UPDATE
		SET current_balance = 12000000, monthly_expense_avg = 4500000,
		    target_balance = 27000000, months_covered = 2.67, updated_at = NOW()`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert seed dana darurat")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit seed data")
	}

	return &finance.StatusResponse{Success: true, Message: fmt.Sprintf("Data contoh berhasil dimuat untuk family %s", sess.FamilyID)}, nil
}

func (s *DashboardServer) ClearData(ctx context.Context, req *finance.ClearDataRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// 1. Hapus child tables yang tidak punya family_id langsung
	_, err = tx.Exec(ctx, `
		DELETE FROM saving_deposits
		WHERE saving_id IN (SELECT id FROM savings WHERE family_id = $1)`,
		sess.FamilyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus saving_deposits")
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM debt_payments
		WHERE debt_id IN (SELECT id FROM debts WHERE family_id = $1)`,
		sess.FamilyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus debt_payments")
	}

	// 2. Null-kan FK transactions.debt_payment_id sebelum hapus transactions
	_, err = tx.Exec(ctx,
		`UPDATE transactions SET debt_payment_id = NULL WHERE family_id = $1`, sess.FamilyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal null debt_payment_id")
	}

	// 3. Hapus tabel utama
	for _, tbl := range []string{"transactions", "debts", "savings", "budgets"} {
		if _, err = tx.Exec(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE family_id = $1`, tbl),
			sess.FamilyID); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal hapus %s: %v", tbl, err)
		}
	}

	// 4. Reset emergency_fund (jangan hapus baris, cukup nolkan)
	_, err = tx.Exec(ctx, `
		INSERT INTO emergency_fund (family_id) VALUES ($1)
		ON CONFLICT (family_id) DO UPDATE
		SET current_balance = 0, monthly_expense_avg = 0,
		    target_balance = 0, months_covered = 0, updated_at = NOW()`,
		sess.FamilyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal reset emergency_fund")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit clear data")
	}

	return &finance.StatusResponse{Success: true, Message: "Semua data berhasil dihapus"}, nil
}

func thisMonth(t time.Time) string { return t.Format("2006-01") }

// ─────────────────────────────────────────────────────────────────────────────
// ExportBackup — mengambil semua data keluarga dan mengembalikannya sebagai JSON
// ─────────────────────────────────────────────────────────────────────────────

type backupData struct {
	ExportedAt   string                `json:"exported_at"`
	FamilyID     string                `json:"family_id"`
	Categories   []backupCategory      `json:"categories"`
	Wallets      []backupWallet        `json:"wallets"`
	Transactions []backupTransaction   `json:"transactions"`
	Budgets      []backupBudget        `json:"budgets"`
	Debts        []backupDebt          `json:"debts"`
	DebtPayments []backupDebtPayment   `json:"debt_payments"`
	Savings      []backupSaving        `json:"savings"`
	Deposits     []backupDeposit       `json:"saving_deposits"`
	Emergency    *backupEmergencyFund  `json:"emergency_fund"`
}

type backupCategory struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type backupWallet struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	Balance float64 `json:"balance"`
	Color   string  `json:"color"`
	Note    string  `json:"note"`
}

type backupTransaction struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	CategoryName  string  `json:"category_name"`
	Member        string  `json:"member"`
	Date          string  `json:"date"`
	Note          string  `json:"note"`
	DebtPaymentID *string `json:"debt_payment_id,omitempty"`
	WalletID      *string `json:"wallet_id,omitempty"`
}

type backupBudget struct {
	CategoryName string  `json:"category_name"`
	Month        string  `json:"month"`
	Amount       float64 `json:"amount"`
}

type backupDebt struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	TotalAmount    float64 `json:"total_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
	MonthlyPayment float64 `json:"monthly_payment"`
	DueDate        string  `json:"due_date"`
	IsPaid         bool    `json:"is_paid"`
}

type backupDebtPayment struct {
	ID     string  `json:"id"`
	DebtID string  `json:"debt_id"`
	PaidBy string  `json:"paid_by"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
}

type backupSaving struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	TargetAmount   float64 `json:"target_amount"`
	CurrentBalance float64 `json:"current_balance"`
	TargetDate     string  `json:"target_date"`
}

type backupDeposit struct {
	SavingID string  `json:"saving_id"`
	Amount   float64 `json:"amount"`
	Date     string  `json:"date"`
	Note     string  `json:"note"`
}

type backupEmergencyFund struct {
	CurrentBalance    float64 `json:"current_balance"`
	MonthlyExpenseAvg float64 `json:"monthly_expense_avg"`
}

func (s *DashboardServer) ExportBackup(ctx context.Context, req *finance.ExportBackupRequest) (*finance.BackupResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	fid := sess.FamilyID

	bd, err := CollectFamilyBackupData(ctx, fid)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Trigger penyimpanan / replace backup mingguan di server
	_ = BackupFamilyToFile(ctx, fid)

	jsonBytes, err := json.Marshal(bd)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal encode backup JSON")
	}

	return &finance.BackupResponse{
		Success:    true,
		Message:    fmt.Sprintf("Backup berhasil: %d kategori, %d dompet, %d transaksi, %d anggaran, %d hutang, %d tabungan", len(bd.Categories), len(bd.Wallets), len(bd.Transactions), len(bd.Budgets), len(bd.Debts), len(bd.Savings)),
		JsonData:   string(jsonBytes),
		ExportedAt: bd.ExportedAt,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ImportRestore — menggantikan data keluarga dengan isi file backup
// ─────────────────────────────────────────────────────────────────────────────

func (s *DashboardServer) ImportRestore(ctx context.Context, req *finance.ImportRestoreRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	fid := sess.FamilyID

	var bd backupData
	if err := json.Unmarshal([]byte(req.JsonData), &bd); err != nil {
		return nil, status.Error(codes.InvalidArgument, "format JSON backup tidak valid")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// 1. Hapus data lama (sama urutan seperti ClearData)
	tx.Exec(ctx, `DELETE FROM saving_deposits WHERE saving_id IN (SELECT id FROM savings WHERE family_id = $1)`, fid)
	tx.Exec(ctx, `DELETE FROM debt_payments WHERE debt_id IN (SELECT id FROM debts WHERE family_id = $1)`, fid)
	tx.Exec(ctx, `UPDATE transactions SET debt_payment_id = NULL, wallet_id = NULL WHERE family_id = $1`, fid)
	tx.Exec(ctx, `DELETE FROM wallet_transfers WHERE family_id = $1`, fid)
	for _, tbl := range []string{"transactions", "debts", "savings", "budgets", "wallets"} {
		tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE family_id = $1`, tbl), fid)
	}

	// 1b. Restore categories (hapus dulu yang ada, insert ulang dari backup)
	if len(bd.Categories) > 0 {
		tx.Exec(ctx, `DELETE FROM categories WHERE family_id = $1`, fid)
		for _, c := range bd.Categories {
			tx.Exec(ctx,
				`INSERT INTO categories (family_id, name, type) VALUES ($1, $2, $3)
				 ON CONFLICT (family_id, name) DO UPDATE SET type = EXCLUDED.type`,
				fid, c.Name, c.Type,
			)
		}
	}

	// 1c. Restore wallets (harus ada sebelum transactions)
	for _, w := range bd.Wallets {
		if _, err = tx.Exec(ctx,
			`INSERT INTO wallets (id, family_id, name, type, balance, color, note)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (family_id, name) DO UPDATE SET type=EXCLUDED.type, balance=EXCLUDED.balance, color=EXCLUDED.color, note=EXCLUDED.note`,
			w.ID, fid, w.Name, w.Type, w.Balance, w.Color, w.Note,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert dompet '%s': %v", w.Name, err)
		}
	}

	// 2. Insert debts (debt_payments butuh debt.id)
	for _, d := range bd.Debts {
		daysUntilDue := helpers.RecalcDaysUntilDue(d.DueDate)
		monthly, pct := helpers.RecalcSavingFields(d.RemainingAmount, d.TotalAmount, d.DueDate)
		_ = monthly; _ = pct
		if _, err = tx.Exec(ctx,
			`INSERT INTO debts (id, family_id, name, total_amount, remaining_amount, monthly_payment, due_date, is_paid, days_until_due)
			 VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9)`,
			d.ID, fid, d.Name, d.TotalAmount, d.RemainingAmount, d.MonthlyPayment, d.DueDate, d.IsPaid, daysUntilDue,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert hutang '%s': %v", d.Name, err)
		}
	}

	// 3. Insert debt_payments
	for _, p := range bd.DebtPayments {
		if _, err = tx.Exec(ctx,
			`INSERT INTO debt_payments (id, debt_id, paid_by, amount, date) VALUES ($1, $2, $3, $4, $5::date)`,
			p.ID, p.DebtID, p.PaidBy, p.Amount, p.Date,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert debt_payment: %v", err)
		}
	}

	// 4. Insert transactions (debt_payment_id dan wallet_id sudah ada sekarang)
	for _, t := range bd.Transactions {
		if _, err = tx.Exec(ctx,
			`INSERT INTO transactions (id, family_id, created_by, type, amount, category_name, member, date, note, debt_payment_id, wallet_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8::date, $9, $10::uuid, $11::uuid)`,
			t.ID, fid, sess.UserID, t.Type, t.Amount, t.CategoryName, t.Member, t.Date, t.Note, t.DebtPaymentID, t.WalletID,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert transaksi: %v", err)
		}
	}

	// 5. Insert budgets
	for _, b := range bd.Budgets {
		if _, err = tx.Exec(ctx,
			`INSERT INTO budgets (family_id, category_name, month, amount) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (family_id, category_name, month) DO UPDATE SET amount = EXCLUDED.amount`,
			fid, b.CategoryName, b.Month, b.Amount,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert anggaran: %v", err)
		}
	}

	// 6. Insert savings
	for _, sv := range bd.Savings {
		monthly, pct := helpers.RecalcSavingFields(sv.CurrentBalance, sv.TargetAmount, sv.TargetDate)
		if _, err = tx.Exec(ctx,
			`INSERT INTO savings (id, family_id, name, target_amount, current_balance, target_date, monthly_required, progress_pct)
			 VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8)`,
			sv.ID, fid, sv.Name, sv.TargetAmount, sv.CurrentBalance, sv.TargetDate, monthly, pct,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert tabungan '%s': %v", sv.Name, err)
		}
	}

	// 7. Insert saving_deposits
	for _, dep := range bd.Deposits {
		if _, err = tx.Exec(ctx,
			`INSERT INTO saving_deposits (saving_id, amount, date, note) VALUES ($1, $2, $3::date, $4)`,
			dep.SavingID, dep.Amount, dep.Date, dep.Note,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal insert setoran tabungan: %v", err)
		}
	}

	// 8. Restore emergency fund
	if bd.Emergency != nil {
		ef := bd.Emergency
		target := ef.MonthlyExpenseAvg * 6
		months := 0.0
		if ef.MonthlyExpenseAvg > 0 {
			months = ef.CurrentBalance / ef.MonthlyExpenseAvg
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO emergency_fund (family_id, current_balance, monthly_expense_avg, target_balance, months_covered, updated_at)
			 VALUES ($1, $2, $3, $4, $5, NOW())
			 ON CONFLICT (family_id) DO UPDATE
			 SET current_balance = $2, monthly_expense_avg = $3, target_balance = $4, months_covered = $5, updated_at = NOW()`,
			fid, ef.CurrentBalance, ef.MonthlyExpenseAvg, target, months,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "gagal restore dana darurat: %v", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit restore data")
	}

	return &finance.StatusResponse{
		Success: true,
		Message: fmt.Sprintf("Restore berhasil: %d kategori, %d dompet, %d transaksi, %d anggaran, %d hutang, %d tabungan",
			len(bd.Categories), len(bd.Wallets), len(bd.Transactions), len(bd.Budgets), len(bd.Debts), len(bd.Savings)),
	}, nil
}
