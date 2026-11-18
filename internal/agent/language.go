package agent

import (
	"context"
	_ "embed"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

//go:embed prompts/language_system.txt
var languageSystemPromptFile string

var languageSystemPrompt = strings.TrimSpace(languageSystemPromptFile)

var languageNames = map[string]string{
	models.LocaleEnglish: "English",
	models.LocaleChinese: "Simplified Chinese (简体中文)",
}

func languagePrompt(locale string) string {
	name, ok := languageNames[locale]
	if !ok {
		name = languageNames[models.LocaleEnglish]
	}
	return strings.ReplaceAll(languageSystemPrompt, "{{language}}", name)
}

func languageMessage(ctx context.Context, db *gorm.DB) ai.Message {
	locale := models.LocaleEnglish
	if setting, err := models.LoadSetting(db.WithContext(ctx)); err == nil {
		locale = setting.Locale
	}
	return ai.Message{Role: ai.RoleSystem, Content: languagePrompt(locale)}
}
