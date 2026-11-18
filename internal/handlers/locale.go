package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"vekr/internal/models"
)

type LocaleHandler struct {
	db *gorm.DB
}

func NewLocaleHandler(db *gorm.DB) *LocaleHandler {
	return &LocaleHandler{db: db}
}

func (h *LocaleHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	if _, err := models.ToggleLocale(h.db); err != nil {
		http.Error(w, "could not switch language", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, returnTo(r, "/"), http.StatusSeeOther)
}

func SettingLocale(db *gorm.DB) func(*http.Request) string {
	return func(r *http.Request) string {
		setting, err := models.LoadSetting(db.WithContext(r.Context()))
		if err != nil {
			return ""
		}
		return setting.Locale
	}
}
