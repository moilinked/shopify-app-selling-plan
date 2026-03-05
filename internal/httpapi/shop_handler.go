package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"shopify-app-authentication/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShopHandler struct {
	DB *gorm.DB
}

// Routes 注册店铺资源的 RESTful 路由。
func (h *ShopHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// List 查询全部店铺列表。 GET /admin/shops
func (h *ShopHandler) List(w http.ResponseWriter, r *http.Request) {
	var shops []models.Shop
	if err := h.DB.Find(&shops).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, shops)
}

// Create 新增一个店铺。 POST /admin/shops
func (h *ShopHandler) Create(w http.ResponseWriter, r *http.Request) {
	var shop models.Shop
	if err := json.NewDecoder(r.Body).Decode(&shop); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if shop.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if err := h.DB.Create(&shop).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, shop)
}

// Get 根据 ID 查询单个店铺。 GET /admin/shops/{id}
func (h *ShopHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var shop models.Shop
	if err := h.DB.First(&shop, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "shop not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, shop)
}

// Update 根据 ID 更新店铺信息。 PUT /admin/shops/{id}
func (h *ShopHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var shop models.Shop
	if err := h.DB.First(&shop, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "shop not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var updates models.Shop
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.DB.Model(&shop).Updates(models.Shop{
		Name:          updates.Name,
		AdminAPI:      updates.AdminAPI,
		OnlineShopURL: updates.OnlineShopURL,
		StorefrontAPI: updates.StorefrontAPI,
	}).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, shop)
}

// Delete 根据 ID 软删除店铺。 DELETE /admin/shops/{id}
func (h *ShopHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	result := h.DB.Delete(&models.Shop{}, "id = ?", id)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "shop not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// writeJSON 将数据序列化为 JSON 并写入响应。
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
