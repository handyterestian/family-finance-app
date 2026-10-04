package services

import (
	"context"
	"strings"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/helpers"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WalletServer mengimplementasikan finance.WalletServiceServer.
type WalletServer struct {
	finance.UnimplementedWalletServiceServer
}

func scanWallet(w *finance.Wallet, row interface {
	Scan(...any) error
}) error {
	return row.Scan(&w.Id, &w.FamilyId, &w.Name, &w.Type, &w.Balance, &w.Color, &w.Note, &w.CreatedAt)
}

func (s *WalletServer) ListWallets(ctx context.Context, req *finance.ListWalletsRequest) (*finance.WalletListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, family_id, name, type, balance::float8, color, note, created_at::text
		FROM wallets WHERE family_id = $1 ORDER BY name`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query dompet")
	}
	defer rows.Close()

	var wallets []*finance.Wallet
	for rows.Next() {
		w := &finance.Wallet{}
		if err := rows.Scan(&w.Id, &w.FamilyId, &w.Name, &w.Type, &w.Balance, &w.Color, &w.Note, &w.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan dompet")
		}
		wallets = append(wallets, w)
	}
	return &finance.WalletListResponse{Success: true, Wallets: wallets}, nil
}

func (s *WalletServer) CreateWallet(ctx context.Context, req *finance.CreateWalletRequest) (*finance.WalletResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, status.Error(codes.InvalidArgument, "nama dompet wajib diisi")
	}
	validTypes := map[string]bool{"cash": true, "bank": true, "e-wallet": true, "investment": true, "other": true}
	if !validTypes[req.Type] {
		req.Type = "cash"
	}
	color := req.Color
	if color == "" {
		color = "#6366f1"
	}

	w := &finance.Wallet{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO wallets (family_id, name, type, balance, color, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, family_id, name, type, balance::float8, color, note, created_at::text`,
		sess.FamilyID, strings.TrimSpace(req.Name), req.Type, req.Balance, color, req.Note,
	).Scan(&w.Id, &w.FamilyId, &w.Name, &w.Type, &w.Balance, &w.Color, &w.Note, &w.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, status.Error(codes.AlreadyExists, "nama dompet sudah digunakan")
		}
		return nil, status.Error(codes.Internal, "gagal buat dompet: "+err.Error())
	}
	return &finance.WalletResponse{Success: true, Message: "Dompet berhasil dibuat", Wallet: w}, nil
}

func (s *WalletServer) UpdateWallet(ctx context.Context, req *finance.UpdateWalletRequest) (*finance.WalletResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.WalletId == "" {
		return nil, status.Error(codes.InvalidArgument, "wallet_id wajib diisi")
	}

	w := &finance.Wallet{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE wallets
		SET name = COALESCE(NULLIF($1,''), name),
		    type = COALESCE(NULLIF($2,''), type),
		    color = COALESCE(NULLIF($3,''), color),
		    note = $4
		WHERE id = $5 AND family_id = $6
		RETURNING id, family_id, name, type, balance::float8, color, note, created_at::text`,
		req.Name, req.Type, req.Color, req.Note, req.WalletId, sess.FamilyID,
	).Scan(&w.Id, &w.FamilyId, &w.Name, &w.Type, &w.Balance, &w.Color, &w.Note, &w.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, "dompet tidak ditemukan")
		}
		if strings.Contains(err.Error(), "unique") {
			return nil, status.Error(codes.AlreadyExists, "nama dompet sudah digunakan")
		}
		return nil, status.Error(codes.Internal, "gagal update dompet: "+err.Error())
	}
	return &finance.WalletResponse{Success: true, Message: "Dompet berhasil diupdate", Wallet: w}, nil
}

func (s *WalletServer) DeleteWallet(ctx context.Context, req *finance.DeleteWalletRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	res, err := db.Pool.Exec(ctx,
		`DELETE FROM wallets WHERE id = $1 AND family_id = $2`,
		req.WalletId, sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus dompet: "+err.Error())
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "dompet tidak ditemukan")
	}
	return &finance.StatusResponse{Success: true, Message: "Dompet berhasil dihapus"}, nil
}

func (s *WalletServer) TransferWallet(ctx context.Context, req *finance.TransferWalletRequest) (*finance.TransferWalletResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if req.FromWalletId == "" || req.ToWalletId == "" {
		return nil, status.Error(codes.InvalidArgument, "from_wallet_id dan to_wallet_id wajib diisi")
	}
	if req.FromWalletId == req.ToWalletId {
		return nil, status.Error(codes.InvalidArgument, "dompet asal dan tujuan tidak boleh sama")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount harus lebih dari 0")
	}
	if req.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "date wajib diisi")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// Kurangi saldo dompet asal
	fromW := &finance.Wallet{}
	err = tx.QueryRow(ctx, `
		UPDATE wallets SET balance = balance - $1
		WHERE id = $2 AND family_id = $3
		RETURNING id, family_id, name, type, balance::float8, color, note, created_at::text`,
		req.Amount, req.FromWalletId, sess.FamilyID,
	).Scan(&fromW.Id, &fromW.FamilyId, &fromW.Name, &fromW.Type, &fromW.Balance, &fromW.Color, &fromW.Note, &fromW.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, "dompet asal tidak ditemukan")
		}
		return nil, status.Error(codes.Internal, "gagal update dompet asal: "+err.Error())
	}

	// Tambah saldo dompet tujuan
	toW := &finance.Wallet{}
	err = tx.QueryRow(ctx, `
		UPDATE wallets SET balance = balance + $1
		WHERE id = $2 AND family_id = $3
		RETURNING id, family_id, name, type, balance::float8, color, note, created_at::text`,
		req.Amount, req.ToWalletId, sess.FamilyID,
	).Scan(&toW.Id, &toW.FamilyId, &toW.Name, &toW.Type, &toW.Balance, &toW.Color, &toW.Note, &toW.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, "dompet tujuan tidak ditemukan")
		}
		return nil, status.Error(codes.Internal, "gagal update dompet tujuan: "+err.Error())
	}

	// Catat transfer
	tr := &finance.WalletTransfer{}
	err = tx.QueryRow(ctx, `
		INSERT INTO wallet_transfers (family_id, from_wallet_id, to_wallet_id, amount, note, date)
		VALUES ($1, $2, $3, $4, $5, $6::date)
		RETURNING id, from_wallet_id, to_wallet_id, amount::float8, note, date::text, created_at::text`,
		sess.FamilyID, req.FromWalletId, req.ToWalletId, req.Amount, req.Note, req.Date,
	).Scan(&tr.Id, &tr.FromWalletId, &tr.ToWalletId, &tr.Amount, &tr.Note, &tr.Date, &tr.CreatedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal catat transfer: "+err.Error())
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit transfer")
	}

	return &finance.TransferWalletResponse{
		Success:    true,
		Message:    "Transfer berhasil",
		FromWallet: fromW,
		ToWallet:   toW,
		Transfer:   tr,
	}, nil
}
