package handlers

import (
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
)

// BudgetList — GET /budgets?month=YYYY-MM
func BudgetList(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	resp, err := grpcclient.Budget.ListBudgets(r.Context(), &finance.ListBudgetsRequest{
		SessionId: middleware.SessionID(r.Context()),
		Month:     month,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// BudgetSet — POST /budgets
func BudgetSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CategoryName string  `json:"category_name"`
		Month        string  `json:"month"`
		Amount       float64 `json:"amount"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Budget.SetBudget(r.Context(), &finance.SetBudgetRequest{
		SessionId:    middleware.SessionID(r.Context()),
		CategoryName: body.CategoryName,
		Month:        body.Month,
		Amount:       body.Amount,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// BudgetUpdate — PUT /budgets/{id}
func BudgetUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Amount float64 `json:"amount"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Budget.UpdateBudget(r.Context(), &finance.UpdateBudgetRequest{
		SessionId: middleware.SessionID(r.Context()),
		BudgetId:  id,
		Amount:    body.Amount,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// BudgetDelete — DELETE /budgets/{id}
func BudgetDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := grpcclient.Budget.DeleteBudget(r.Context(), &finance.DeleteBudgetRequest{
		SessionId: middleware.SessionID(r.Context()),
		BudgetId:  id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// EmergencyFundGet — GET /budgets/emergency
func EmergencyFundGet(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Budget.GetEmergencyFund(r.Context(), &finance.GetEmergencyFundRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// EmergencyFundSet — PUT /budgets/emergency
func EmergencyFundSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentBalance float64 `json:"current_balance"`
		TargetMonths   int32   `json:"target_months"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Budget.SetEmergencyFund(r.Context(), &finance.SetEmergencyFundRequest{
		SessionId:      middleware.SessionID(r.Context()),
		CurrentBalance: body.CurrentBalance,
		TargetMonths:   body.TargetMonths,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// EmergencyFundDeposit — POST /budgets/emergency/deposit
func EmergencyFundDeposit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Amount float64 `json:"amount"`
		Note   string  `json:"note"`
		Date   string  `json:"date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Budget.DepositEmergencyFund(r.Context(), &finance.DepositEmergencyFundRequest{
		SessionId: middleware.SessionID(r.Context()),
		Amount:    body.Amount,
		Note:      body.Note,
		Date:      body.Date,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// EmergencyFundHistory — GET /budgets/emergency/history
func EmergencyFundHistory(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Budget.ListEmergencyFundHistory(r.Context(), &finance.ListEmergencyFundHistoryRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// RecurringBudgetList — GET /budgets/recurring
func RecurringBudgetList(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Budget.ListRecurringBudgets(r.Context(), &finance.ListRecurringBudgetsRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// RecurringBudgetSet — POST /budgets/recurring
func RecurringBudgetSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CategoryName string  `json:"category_name"`
		Amount       float64 `json:"amount"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Budget.SetRecurringBudget(r.Context(), &finance.SetRecurringBudgetRequest{
		SessionId:    middleware.SessionID(r.Context()),
		CategoryName: body.CategoryName,
		Amount:       body.Amount,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// RecurringBudgetDelete — DELETE /budgets/recurring/{id}
func RecurringBudgetDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := grpcclient.Budget.DeleteRecurringBudget(r.Context(), &finance.DeleteRecurringBudgetRequest{
		SessionId:         middleware.SessionID(r.Context()),
		RecurringBudgetId: id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// RecurringBudgetApply — POST /budgets/recurring/apply
func RecurringBudgetApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Month string `json:"month"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Budget.ApplyRecurringBudgets(r.Context(), &finance.ApplyRecurringBudgetsRequest{
		SessionId: middleware.SessionID(r.Context()),
		Month:     body.Month,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
