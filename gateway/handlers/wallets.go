package handlers

import (
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
)

// WalletList — GET /wallets
func WalletList(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Wallet.ListWallets(r.Context(), &finance.ListWalletsRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// WalletCreate — POST /wallets
func WalletCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string  `json:"name"`
		Type    string  `json:"type"`
		Balance float64 `json:"balance"`
		Color   string  `json:"color"`
		Note    string  `json:"note"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	resp, err := grpcclient.Wallet.CreateWallet(r.Context(), &finance.CreateWalletRequest{
		SessionId: middleware.SessionID(r.Context()),
		Name:      body.Name,
		Type:      body.Type,
		Balance:   body.Balance,
		Color:     body.Color,
		Note:      body.Note,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// WalletUpdate — PUT /wallets/{id}
func WalletUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Color string `json:"color"`
		Note  string `json:"note"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	resp, err := grpcclient.Wallet.UpdateWallet(r.Context(), &finance.UpdateWalletRequest{
		SessionId: middleware.SessionID(r.Context()),
		WalletId:  id,
		Name:      body.Name,
		Type:      body.Type,
		Color:     body.Color,
		Note:      body.Note,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// WalletDelete — DELETE /wallets/{id}
func WalletDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp, err := grpcclient.Wallet.DeleteWallet(r.Context(), &finance.DeleteWalletRequest{
		SessionId: middleware.SessionID(r.Context()),
		WalletId:  id,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// WalletTransfer — POST /wallets/transfer
func WalletTransfer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromWalletID string  `json:"from_wallet_id"`
		ToWalletID   string  `json:"to_wallet_id"`
		Amount       float64 `json:"amount"`
		Note         string  `json:"note"`
		Date         string  `json:"date"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	resp, err := grpcclient.Wallet.TransferWallet(r.Context(), &finance.TransferWalletRequest{
		SessionId:    middleware.SessionID(r.Context()),
		FromWalletId: body.FromWalletID,
		ToWalletId:   body.ToWalletID,
		Amount:       body.Amount,
		Note:         body.Note,
		Date:         body.Date,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
