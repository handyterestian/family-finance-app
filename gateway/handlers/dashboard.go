package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"
)

// DashboardGet — GET /dashboard
func DashboardGet(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Dashboard.GetDashboard(r.Context(), &finance.GetDashboardRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DashboardSeed — POST /dashboard/seed
func DashboardSeed(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Dashboard.SeedData(r.Context(), &finance.SeedDataRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DashboardClear — DELETE /dashboard/clear
func DashboardClear(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Dashboard.ClearData(r.Context(), &finance.ClearDataRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// BackupExport — GET /backup
// Memanggil ExportBackup di server lalu mengirimkan file JSON ke browser.
func BackupExport(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Dashboard.ExportBackup(r.Context(), &finance.ExportBackupRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	filename := fmt.Sprintf("backup-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp.JsonData))
}

// BackupRestore — POST /restore
// Menerima file JSON dari multipart/form-data (field "file") atau raw JSON body.
func BackupRestore(w http.ResponseWriter, r *http.Request) {
	var jsonData string

	ct := r.Header.Get("Content-Type")
	if len(ct) >= 19 && ct[:19] == "multipart/form-data" {
		// Multipart: ambil field "file"
		if err := r.ParseMultipartForm(10 << 20); err != nil { // maks 10 MB
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gagal parse form"})
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field 'file' tidak ditemukan"})
			return
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gagal membaca file"})
			return
		}
		jsonData = string(b)
	} else {
		// Raw JSON body
		b, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gagal membaca body"})
			return
		}
		jsonData = string(b)
	}

	resp, err := grpcclient.Dashboard.ImportRestore(r.Context(), &finance.ImportRestoreRequest{
		SessionId: middleware.SessionID(r.Context()),
		JsonData:  jsonData,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
