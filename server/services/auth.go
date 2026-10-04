package services

import (
	"context"
	"strings"

	"family-finance/server/db"
	"family-finance/server/generated"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// defaultCategories adalah kategori yang di-insert saat Register.
var defaultCategories = []struct {
	name string
	typ  string
}{
	{"Gaji", "income"},
	{"Bonus", "income"},
	{"Investasi", "income"},
	{"Makanan", "expense"},
	{"Transport", "expense"},
	{"Tagihan", "expense"},
	{"Kesehatan", "expense"},
	{"Pendidikan", "expense"},
	{"Hiburan", "expense"},
	{"Belanja", "expense"},
	{"Lainnya", "both"},
	{"Cicilan Hutang", "expense"},
}

// AuthServer mengimplementasikan finance.AuthServiceServer.
type AuthServer struct {
	finance.UnimplementedAuthServiceServer
}

func (s *AuthServer) Register(ctx context.Context, req *finance.RegisterRequest) (*finance.StatusResponse, error) {
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Email) == "" ||
		req.Password == "" || strings.TrimSpace(req.FamilyName) == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, password, dan family_name wajib diisi")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hash password")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// Buat family
	var familyID string
	err = tx.QueryRow(ctx,
		`INSERT INTO families (name) VALUES ($1) RETURNING id`,
		strings.TrimSpace(req.FamilyName),
	).Scan(&familyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal buat family")
	}

	// Buat user (owner)
	_, err = tx.Exec(ctx,
		`INSERT INTO users (family_id, username, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, 'owner')`,
		familyID, strings.TrimSpace(req.Username),
		strings.ToLower(strings.TrimSpace(req.Email)), string(hash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, status.Error(codes.AlreadyExists, "username atau email sudah digunakan")
		}
		return nil, status.Error(codes.Internal, "gagal buat user")
	}

	// Insert kategori default
	for _, cat := range defaultCategories {
		_, err = tx.Exec(ctx,
			`INSERT INTO categories (family_id, name, type) VALUES ($1, $2, $3)
			 ON CONFLICT (family_id, name) DO NOTHING`,
			familyID, cat.name, cat.typ,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "gagal insert kategori default")
		}
	}

	// Buat baris emergency_fund kosong untuk family ini
	_, err = tx.Exec(ctx,
		`INSERT INTO emergency_fund (family_id) VALUES ($1)`,
		familyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal init emergency fund")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit")
	}

	return &finance.StatusResponse{Success: true, Message: "Registrasi berhasil"}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *finance.LoginRequest) (*finance.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email dan password wajib diisi")
	}

	var userID, passwordHash string
	err := db.Pool.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(req.Email)),
	).Scan(&userID, &passwordHash)
	if err != nil {
		return &finance.LoginResponse{Success: false, Message: "Email atau password salah"}, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return &finance.LoginResponse{Success: false, Message: "Email atau password salah"}, nil
	}

	var sessionID string
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id) VALUES ($1) RETURNING id`,
		userID,
	).Scan(&sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal buat session")
	}

	return &finance.LoginResponse{Success: true, Message: "Login berhasil", SessionId: sessionID}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *finance.SessionRequest) (*finance.StatusResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id wajib diisi")
	}
	_, err := db.Pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, req.SessionId)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus session")
	}
	return &finance.StatusResponse{Success: true, Message: "Logout berhasil"}, nil
}

func (s *AuthServer) GetSession(ctx context.Context, req *finance.SessionRequest) (*finance.SessionResponse, error) {
	if req.SessionId == "" {
		return &finance.SessionResponse{Valid: false}, nil
	}

	var userID, familyID, username string
	err := db.Pool.QueryRow(ctx, `
		SELECT u.id, u.family_id, u.username
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > NOW()
	`, req.SessionId).Scan(&userID, &familyID, &username)
	if err != nil {
		return &finance.SessionResponse{Valid: false}, nil
	}

	return &finance.SessionResponse{
		Valid:    true,
		UserId:   userID,
		FamilyId: familyID,
		Username: username,
	}, nil
}
