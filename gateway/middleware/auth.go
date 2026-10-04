package middleware

import (
	"context"
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
)

// contextKey adalah tipe private untuk context keys.
type contextKey string

const (
	KeySessionID contextKey = "session_id"
	KeyUserID    contextKey = "user_id"
	KeyFamilyID  contextKey = "family_id"
	KeyUsername  contextKey = "username"
)

// RequireAuth adalah middleware chi yang memvalidasi cookie session_id.
// Jika valid, menyimpan user_id, family_id, dan username ke context.
// Jika tidak valid, mengembalikan 401.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		resp, err := grpcclient.Auth.GetSession(r.Context(), &finance.SessionRequest{
			SessionId: cookie.Value,
		})
		if err != nil || !resp.Valid {
			http.Error(w, `{"error":"Session tidak valid atau sudah expired"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), KeySessionID, cookie.Value)
		ctx = context.WithValue(ctx, KeyUserID, resp.UserId)
		ctx = context.WithValue(ctx, KeyFamilyID, resp.FamilyId)
		ctx = context.WithValue(ctx, KeyUsername, resp.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SessionID mengambil session_id dari context.
func SessionID(ctx context.Context) string {
	v, _ := ctx.Value(KeySessionID).(string)
	return v
}

// UserID mengambil user_id dari context.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(KeyUserID).(string)
	return v
}

// FamilyID mengambil family_id dari context.
func FamilyID(ctx context.Context) string {
	v, _ := ctx.Value(KeyFamilyID).(string)
	return v
}

// Username mengambil username dari context.
func Username(ctx context.Context) string {
	v, _ := ctx.Value(KeyUsername).(string)
	return v
}
