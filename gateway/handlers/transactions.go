package handlers

import (
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
)

// TransactionList — GET /transactions?month=YYYY-MM
func TransactionList(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	resp, err := grpcclient.Transaction.ListTransactions(r.Context(), &finance.ListTransactionsRequest{
		SessionId: middleware.SessionID(r.Context()),
		Month:     month,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// TransactionCreate — POST /transactions
func TransactionCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type         string  `json:"type"`
		Amount       float64 `json:"amount"`
		CategoryName string  `json:"category_name"`
		Member       string  `json:"member"`
		Date         string  `json:"date"`
		Note         string  `json:"note"`
		WalletID     string  `json:"wallet_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Transaction.CreateTransaction(r.Context(), &finance.CreateTransactionRequest{
		SessionId:    middleware.SessionID(r.Context()),
		Type:         body.Type,
		Amount:       body.Amount,
		CategoryName: body.CategoryName,
		Member:       body.Member,
		Date:         body.Date,
		Note:         body.Note,
		WalletId:     body.WalletID,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// TransactionUpdate — PUT /transactions/{id}
func TransactionUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Type         string  `json:"type"`
		Amount       float64 `json:"amount"`
		CategoryName string  `json:"category_name"`
		Member       string  `json:"member"`
		Date         string  `json:"date"`
		Note         string  `json:"note"`
		WalletID     string  `json:"wallet_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Transaction.UpdateTransaction(r.Context(), &finance.UpdateTransactionRequest{
		SessionId:     middleware.SessionID(r.Context()),
		TransactionId: id,
		Type:          body.Type,
		Amount:        body.Amount,
		CategoryName:  body.CategoryName,
		Member:        body.Member,
		Date:          body.Date,
		Note:          body.Note,
		WalletId:      body.WalletID,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// TransactionDelete — DELETE /transactions/{id}
func TransactionDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := grpcclient.Transaction.DeleteTransaction(r.Context(), &finance.DeleteTransactionRequest{
		SessionId:     middleware.SessionID(r.Context()),
		TransactionId: id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
