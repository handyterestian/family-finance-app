package handlers

import (
	"net/http"
	"time"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"
)

// AuthRegister — POST /auth/register
func AuthRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username   string `json:"username"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		FamilyName string `json:"family_name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Auth.Register(r.Context(), &finance.RegisterRequest{
		Username:   body.Username,
		Email:      body.Email,
		Password:   body.Password,
		FamilyName: body.FamilyName,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// AuthLogin — POST /auth/login
func AuthLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Auth.Login(r.Context(), &finance.LoginRequest{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	if !resp.Success {
		writeError(w, http.StatusUnauthorized, resp.Message)
		return
	}

	// Set cookie session_id
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    resp.SessionId,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"message":    resp.Message,
		"session_id": resp.SessionId,
	})
}

// AuthLogout — POST /auth/logout
func AuthLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := middleware.SessionID(r.Context())

	grpcclient.Auth.Logout(r.Context(), &finance.SessionRequest{SessionId: sessionID})

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Logout berhasil"})
}

// AuthMe — GET /auth/me
func AuthMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":   middleware.UserID(r.Context()),
		"family_id": middleware.FamilyID(r.Context()),
		"username":  middleware.Username(r.Context()),
	})
}
