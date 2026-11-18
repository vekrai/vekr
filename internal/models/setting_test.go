package models

import "testing"

func TestLoadSettingDefaultsToEnglish(t *testing.T) {
	db := testDB(t)
	db.Exec("DELETE FROM settings")

	setting, err := LoadSetting(db)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if setting.Locale != LocaleEnglish {
		t.Errorf("locale = %q, want en", setting.Locale)
	}
}

func TestToggleLocaleFlipsBetweenEnglishAndChinese(t *testing.T) {
	db := testDB(t)
	db.Exec("DELETE FROM settings")

	setting, err := ToggleLocale(db)
	if err != nil || setting.Locale != LocaleChinese {
		t.Fatalf("first toggle = %q, %v; want zh", setting.Locale, err)
	}
	setting, err = ToggleLocale(db)
	if err != nil || setting.Locale != LocaleEnglish {
		t.Fatalf("second toggle = %q, %v; want en", setting.Locale, err)
	}

	stored, _ := LoadSetting(db)
	if stored.Locale != LocaleEnglish {
		t.Errorf("stored locale = %q, want en", stored.Locale)
	}
}
