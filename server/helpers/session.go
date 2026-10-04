package helpers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionInfo menyimpan data hasil validasi session.
type SessionInfo struct {
	UserID   string
	FamilyID string
	Username string
}

// ValidateSession memvalidasi sessionID ke DB dan mengembalikan SessionInfo
// atau error UNAUTHENTICATED. sessionID diambil langsung dari field proto request.
func ValidateSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*SessionInfo, error) {
	if sessionID == "" {
		return nil, status.Error(codes.Unauthenticated, "session-id tidak ada")
	}

	row := pool.QueryRow(ctx, `
		SELECT u.id, u.family_id, u.username
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1
		  AND s.expires_at > NOW()
	`, sessionID)

	var info SessionInfo
	err := row.Scan(&info.UserID, &info.FamilyID, &info.Username)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "session tidak valid atau sudah expired")
	}
	return &info, nil
}

// RecalcDaysUntilDue menghitung selisih hari dari hari ini ke dueDate (YYYY-MM-DD).
// Nilai negatif berarti sudah lewat jatuh tempo.
func RecalcDaysUntilDue(dueDate string) int32 {
	t, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return 0
	}
	now := time.Now().Truncate(24 * time.Hour)
	return int32(t.Sub(now).Hours() / 24)
}

// RecalcSavingFields menghitung monthly_required dan progress_pct.
// monthly_required = (target - current) / bulan_tersisa; min 0.
// progress_pct     = current / target * 100; max 100.
func RecalcSavingFields(currentBalance, targetAmount float64, targetDate string) (monthlyRequired, progressPct float64) {
	if targetAmount <= 0 {
		return 0, 0
	}
	progressPct = currentBalance / targetAmount * 100
	if progressPct > 100 {
		progressPct = 100
	}

	t, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return 0, progressPct
	}
	now := time.Now()
	monthsLeft := (t.Year()-now.Year())*12 + int(t.Month()-now.Month())
	if monthsLeft <= 0 {
		return 0, progressPct
	}
	remaining := targetAmount - currentBalance
	if remaining <= 0 {
		return 0, progressPct
	}
	monthlyRequired = remaining / float64(monthsLeft)
	return monthlyRequired, progressPct
}

// RecalcEmergencyFund menghitung target_balance dan months_covered (default 6 bulan).
func RecalcEmergencyFund(currentBalance, monthlyAvg float64) (targetBalance, monthsCovered float64) {
	return RecalcEmergencyFundCustom(currentBalance, monthlyAvg, 6)
}

// RecalcEmergencyFundCustom menghitung dengan target_months kustom.
func RecalcEmergencyFundCustom(currentBalance, monthlyAvg float64, targetMonths int) (targetBalance, monthsCovered float64) {
	if targetMonths <= 0 {
		targetMonths = 6
	}
	targetBalance = monthlyAvg * float64(targetMonths)
	if monthlyAvg > 0 {
		monthsCovered = currentBalance / monthlyAvg
	}
	return targetBalance, monthsCovered
}
