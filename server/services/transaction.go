package services

import (
	"context"
	"strings"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/helpers"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TransactionServer mengimplementasikan finance.TransactionServiceServer.
type TransactionServer struct {
	finance.UnimplementedTransactionServiceServer
}

// txSelect adalah kolom SELECT standar untuk transaction + wallet_name (LEFT JOIN)
const txSelect = `
	t.id, t.family_id, t.created_by, t.type, t.amount::float8,
	t.category_name, t.member, t.date::text, t.note,
	COALESCE(t.debt_payment_id::text,''), t.created_at::text,
	COALESCE(t.saving_deposit_id::text,''),
	COALESCE(t.wallet_id::text,''),
	COALESCE(w.name,'')`

func scanTx(t *finance.Transaction, row interface{ Scan(...any) error }) error {
	return row.Scan(&t.Id, &t.FamilyId, &t.CreatedBy, &t.Type, &t.Amount,
		&t.CategoryName, &t.Member, &t.Date, &t.Note, &t.DebtPaymentId, &t.CreatedAt,
		&t.SavingDepositId, &t.WalletId, &t.WalletName)
}

// walletIDArg konversi string UUID ke *string (nil jika kosong, agar pg menerima NULL)
func walletIDArg(id string) interface{} {
	if id == "" {
		return nil
	}
	return id
}

func (s *TransactionServer) CreateTransaction(ctx context.Context, req *finance.CreateTransactionRequest) (*finance.TransactionResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount harus lebih dari 0")
	}
	if req.Type != "income" && req.Type != "expense" {
		return nil, status.Error(codes.InvalidArgument, "type harus income atau expense")
	}
	if req.CategoryName == "" || req.Member == "" || req.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "category_name, member, dan date wajib diisi")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi DB")
	}
	defer tx.Rollback(ctx)

	// Jika ada wallet_id, update saldo dompet
	if req.WalletId != "" {
		delta := req.Amount
		if req.Type == "expense" {
			delta = -req.Amount
		}
		res, err := tx.Exec(ctx,
			`UPDATE wallets SET balance = balance + $1 WHERE id = $2 AND family_id = $3`,
			delta, req.WalletId, sess.FamilyID,
		)
		if err != nil || res.RowsAffected() == 0 {
			return nil, status.Error(codes.NotFound, "dompet tidak ditemukan")
		}
	}

	t := &finance.Transaction{}
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions
		  (family_id, created_by, type, amount, category_name, member, date, note, wallet_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9::uuid)
		RETURNING id, family_id, created_by, type, amount::float8,
		          category_name, member, date::text, note,
		          COALESCE(debt_payment_id::text,''), created_at::text,
		          COALESCE(saving_deposit_id::text,''),
		          COALESCE(wallet_id::text,''), ''`,
		sess.FamilyID, sess.UserID, req.Type, req.Amount,
		req.CategoryName, req.Member, req.Date, req.Note, walletIDArg(req.WalletId),
	).Scan(&t.Id, &t.FamilyId, &t.CreatedBy, &t.Type, &t.Amount,
		&t.CategoryName, &t.Member, &t.Date, &t.Note, &t.DebtPaymentId, &t.CreatedAt,
		&t.SavingDepositId, &t.WalletId, &t.WalletName)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal buat transaksi: "+err.Error())
	}

	// Ambil nama wallet setelah insert
	if t.WalletId != "" {
		tx.QueryRow(ctx, `SELECT name FROM wallets WHERE id = $1`, t.WalletId).Scan(&t.WalletName)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit transaksi")
	}

	return &finance.TransactionResponse{Success: true, Message: "Transaksi berhasil dibuat", Transaction: t}, nil
}

func (s *TransactionServer) ListTransactions(ctx context.Context, req *finance.ListTransactionsRequest) (*finance.TransactionListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if req.Month != "" {
		rows, err = db.Pool.Query(ctx, `
			SELECT `+txSelect+`
			FROM transactions t
			LEFT JOIN wallets w ON w.id = t.wallet_id
			WHERE t.family_id = $1
			  AND TO_CHAR(t.date, 'YYYY-MM') = $2
			ORDER BY t.date DESC, t.created_at DESC`,
			sess.FamilyID, req.Month,
		)
	} else {
		rows, err = db.Pool.Query(ctx, `
			SELECT `+txSelect+`
			FROM transactions t
			LEFT JOIN wallets w ON w.id = t.wallet_id
			WHERE t.family_id = $1
			ORDER BY t.date DESC, t.created_at DESC`,
			sess.FamilyID,
		)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query transaksi")
	}
	defer rows.Close()

	var txs []*finance.Transaction
	for rows.Next() {
		t := &finance.Transaction{}
		if err := rows.Scan(&t.Id, &t.FamilyId, &t.CreatedBy, &t.Type, &t.Amount,
			&t.CategoryName, &t.Member, &t.Date, &t.Note, &t.DebtPaymentId, &t.CreatedAt,
			&t.SavingDepositId, &t.WalletId, &t.WalletName); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan transaksi")
		}
		txs = append(txs, t)
	}

	return &finance.TransactionListResponse{Success: true, Transactions: txs}, nil
}

func (s *TransactionServer) UpdateTransaction(ctx context.Context, req *finance.UpdateTransactionRequest) (*finance.TransactionResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.TransactionId == "" {
		return nil, status.Error(codes.InvalidArgument, "transaction_id wajib diisi")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount harus lebih dari 0")
	}
	if req.Type != "income" && req.Type != "expense" {
		return nil, status.Error(codes.InvalidArgument, "type harus income atau expense")
	}

	dbTx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi DB")
	}
	defer dbTx.Rollback(ctx)

	// Ambil transaksi lama untuk revert saldo dompet lama
	var oldWalletID, oldType string
	var oldAmount float64
	dbTx.QueryRow(ctx,
		`SELECT COALESCE(wallet_id::text,''), type, amount::float8 FROM transactions WHERE id = $1 AND family_id = $2`,
		req.TransactionId, sess.FamilyID,
	).Scan(&oldWalletID, &oldType, &oldAmount)

	// Revert saldo dompet lama
	if oldWalletID != "" {
		revert := oldAmount
		if oldType == "expense" {
			revert = -oldAmount
		}
		dbTx.Exec(ctx,
			`UPDATE wallets SET balance = balance - $1 WHERE id = $2 AND family_id = $3`,
			revert, oldWalletID, sess.FamilyID,
		)
	}

	// Apply saldo dompet baru
	if req.WalletId != "" {
		delta := req.Amount
		if req.Type == "expense" {
			delta = -req.Amount
		}
		res, err := dbTx.Exec(ctx,
			`UPDATE wallets SET balance = balance + $1 WHERE id = $2 AND family_id = $3`,
			delta, req.WalletId, sess.FamilyID,
		)
		if err != nil || res.RowsAffected() == 0 {
			return nil, status.Error(codes.NotFound, "dompet tidak ditemukan")
		}
	}

	t := &finance.Transaction{}
	err = dbTx.QueryRow(ctx, `
		UPDATE transactions
		SET type = $1, amount = $2, category_name = $3,
		    member = $4, date = $5::date, note = $6, wallet_id = $7::uuid
		WHERE id = $8 AND family_id = $9
		RETURNING id, family_id, created_by, type, amount::float8,
		          category_name, member, date::text, note,
		          COALESCE(debt_payment_id::text,''), created_at::text,
		          COALESCE(saving_deposit_id::text,''),
		          COALESCE(wallet_id::text,''), ''`,
		req.Type, req.Amount, req.CategoryName,
		req.Member, req.Date, req.Note, walletIDArg(req.WalletId),
		req.TransactionId, sess.FamilyID,
	).Scan(&t.Id, &t.FamilyId, &t.CreatedBy, &t.Type, &t.Amount,
		&t.CategoryName, &t.Member, &t.Date, &t.Note, &t.DebtPaymentId, &t.CreatedAt,
		&t.SavingDepositId, &t.WalletId, &t.WalletName)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, "transaksi tidak ditemukan")
		}
		return nil, status.Error(codes.Internal, "gagal update transaksi: "+err.Error())
	}

	if t.WalletId != "" {
		dbTx.QueryRow(ctx, `SELECT name FROM wallets WHERE id = $1`, t.WalletId).Scan(&t.WalletName)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit update transaksi")
	}

	return &finance.TransactionResponse{Success: true, Message: "Transaksi berhasil diupdate", Transaction: t}, nil
}

func (s *TransactionServer) DeleteTransaction(ctx context.Context, req *finance.DeleteTransactionRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	dbTx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi DB")
	}
	defer dbTx.Rollback(ctx)

	// Ambil info transaksi dulu untuk revert saldo dompet
	var walletID, txType string
	var amount float64
	dbTx.QueryRow(ctx,
		`SELECT COALESCE(wallet_id::text,''), type, amount::float8 FROM transactions WHERE id = $1 AND family_id = $2`,
		req.TransactionId, sess.FamilyID,
	).Scan(&walletID, &txType, &amount)

	res, err := dbTx.Exec(ctx,
		`DELETE FROM transactions WHERE id = $1 AND family_id = $2`,
		req.TransactionId, sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus transaksi")
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "transaksi tidak ditemukan")
	}

	// Revert saldo dompet
	if walletID != "" {
		revert := amount
		if txType == "expense" {
			revert = -amount
		}
		dbTx.Exec(ctx,
			`UPDATE wallets SET balance = balance - $1 WHERE id = $2 AND family_id = $3`,
			revert, walletID, sess.FamilyID,
		)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit hapus transaksi")
	}

	return &finance.StatusResponse{Success: true, Message: "Transaksi berhasil dihapus"}, nil
}
