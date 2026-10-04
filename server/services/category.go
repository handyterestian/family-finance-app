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

// CategoryServer mengimplementasikan finance.CategoryServiceServer.
type CategoryServer struct {
	finance.UnimplementedCategoryServiceServer
}

func (s *CategoryServer) ListCategories(ctx context.Context, req *finance.ListCategoriesRequest) (*finance.CategoryListResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, family_id, name, type, created_at::text
		 FROM categories WHERE family_id = $1 ORDER BY name`,
		sess.FamilyID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal query kategori")
	}
	defer rows.Close()

	var cats []*finance.Category
	for rows.Next() {
		c := &finance.Category{}
		if err := rows.Scan(&c.Id, &c.FamilyId, &c.Name, &c.Type, &c.CreatedAt); err != nil {
			return nil, status.Error(codes.Internal, "gagal scan kategori")
		}
		cats = append(cats, c)
	}
	return &finance.CategoryListResponse{Success: true, Categories: cats}, nil
}

func (s *CategoryServer) CreateCategory(ctx context.Context, req *finance.CreateCategoryRequest) (*finance.CategoryResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, status.Error(codes.InvalidArgument, "nama kategori wajib diisi")
	}
	if req.Type != "income" && req.Type != "expense" && req.Type != "both" {
		return nil, status.Error(codes.InvalidArgument, "type harus income, expense, atau both")
	}

	c := &finance.Category{}
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO categories (family_id, name, type)
		 VALUES ($1, $2, $3)
		 RETURNING id, family_id, name, type, created_at::text`,
		sess.FamilyID, strings.TrimSpace(req.Name), req.Type,
	).Scan(&c.Id, &c.FamilyId, &c.Name, &c.Type, &c.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, status.Error(codes.AlreadyExists, "kategori sudah ada")
		}
		return nil, status.Error(codes.Internal, "gagal buat kategori")
	}

	return &finance.CategoryResponse{Success: true, Message: "Kategori berhasil dibuat", Category: c}, nil
}

func (s *CategoryServer) UpdateCategory(ctx context.Context, req *finance.UpdateCategoryRequest) (*finance.CategoryResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, status.Error(codes.InvalidArgument, "name wajib diisi")
	}
	if req.Type != "income" && req.Type != "expense" && req.Type != "both" {
		return nil, status.Error(codes.InvalidArgument, "type harus income, expense, atau both")
	}

	newName := strings.TrimSpace(req.NewName)
	if newName == "" {
		newName = strings.TrimSpace(req.Name) // tidak ganti nama, hanya tipe
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal mulai transaksi")
	}
	defer tx.Rollback(ctx)

	// Update nama dan tipe di tabel categories
	c := &finance.Category{}
	err = tx.QueryRow(ctx,
		`UPDATE categories
		 SET name = $1, type = $2
		 WHERE family_id = $3 AND name = $4
		 RETURNING id, family_id, name, type, created_at::text`,
		newName, req.Type, sess.FamilyID, strings.TrimSpace(req.Name),
	).Scan(&c.Id, &c.FamilyId, &c.Name, &c.Type, &c.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, status.Error(codes.AlreadyExists, "nama kategori baru sudah digunakan")
		}
		return nil, status.Error(codes.NotFound, "kategori tidak ditemukan")
	}

	// Jika nama berubah, cascade UPDATE ke transactions dan budgets
	if newName != strings.TrimSpace(req.Name) {
		_, err = tx.Exec(ctx,
			`UPDATE transactions SET category_name = $1
			 WHERE family_id = $2 AND category_name = $3`,
			newName, sess.FamilyID, strings.TrimSpace(req.Name),
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "gagal update category_name di transactions")
		}
		_, err = tx.Exec(ctx,
			`UPDATE budgets SET category_name = $1
			 WHERE family_id = $2 AND category_name = $3`,
			newName, sess.FamilyID, strings.TrimSpace(req.Name),
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "gagal update category_name di budgets")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "gagal commit")
	}

	return &finance.CategoryResponse{Success: true, Message: "Kategori berhasil diupdate", Category: c}, nil
}

func (s *CategoryServer) DeleteCategory(ctx context.Context, req *finance.DeleteCategoryRequest) (*finance.StatusResponse, error) {
	sess, err := helpers.ValidateSession(ctx, db.Pool, req.SessionId)
	if err != nil {
		return nil, err
	}

	// Cek apakah ada transaksi yang masih pakai kategori ini
	var count int
	err = db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM transactions
		 WHERE family_id = $1 AND category_name = $2`,
		sess.FamilyID, req.Name,
	).Scan(&count)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal cek transaksi")
	}
	if count > 0 {
		return nil, status.Errorf(codes.FailedPrecondition,
			"kategori masih digunakan oleh %d transaksi, tidak bisa dihapus", count)
	}

	res, err := db.Pool.Exec(ctx,
		`DELETE FROM categories WHERE family_id = $1 AND name = $2`,
		sess.FamilyID, req.Name,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "gagal hapus kategori")
	}
	if res.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "kategori tidak ditemukan")
	}

	return &finance.StatusResponse{Success: true, Message: "Kategori berhasil dihapus"}, nil
}
