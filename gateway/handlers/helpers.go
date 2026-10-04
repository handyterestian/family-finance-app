package handlers

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// writeJSON menulis response JSON dengan status code yang diberikan.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// writeError menulis response error JSON.
func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// grpcErr memetakan error gRPC ke HTTP status code dan menulis response.
func grpcErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch st.Code() {
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, st.Message())
	case codes.FailedPrecondition:
		writeError(w, http.StatusConflict, st.Message())
	default:
		writeError(w, http.StatusInternalServerError, st.Message())
	}
}

// decodeJSON mendecode request body JSON ke v. Mengembalikan false jika gagal.
func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Body JSON tidak valid: "+err.Error())
		return false
	}
	return true
}
