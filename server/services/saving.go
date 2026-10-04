package services

import (
	"context"
	"fmt"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/helpers"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SavingServer mengimplementasikan finance.SavingServiceServer.
type SavingServer struct {
	finance.UnimplementedSavingServiceServer
}

// savingCols adalah kolom SELECT untuk tabel savings.
const savingCols = `id, family_id, name, target_amount::float8, current_balance::float8,
                    target_date::text, monthly_required::float8, progress_pct::float8, created_at::text`

func (s *SavingServer) CreateSaving(ctx context.Context, req *finance.CreateSavingRequest) (*finance.SavingResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.Name == "" || req.TargetDate == "" {
		return nil, status.Error(codes.InvalidArgument, "name dan target_date wajib diisi")
	}
	if req.TargetAmount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "target_amount harus lebih dari 0")
	}
	if req.InitialBalance < 0 {
		return nil, status.Error(codes.InvalidArgument, "initial_balance tidak boleh negatif")
	}

	monthlyRequired, progressPct := helpers.RecalcSavingFields(req.InitialBalance, req.TargetAmount, req.TargetDate)

	sv := &finance.Saving{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO savings
		  (family_id, name, target_amount, current_balance, target_date, monthly_required, progress_pct)
		VALUES ($1, $2, $3, $4, $5::date, $6, $7)
		RETURNING `+savingCols,
		sess.FamilyID, req.Name, req.TargetAmount, req.InitialBalance,
		req.TargetDate, monthlyRequired, progressPct,
	).Scan(&sv.Id, &sv.FamilyId, &sv.Name, &sv.TargetAmount, &sv.CurrentBalance,
		&sv.TargetDate, &sv.MonthlyRequired, &sv.ProgressPct, &sv.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal buat tabungan: "+err.Error())
	}

	return &finance.SavingResponse{Success: true, Message: "Tujuan tabungan berhasil dibuat", Saving: sv}, nil
}

func (s *SavingServer) ListSavings(ctx context.Context, req *finance.ListSavingsRequest) (*finance.SavingListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT `+savingCols+`
		FROM savings WHERE family_id = $1 ORDER BY created_at DESC`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query tabungan")
	}
	defer rows.Close()

	var savings []*finance.Saving
	for rows.Next() {
		sv := &finance.Saving{}
		if err := rows.Scan(&sv.Id, &sv.FamilyId, &sv.Name, &sv.TargetAmount, &sv.CurrentBalance,
			&sv.TargetDate, &sv.MonthlyRequired, &sv.ProgressPct, &sv.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan tabungan")
		}
		savings = append(savings, sv)
	}

	return &finance.SavingListResponse{Success: true, Savings: savings}, nil
}

func (s *SavingServer) UpdateSaving(ctx context.Context, req *finance.UpdateSavingRequest) (*finance.SavingResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.SavingId == "" {
		return nil, status.Error(codes.InvalidArgument, "saving_id wajib diisi")
	}
	if req.TargetAmount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "target_amount harus lebih dari 0")
	}

	// Ambil current_balance dulu untuk recalculate
	var currentBalance float64
	err = db.Pool.QueryRow(ctx,
		`SELECT current_balance::float8 FROM savings WHERE id = $1 AND family_id = $2`,
		req.SavingId, sess.FamilyID,
	).Scan(&currentBalance)
	if err != nil {
		return nil, status.Error(codes.NotFound, "tabungan tidak ditemukan")
	}

	monthlyRequired, progressPct := helpers.RecalcSavingFields(currentBalance, req.TargetAmount, req.TargetDate)

	sv := &finance.Saving{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE savings
		SET name = $1, target_amount = $2, target_date = $3::date,
		    monthly_required = $4, progress_pct = $5
		WHERE id = $6 AND family_id = $7
		RETURNING `+savingCols,
		req.Name, req.TargetAmount, req.TargetDate,
		monthlyRequired, progressPct,
		req.SavingId, sess.FamilyID,
	).Scan(&sv.Id, &sv.FamilyId, &sv.Name, &sv.TargetAmount, &sv.CurrentBalance,
		&sv.TargetDate, &sv.MonthlyRequired, &sv.ProgressPct, &sv.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.NotFound, "tabungan tidak ditemukan")
	}

	return &finance.SavingResponse{Success: true, Message: "Tabungan berhasil diupdate", Saving: sv}, nil
}

func (s *SavingServer) DepositSaving(ctx context.Context, req *finance.DepositSavingRequest) (*finance.SavingResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.SavingId == "" || req.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "saving_id dan date wajib diisi")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount harus lebih dari 0")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// Ambil data tabungan dan lock
	var targetAmount float64
	var targetDate string
	err = tx.QueryRow(ctx, `
		SELECT target_amount::float8, target_date::text
		FROM savings WHERE id = $1 AND family_id = $2 FOR UPDATE`,
		req.SavingId, sess.FamilyID,
	).Scan(&targetAmount, &targetDate)
	if err != nil {
		return nil, status.Error(codes.NotFound, "tabungan tidak ditemukan")
	}

	// Update balance
	var newBalance float64
	err = tx.QueryRow(ctx,
		`UPDATE savings SET current_balance = current_balance + $1 WHERE id = $2
		 RETURNING current_balance::float8`,
		req.Amount, req.SavingId,
	).Scan(&newBalance)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal update saldo tabungan")
	}

	// Recalculate dan simpan computed fields
	monthlyRequired, progressPct := helpers.RecalcSavingFields(newBalance, targetAmount, targetDate)
	sv := &finance.Saving{}
	err = tx.QueryRow(ctx, `
		UPDATE savings SET monthly_required = $1, progress_pct = $2 WHERE id = $3
		RETURNING `+savingCols,
		monthlyRequired, progressPct, req.SavingId,
	).Scan(&sv.Id, &sv.FamilyId, &sv.Name, &sv.TargetAmount, &sv.CurrentBalance,
		&sv.TargetDate, &sv.MonthlyRequired, &sv.ProgressPct, &sv.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal update computed fields tabungan")
	}

	// Insert deposit record dan ambil ID-nya
	var depositID string
	err = tx.QueryRow(ctx, `
		INSERT INTO saving_deposits (saving_id, amount, date, note)
		VALUES ($1, $2, $3::date, $4)
		RETURNING id`,
		req.SavingId, req.Amount, req.Date, req.Note,
	).Scan(&depositID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert setoran")
	}

	// Insert transaksi pengeluaran otomatis yang terhubung ke deposit ini
	note := req.Note
	if note == "" {
		note = fmt.Sprintf("Setoran tabungan: %s", sv.Name)
	}
	trx := &finance.Transaction{}
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions
		  (family_id, created_by, type, amount, category_name, member, date, note, saving_deposit_id)
		VALUES ($1, $2, 'expense', $3, $4, $5, $6::date, $7, $8)
		RETURNING id, family_id, created_by, type, amount::float8,
		          category_name, member, date::text, note,
		          COALESCE(debt_payment_id::text,''),
		          created_at::text,
		          COALESCE(saving_deposit_id::text,'')`,
		sess.FamilyID, sess.UserID, req.Amount, sv.Name,
		req.Member, req.Date, note, depositID,
	).Scan(&trx.Id, &trx.FamilyId, &trx.CreatedBy, &trx.Type, &trx.Amount,
		&trx.CategoryName, &trx.Member, &trx.Date, &trx.Note,
		&trx.DebtPaymentId, &trx.CreatedAt, &trx.SavingDepositId)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal insert transaksi setoran: "+err.Error())
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit")
	}

	return &finance.SavingResponse{Success: true, Message: "Setoran berhasil dicatat", Saving: sv, Transaction: trx}, nil
}

func (s *SavingServer) DeleteSaving(ctx context.Context, req *finance.DeleteSavingRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	res, err := db.Pool.Exec(ctx,
		`DELETE FROM savings WHERE id = $1 AND family_id = $2`,
		req.SavingId, sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus tabungan")
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "tabungan tidak ditemukan")
	}

	return &finance.StatusResponse{Success: true, Message: "Tabungan berhasil dihapus"}, nil
}
