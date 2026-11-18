package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	LocaleEnglish = "en"
	LocaleChinese = "zh"
)

type Setting struct {
	ID        uint   `gorm:"primaryKey"`
	Locale    string `gorm:"default:en"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func LoadSetting(db *gorm.DB) (Setting, error) {
	var setting Setting
	err := db.Order("id").FirstOrCreate(&setting).Error
	return setting, err
}

func ToggleLocale(db *gorm.DB) (Setting, error) {
	setting, err := LoadSetting(db)
	if err != nil {
		return setting, err
	}
	if setting.Locale == LocaleChinese {
		setting.Locale = LocaleEnglish
	} else {
		setting.Locale = LocaleChinese
	}
	err = db.Model(&setting).Update("locale", setting.Locale).Error
	return setting, err
}
