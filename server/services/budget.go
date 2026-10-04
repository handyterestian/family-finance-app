package services

import (
	"context"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/helpers"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BudgetServer mengimplementasikan finance.BudgetServiceServer.
type BudgetServer struct {
	finance.UnimplementedBudgetServiceServer
}

func (s *BudgetServer) SetBudget(ctx context.Context, req *finance.SetBudgetRequest) (*finance.BudgetResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.CategoryName == "" || req.Month == "" {
		return nil, status.Error(codes.InvalidArgument, "category_name dan month wajib diisi")
	}
	if req.Amount < 0 {
		return nil, status.Error(codes.InvalidArgument, "amount tidak boleh negatif")
	}

	b := &finance.Budget{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO budgets (family_id, category_name, month, amount)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (family_id, category_name, month)
		DO UPDATE SET amount = EXCLUDED.amount
		RETURNING id, family_id, category_name, month, amount::float8, created_at::text`,
		sess.FamilyID, req.CategoryName, req.Month, req.Amount,
	).Scan(&b.Id, &b.FamilyId, &b.CategoryName, &b.Month, &b.Amount, &b.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal set anggaran: "+err.Error())
	}

	return &finance.BudgetResponse{Success: true, Message: "Anggaran berhasil disimpan", Budget: b}, nil
}

func (s *BudgetServer) ListBudgets(ctx context.Context, req *finance.ListBudgetsRequest) (*finance.BudgetListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.Month == "" {
		return nil, status.Error(codes.InvalidArgument, "month wajib diisi")
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, category_name, month, amount::float8, created_at::text
		FROM budgets
		WHERE family_id = $1 AND month = $2
		ORDER BY category_name`,
		sess.FamilyID, req.Month,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query anggaran")
	}
	defer rows.Close()

	var buds []*finance.Budget
	for rows.Next() {
		b := &finance.Budget{}
		if err := rows.Scan(&b.Id, &b.FamilyId, &b.CategoryName, &b.Month, &b.Amount, &b.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan anggaran")
		}
		buds = append(buds, b)
	}
	return &finance.BudgetListResponse{Success: true, Budgets: buds}, nil
}

func (s *BudgetServer) UpdateBudget(ctx context.Context, req *finance.UpdateBudgetRequest) (*finance.BudgetResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.BudgetId == "" {
		return nil, status.Error(codes.InvalidArgument, "budget_id wajib diisi")
	}
	if req.Amount < 0 {
		return nil, status.Error(codes.InvalidArgument, "amount tidak boleh negatif")
	}

	b := &finance.Budget{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE budgets SET amount = $1
		WHERE id = $2 AND family_id = $3
		RETURNING id, family_id, category_name, month, amount::float8, created_at::text`,
		req.Amount, req.BudgetId, sess.FamilyID,
	).Scan(&b.Id, &b.FamilyId, &b.CategoryName, &b.Month, &b.Amount, &b.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.NotFound, "anggaran tidak ditemukan")
	}

	return &finance.BudgetResponse{Success: true, Message: "Anggaran berhasil diupdate", Budget: b}, nil
}

func (s *BudgetServer) DeleteBudget(ctx context.Context, req *finance.DeleteBudgetRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	res, err := db.Pool.Exec(ctx,
		`DELETE FROM budgets WHERE id = $1 AND family_id = $2`,
		req.BudgetId, sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus anggaran")
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "anggaran tidak ditemukan")
	}

	return &finance.StatusResponse{Success: true, Message: "Anggaran berhasil dihapus"}, nil
}

func (s *BudgetServer) GetEmergencyFund(ctx context.Context, req *finance.GetEmergencyFundRequest) (*finance.EmergencyFundResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	ef := &finance.EmergencyFund{}
	err = db.Pool.QueryRow(ctx, `
		SELECT id, family_id, current_balance::float8, monthly_expense_avg::float8,
		       target_balance::float8, months_covered::float8, updated_at::text, target_months
		FROM emergency_fund
		WHERE family_id = $1`,
		sess.FamilyID,
	).Scan(&ef.Id, &ef.FamilyId, &ef.CurrentBalance, &ef.MonthlyExpenseAvg,
		&ef.TargetBalance, &ef.MonthsCovered, &ef.UpdatedAt, &ef.TargetMonths)
	if err != nil {
		ef = &finance.EmergencyFund{FamilyId: sess.FamilyID, TargetMonths: 6}
	}

	return &finance.EmergencyFundResponse{Success: true, EmergencyFund: ef}, nil
}

func (s *BudgetServer) SetEmergencyFund(ctx context.Context, req *finance.SetEmergencyFundRequest) (*finance.EmergencyFundResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.CurrentBalance < 0 {
		return nil, status.Error(codes.InvalidArgument, "current_balance tidak boleh negatif")
	}

	// Tentukan target_months: ambil nilai lama jika req.TargetMonths == 0
	targetMonths := req.TargetMonths
	if targetMonths <= 0 {
		_ = db.Pool.QueryRow(ctx,
			`SELECT target_months FROM emergency_fund WHERE family_id = $1`,
			sess.FamilyID,
		).Scan(&targetMonths)
		if targetMonths <= 0 {
			targetMonths = 6
		}
	}

	// Hitung monthly_expense_avg dari rata-rata pengeluaran 3 bulan terakhir
	var monthlyAvg float64
	err = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(monthly_total), 0)
		FROM (
			SELECT TO_CHAR(date, 'YYYY-MM') AS mon, SUM(amount) AS monthly_total
			FROM transactions
			WHERE family_id = $1
			  AND type = 'expense'
			  AND date >= NOW() - INTERVAL '3 months'
			GROUP BY mon
		) sub`,
		sess.FamilyID,
	).Scan(&monthlyAvg)
	if err != nil {
		monthlyAvg = 0
	}

	targetBalance, monthsCovered := helpers.RecalcEmergencyFundCustom(req.CurrentBalance, monthlyAvg, int(targetMonths))

	ef := &finance.EmergencyFund{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO emergency_fund (family_id, current_balance, monthly_expense_avg, target_balance, months_covered, target_months, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (family_id) DO UPDATE
		SET current_balance     = EXCLUDED.current_balance,
		    monthly_expense_avg = EXCLUDED.monthly_expense_avg,
		    target_balance      = EXCLUDED.target_balance,
		    months_covered      = EXCLUDED.months_covered,
		    target_months       = EXCLUDED.target_months,
		    updated_at          = NOW()
		RETURNING id, family_id, current_balance::float8, monthly_expense_avg::float8,
		          target_balance::float8, months_covered::float8, updated_at::text, target_months`,
		sess.FamilyID, req.CurrentBalance, monthlyAvg, targetBalance, monthsCovered, targetMonths,
	).Scan(&ef.Id, &ef.FamilyId, &ef.CurrentBalance, &ef.MonthlyExpenseAvg,
		&ef.TargetBalance, &ef.MonthsCovered, &ef.UpdatedAt, &ef.TargetMonths)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal update dana darurat: "+err.Error())
	}

	return &finance.EmergencyFundResponse{Success: true, Message: "Dana darurat berhasil diupdate", EmergencyFund: ef}, nil
}

// DepositEmergencyFund — catat setoran (+) atau penarikan (-) dan update saldo.
func (s *BudgetServer) DepositEmergencyFund(ctx context.Context, req *finance.DepositEmergencyFundRequest) (*finance.EmergencyFundResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.Amount == 0 {
		return nil, status.Error(codes.InvalidArgument, "amount tidak boleh nol")
	}
	if req.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "date wajib diisi")
	}

	// Ambil saldo & target_months saat ini
	var curBal float64
	var targetMonths int32
	err = db.Pool.QueryRow(ctx,
		`SELECT current_balance, target_months FROM emergency_fund WHERE family_id = $1`,
		sess.FamilyID,
	).Scan(&curBal, &targetMonths)
	if err != nil {
		return nil, status.Error(codes.NotFound, "data dana darurat belum ada, buat terlebih dahulu")
	}

	newBal := curBal + req.Amount
	if newBal < 0 {
		return nil, status.Error(codes.InvalidArgument, "saldo tidak cukup untuk penarikan ini")
	}

	// Catat deposit
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO emergency_fund_deposits (family_id, amount, note, date) VALUES ($1, $2, $3, $4)`,
		sess.FamilyID, req.Amount, req.Note, req.Date,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal catat transaksi dana darurat: "+err.Error())
	}

	// Hitung ulang avg pengeluaran
	var monthlyAvg float64
	_ = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(monthly_total), 0)
		FROM (
			SELECT TO_CHAR(date, 'YYYY-MM') AS mon, SUM(amount) AS monthly_total
			FROM transactions
			WHERE family_id = $1
			  AND type = 'expense'
			  AND date >= NOW() - INTERVAL '3 months'
			GROUP BY mon
		) sub`, sess.FamilyID,
	).Scan(&monthlyAvg)

	targetBalance, monthsCovered := helpers.RecalcEmergencyFundCustom(newBal, monthlyAvg, int(targetMonths))

	ef := &finance.EmergencyFund{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE emergency_fund
		SET current_balance = $1, monthly_expense_avg = $2,
		    target_balance = $3, months_covered = $4, updated_at = NOW()
		WHERE family_id = $5
		RETURNING id, family_id, current_balance::float8, monthly_expense_avg::float8,
		          target_balance::float8, months_covered::float8, updated_at::text, target_months`,
		newBal, monthlyAvg, targetBalance, monthsCovered, sess.FamilyID,
	).Scan(&ef.Id, &ef.FamilyId, &ef.CurrentBalance, &ef.MonthlyExpenseAvg,
		&ef.TargetBalance, &ef.MonthsCovered, &ef.UpdatedAt, &ef.TargetMonths)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal update saldo dana darurat: "+err.Error())
	}

	msg := "Setoran berhasil dicatat"
	if req.Amount < 0 {
		msg = "Penarikan berhasil dicatat"
	}
	return &finance.EmergencyFundResponse{Success: true, Message: msg, EmergencyFund: ef}, nil
}

// ListEmergencyFundHistory — riwayat setoran/penarikan terbaru.
func (s *BudgetServer) ListEmergencyFundHistory(ctx context.Context, req *finance.ListEmergencyFundHistoryRequest) (*finance.EmergencyFundHistoryResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, amount::float8, note, date::text, created_at::text
		FROM emergency_fund_deposits
		WHERE family_id = $1
		ORDER BY date DESC, created_at DESC
		LIMIT $2`,
		sess.FamilyID, limit,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query riwayat dana darurat")
	}
	defer rows.Close()

	var history []*finance.EmergencyFundDeposit
	for rows.Next() {
		d := &finance.EmergencyFundDeposit{}
		if err := rows.Scan(&d.Id, &d.FamilyId, &d.Amount, &d.Note, &d.Date, &d.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan riwayat dana darurat")
		}
		history = append(history, d)
	}
	return &finance.EmergencyFundHistoryResponse{Success: true, History: history}, nil
}

// scanBudgetRows adalah helper untuk ListBudgets agar bisa digunakan ulang.
func scanBudgetRows(rows pgx.Rows) ([]*finance.Budget, error) {
	var buds []*finance.Budget
	for rows.Next() {
		b := &finance.Budget{}
		if err := rows.Scan(&b.Id, &b.FamilyId, &b.CategoryName, &b.Month, &b.Amount, &b.CreatedAt); err != nil {
			return nil, err
		}
		buds = append(buds, b)
	}
	return buds, nil
}

// ─────────────────────────────────────────────────────────────
// Recurring Budgets
// ─────────────────────────────────────────────────────────────

// SetRecurringBudget — upsert template anggaran berulang per kategori.
func (s *BudgetServer) SetRecurringBudget(ctx context.Context, req *finance.SetRecurringBudgetRequest) (*finance.RecurringBudgetResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.CategoryName == "" {
		return nil, status.Error(codes.InvalidArgument, "category_name wajib diisi")
	}
	if req.Amount < 0 {
		return nil, status.Error(codes.InvalidArgument, "amount tidak boleh negatif")
	}

	rb := &finance.RecurringBudget{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO recurring_budgets (family_id, category_name, amount)
		VALUES ($1, $2, $3)
		ON CONFLICT (family_id, category_name)
		DO UPDATE SET amount = EXCLUDED.amount
		RETURNING id, family_id, category_name, amount::float8, created_at::text`,
		sess.FamilyID, req.CategoryName, req.Amount,
	).Scan(&rb.Id, &rb.FamilyId, &rb.CategoryName, &rb.Amount, &rb.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal set anggaran berulang: "+err.Error())
	}

	return &finance.RecurringBudgetResponse{
		Success:         true,
		Message:         "Anggaran berulang berhasil disimpan",
		RecurringBudget: rb,
	}, nil
}

// ListRecurringBudgets — daftar semua template anggaran berulang keluarga.
func (s *BudgetServer) ListRecurringBudgets(ctx context.Context, req *finance.ListRecurringBudgetsRequest) (*finance.RecurringBudgetListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, category_name, amount::float8, created_at::text
		FROM recurring_budgets
		WHERE family_id = $1
		ORDER BY category_name`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query anggaran berulang")
	}
	defer rows.Close()

	var rbs []*finance.RecurringBudget
	for rows.Next() {
		rb := &finance.RecurringBudget{}
		if err := rows.Scan(&rb.Id, &rb.FamilyId, &rb.CategoryName, &rb.Amount, &rb.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan anggaran berulang")
		}
		rbs = append(rbs, rb)
	}
	return &finance.RecurringBudgetListResponse{Success: true, RecurringBudgets: rbs}, nil
}

// DeleteRecurringBudget — hapus template anggaran berulang.
func (s *BudgetServer) DeleteRecurringBudget(ctx context.Context, req *finance.DeleteRecurringBudgetRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.RecurringBudgetId == "" {
		return nil, status.Error(codes.InvalidArgument, "recurring_budget_id wajib diisi")
	}

	res, err := db.Pool.Exec(ctx,
		`DELETE FROM recurring_budgets WHERE id = $1 AND family_id = $2`,
		req.RecurringBudgetId, sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus anggaran berulang")
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "anggaran berulang tidak ditemukan")
	}

	return &finance.StatusResponse{Success: true, Message: "Anggaran berulang berhasil dihapus"}, nil
}

// ApplyRecurringBudgets — terapkan semua template ke bulan tertentu (insert ke tabel budgets).
// Anggaran yang sudah ada di bulan tersebut tidak ditimpa (ON CONFLICT DO NOTHING).
func (s *BudgetServer) ApplyRecurringBudgets(ctx context.Context, req *finance.ApplyRecurringBudgetsRequest) (*finance.ApplyRecurringBudgetsResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.Month == "" {
		return nil, status.Error(codes.InvalidArgument, "month wajib diisi")
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT category_name, amount FROM recurring_budgets WHERE family_id = $1`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal ambil anggaran berulang")
	}
	defer rows.Close()

	type entry struct {
		cat    string
		amount float64
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.cat, &e.amount); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan anggaran berulang")
		}
		entries = append(entries, e)
	}
	rows.Close()

	if len(entries) == 0 {
		return &finance.ApplyRecurringBudgetsResponse{
			Success:      true,
			Message:      "Tidak ada template anggaran berulang",
			AppliedCount: 0,
		}, nil
	}

	var applied int32
	for _, e := range entries {
		tag, err := db.Pool.Exec(ctx, `
			INSERT INTO budgets (family_id, category_name, month, amount)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (family_id, category_name, month) DO NOTHING`,
			sess.FamilyID, e.cat, req.Month, e.amount,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "gagal terapkan anggaran berulang: "+err.Error())
		}
		if tag.RowsAffected() > 0 {
			applied++
		}
	}

	return &finance.ApplyRecurringBudgetsResponse{
		Success:      true,
		Message:      "Anggaran berulang berhasil diterapkan",
		AppliedCount: applied,
	}, nil
}
