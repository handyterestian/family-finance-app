package handlers

import (
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
)

// DebtList — GET /debts
func DebtList(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Debt.ListDebts(r.Context(), &finance.ListDebtsRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DebtCreate — POST /debts
func DebtCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string  `json:"name"`
		TotalAmount   float64 `json:"total_amount"`
		MonthlyPayment float64 `json:"monthly_payment"`
		DueDate       string  `json:"due_date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Debt.CreateDebt(r.Context(), &finance.CreateDebtRequest{
		SessionId:      middleware.SessionID(r.Context()),
		Name:           body.Name,
		TotalAmount:    body.TotalAmount,
		MonthlyPayment: body.MonthlyPayment,
		DueDate:        body.DueDate,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// DebtUpdate — PUT /debts/{id}
func DebtUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name          string  `json:"name"`
		TotalAmount   float64 `json:"total_amount"`
		MonthlyPayment float64 `json:"monthly_payment"`
		DueDate       string  `json:"due_date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Debt.UpdateDebt(r.Context(), &finance.UpdateDebtRequest{
		SessionId:      middleware.SessionID(r.Context()),
		DebtId:         id,
		Name:           body.Name,
		TotalAmount:    body.TotalAmount,
		MonthlyPayment: body.MonthlyPayment,
		DueDate:        body.DueDate,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DebtPay — POST /debts/{id}/pay
func DebtPay(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		PaidBy string  `json:"paid_by"`
		Amount float64 `json:"amount"`
		Date   string  `json:"date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Debt.PayDebt(r.Context(), &finance.PayDebtRequest{
		SessionId: middleware.SessionID(r.Context()),
		DebtId:    id,
		PaidBy:    body.PaidBy,
		Amount:    body.Amount,
		Date:      body.Date,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DebtDelete — DELETE /debts/{id}
func DebtDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := grpcclient.Debt.DeleteDebt(r.Context(), &finance.DeleteDebtRequest{
		SessionId: middleware.SessionID(r.Context()),
		DebtId:    id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DebtPaymentList — GET /debts/{id}/payments
func DebtPaymentList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := grpcclient.Debt.ListDebtPayments(r.Context(), &finance.ListDebtPaymentsRequest{
		SessionId: middleware.SessionID(r.Context()),
		DebtId:    id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
