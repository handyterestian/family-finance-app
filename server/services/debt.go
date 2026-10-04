package services

import (
	"context"
	"fmt"
	"time"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/helpers"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DebtServer mengimplementasikan finance.DebtServiceServer.
type DebtServer struct {
	finance.UnimplementedDebtServiceServer
}

// scanDebt membaca satu baris debt dari query result.
func scanDebt(scanner interface {
	Scan(dest ...interface{}) error
}) (*finance.Debt, error) {
	d := &finance.Debt{}
	return d, scanner.Scan(
		&d.Id, &d.FamilyId, &d.Name,
		&d.TotalAmount, &d.RemainingAmount, &d.MonthlyPayment,
		&d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt,
	)
}

func (s *DebtServer) CreateDebt(ctx context.Context, req *finance.CreateDebtRequest) (*finance.DebtResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.Name == "" || req.DueDate == "" {
		return nil, status.Error(codes.InvalidArgument, "name dan due_date wajib diisi")
	}
	if req.TotalAmount <= 0 || req.MonthlyPayment <= 0 {
		return nil, status.Error(codes.InvalidArgument, "total_amount dan monthly_payment harus lebih dari 0")
	}

	daysUntilDue := helpers.RecalcDaysUntilDue(req.DueDate)

	d := &finance.Debt{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO debts
		  (family_id, name, total_amount, remaining_amount, monthly_payment, due_date, days_until_due)
		VALUES ($1, $2, $3, $3, $4, $5::date, $6)
		RETURNING id, family_id, name, total_amount::float8, remaining_amount::float8,
		          monthly_payment::float8, due_date::text, is_paid, days_until_due, created_at::text`,
		sess.FamilyID, req.Name, req.TotalAmount, req.MonthlyPayment, req.DueDate, daysUntilDue,
	).Scan(&d.Id, &d.FamilyId, &d.Name, &d.TotalAmount, &d.RemainingAmount,
		&d.MonthlyPayment, &d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal buat hutang: "+err.Error())
	}

	return &finance.DebtResponse{Success: true, Message: "Hutang berhasil dicatat", Debt: d}, nil
}

func (s *DebtServer) ListDebts(ctx context.Context, req *finance.ListDebtsRequest) (*finance.DebtListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, name, total_amount::float8, remaining_amount::float8,
		       monthly_payment::float8, due_date::text, is_paid, days_until_due, created_at::text
		FROM debts
		WHERE family_id = $1
		ORDER BY is_paid ASC, due_date ASC`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query hutang")
	}
	defer rows.Close()

	var debts []*finance.Debt
	for rows.Next() {
		d := &finance.Debt{}
		if err := rows.Scan(&d.Id, &d.FamilyId, &d.Name, &d.TotalAmount, &d.RemainingAmount,
			&d.MonthlyPayment, &d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan hutang")
		}
		debts = append(debts, d)
	}

	return &finance.DebtListResponse{Success: true, Debts: debts}, nil
}

func (s *DebtServer) UpdateDebt(ctx context.Context, req *finance.UpdateDebtRequest) (*finance.DebtResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.DebtId == "" {
		return nil, status.Error(codes.InvalidArgument, "debt_id wajib diisi")
	}
	if req.TotalAmount <= 0 || req.MonthlyPayment <= 0 {
		return nil, status.Error(codes.InvalidArgument, "total_amount dan monthly_payment harus lebih dari 0")
	}

	daysUntilDue := helpers.RecalcDaysUntilDue(req.DueDate)

	d := &finance.Debt{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE debts
		SET name = $1, total_amount = $2, monthly_payment = $3,
		    due_date = $4::date, days_until_due = $5
		WHERE id = $6 AND family_id = $7
		RETURNING id, family_id, name, total_amount::float8, remaining_amount::float8,
		          monthly_payment::float8, due_date::text, is_paid, days_until_due, created_at::text`,
		req.Name, req.TotalAmount, req.MonthlyPayment,
		req.DueDate, daysUntilDue, req.DebtId, sess.FamilyID,
	).Scan(&d.Id, &d.FamilyId, &d.Name, &d.TotalAmount, &d.RemainingAmount,
		&d.MonthlyPayment, &d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.NotFound, "hutang tidak ditemukan")
	}

	return &finance.DebtResponse{Success: true, Message: "Hutang berhasil diupdate", Debt: d}, nil
}

func (s *DebtServer) PayDebt(ctx context.Context, req *finance.PayDebtRequest) (*finance.PayDebtResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.DebtId == "" || req.PaidBy == "" || req.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "debt_id, paid_by, dan date wajib diisi")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount harus lebih dari 0")
	}

	payDate := req.Date
	if payDate == "" {
		payDate = time.Now().Format("2006-01-02")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// Ambil data hutang
	d := &finance.Debt{}
	err = tx.QueryRow(ctx, `
		SELECT id, family_id, name, total_amount::float8, remaining_amount::float8,
		       monthly_payment::float8, due_date::text, is_paid, days_until_due, created_at::text
		FROM debts WHERE id = $1 AND family_id = $2 FOR UPDATE`,
		req.DebtId, sess.FamilyID,
	).Scan(&d.Id, &d.FamilyId, &d.Name, &d.TotalAmount, &d.RemainingAmount,
		&d.MonthlyPayment, &d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.NotFound, "hutang tidak ditemukan")
	}
	if d.IsPaid {
		return nil, status.Error(codes.FailedPrecondition, "hutang sudah lunas")
	}

	// Hitung sisa setelah pembayaran
	newRemaining := d.RemainingAmount - req.Amount
	if newRemaining < 0 {
		newRemaining = 0
	}
	isPaid := newRemaining == 0
	daysUntilDue := helpers.RecalcDaysUntilDue(d.DueDate)

	// Update hutang
	err = tx.QueryRow(ctx, `
		UPDATE debts
		SET remaining_amount = $1, is_paid = $2, days_until_due = $3
		WHERE id = $4
		RETURNING id, family_id, name, total_amount::float8, remaining_amount::float8,
		          monthly_payment::float8, due_date::text, is_paid, days_until_due, created_at::text`,
		newRemaining, isPaid, daysUntilDue, d.Id,
	).Scan(&d.Id, &d.FamilyId, &d.Name, &d.TotalAmount, &d.RemainingAmount,
		&d.MonthlyPayment, &d.DueDate, &d.IsPaid, &d.DaysUntilDue, &d.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal update hutang")
	}

	// Insert debt_payment
	pmt := &finance.DebtPayment{}
	err = tx.QueryRow(ctx, `
		INSERT INTO debt_payments (debt_id, paid_by, amount, date)
		VALUES ($1, $2, $3, $4::date)
		RETURNING id, debt_id, paid_by, amount::float8, date::text, created_at::text`,
		d.Id, req.PaidBy, req.Amount, payDate,
	).Scan(&pmt.Id, &pmt.DebtId, &pmt.PaidBy, &pmt.Amount, &pmt.Date, &pmt.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert debt_payment")
	}

	// Insert transaksi pengeluaran otomatis
	trx := &finance.Transaction{}
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions
		  (family_id, created_by, type, amount, category_name, member, date, note, debt_payment_id)
		VALUES ($1, $2, 'expense', $3, 'Cicilan Hutang', $4, $5::date, $6, $7)
		RETURNING id, family_id, created_by, type, amount::float8,
		          category_name, member, date::text, note,
		          COALESCE(debt_payment_id::text,''), created_at::text,
		          COALESCE(saving_deposit_id::text,'')`,
		sess.FamilyID, sess.UserID, req.Amount, req.PaidBy, payDate,
		fmt.Sprintf("Bayar cicilan: %s", d.Name), pmt.Id,
	).Scan(&trx.Id, &trx.FamilyId, &trx.CreatedBy, &trx.Type, &trx.Amount,
		&trx.CategoryName, &trx.Member, &trx.Date, &trx.Note, &trx.DebtPaymentId, &trx.CreatedAt, &trx.SavingDepositId)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert transaksi cicilan")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit")
	}

	return &finance.PayDebtResponse{
		Success:     true,
		Message:     fmt.Sprintf("Cicilan berhasil dibayar. Sisa: %.0f", newRemaining),
		Debt:        d,
		Payment:     pmt,
		Transaction: trx,
	}, nil
}

func (s *DebtServer) DeleteDebt(ctx context.Context, req *finance.DeleteDebtRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	res, err := db.Pool.Exec(ctx,
		`DELETE FROM debts WHERE id = $1 AND family_id = $2`,
		req.DebtId, sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus hutang")
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "hutang tidak ditemukan")
	}

	return &finance.StatusResponse{Success: true, Message: "Hutang berhasil dihapus"}, nil
}

func (s *DebtServer) ListDebtPayments(ctx context.Context, req *finance.ListDebtPaymentsRequest) (*finance.DebtPaymentListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.DebtId == "" {
		return nil, status.Error(codes.InvalidArgument, "debt_id wajib diisi")
	}

	// Pastikan hutang milik family yang sama
	var familyID string
	err = db.Pool.QueryRow(ctx,
		`SELECT family_id FROM debts WHERE id = $1`, req.DebtId,
	).Scan(&familyID)
	if err != nil || familyID != sess.FamilyID {
		return nil, status.Error(codes.NotFound, "hutang tidak ditemukan")
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, debt_id, paid_by, amount::float8, date::text, created_at::text
		FROM debt_payments
		WHERE debt_id = $1
		ORDER BY date DESC, created_at DESC`,
		req.DebtId,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query riwayat pembayaran")
	}
	defer rows.Close()

	var payments []*finance.DebtPayment
	for rows.Next() {
		p := &finance.DebtPayment{}
		if err := rows.Scan(&p.Id, &p.DebtId, &p.PaidBy, &p.Amount, &p.Date, &p.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan riwayat pembayaran")
		}
		payments = append(payments, p)
	}

	return &finance.DebtPaymentListResponse{Success: true, Payments: payments}, nil
}
