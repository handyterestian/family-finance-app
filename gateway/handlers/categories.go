package handlers

import (
	"net/http"

	"family-finance/gateway/generated"
	"family-finance/gateway/grpcclient"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
)

// CategoryList — GET /categories
func CategoryList(w http.ResponseWriter, r *http.Request) {
	resp, err := grpcclient.Category.ListCategories(r.Context(), &finance.ListCategoriesRequest{
		SessionId: middleware.SessionID(r.Context()),
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// CategoryCreate — POST /categories
func CategoryCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Category.CreateCategory(r.Context(), &finance.CreateCategoryRequest{
		SessionId: middleware.SessionID(r.Context()),
		Name:      body.Name,
		Type:      body.Type,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// CategoryUpdate — PUT /categories/{name}
func CategoryUpdate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		NewName string `json:"new_name"`
		Type    string `json:"type"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	resp, err := grpcclient.Category.UpdateCategory(r.Context(), &finance.UpdateCategoryRequest{
		SessionId: middleware.SessionID(r.Context()),
		Name:      name,
		NewName:   body.NewName,
		Type:      body.Type,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// CategoryDelete — DELETE /categories/{name}
func CategoryDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	resp, err := grpcclient.Category.DeleteCategory(r.Context(), &finance.DeleteCategoryRequest{
		SessionId: middleware.SessionID(r.Context()),
		Name:      name,
	})
	if err != nil {
		grpcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
