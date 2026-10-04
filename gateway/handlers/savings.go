package handlers

import (
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
)

// SavingList — GET /savings
func SavingList(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Saving.ListSavings(r.Context(), &finance.ListSavingsRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SavingCreate — POST /savings
func SavingCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string  `json:"name"`
		TargetAmount   float64 `json:"target_amount"`
		InitialBalance float64 `json:"initial_balance"`
		TargetDate     string  `json:"target_date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Saving.CreateSaving(r.Context(), &finance.CreateSavingRequest{
		SessionId:      middleware.SessionID(r.Context()),
		Name:           body.Name,
		TargetAmount:   body.TargetAmount,
		InitialBalance: body.InitialBalance,
		TargetDate:     body.TargetDate,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// SavingUpdate — PUT /savings/{id}
func SavingUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name         string  `json:"name"`
		TargetAmount float64 `json:"target_amount"`
		TargetDate   string  `json:"target_date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Saving.UpdateSaving(r.Context(), &finance.UpdateSavingRequest{
		SessionId:    middleware.SessionID(r.Context()),
		SavingId:     id,
		Name:         body.Name,
		TargetAmount: body.TargetAmount,
		TargetDate:   body.TargetDate,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SavingDeposit — POST /savings/{id}/deposit
func SavingDeposit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Amount float64 `json:"amount"`
		Date   string  `json:"date"`
		Note   string  `json:"note"`
		Member string  `json:"member"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Saving.DepositSaving(r.Context(), &finance.DepositSavingRequest{
		SessionId: middleware.SessionID(r.Context()),
		SavingId:  id,
		Amount:    body.Amount,
		Date:      body.Date,
		Note:      body.Note,
		Member:    body.Member,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SavingDelete — DELETE /savings/{id}
func SavingDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := grpcclient.Saving.DeleteSaving(r.Context(), &finance.DeleteSavingRequest{
		SessionId: middleware.SessionID(r.Context()),
		SavingId:  id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
