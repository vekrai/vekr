package i18n

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"vekr/locales"
)

var DefaultLanguage = language.English

var bundle = mustLoadBundle()

type contextKey struct{}

type state struct {
	lang      language.Tag
	localizer *goi18n.Localizer
}

func mustLoadBundle() *goi18n.Bundle {
	b := goi18n.NewBundle(DefaultLanguage)
	b.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	files, err := fs.Glob(locales.Files, "*.yaml")
	if err != nil {
		panic(fmt.Sprintf("i18n: list locale files: %v", err))
	}
	for _, name := range files {
		if _, err := b.LoadMessageFileFS(locales.Files, name); err != nil {
			panic(fmt.Sprintf("i18n: load %s: %v", name, err))
		}
	}
	return b
}

func supportedLanguages() []language.Tag {
	tags := []language.Tag{DefaultLanguage}
	for _, tag := range bundle.LanguageTags() {
		if tag != DefaultLanguage {
			tags = append(tags, tag)
		}
	}
	return tags
}

func Resolve(sessionLocale, acceptLanguage string) language.Tag {
	var wanted []language.Tag
	if sessionLocale != "" {
		if tag, err := language.Parse(sessionLocale); err == nil {
			wanted = append(wanted, tag)
		}
	}
	if tags, _, err := language.ParseAcceptLanguage(acceptLanguage); err == nil {
		wanted = append(wanted, tags...)
	}

	supported := supportedLanguages()
	_, index, _ := language.NewMatcher(supported).Match(wanted...)
	return supported[index]
}

func WithLanguage(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, contextKey{}, state{lang: lang, localizer: goi18n.NewLocalizer(bundle, lang.String())})
}

func Middleware(locale func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := Resolve(locale(r), r.Header.Get("Accept-Language"))
			next.ServeHTTP(w, r.WithContext(WithLanguage(r.Context(), lang)))
		})
	}
}

func fromContext(ctx context.Context) state {
	if s, ok := ctx.Value(contextKey{}).(state); ok {
		return s
	}
	return state{lang: DefaultLanguage, localizer: goi18n.NewLocalizer(bundle, DefaultLanguage.String())}
}

func Lang(ctx context.Context) string {
	return fromContext(ctx).lang.String()
}

func T(ctx context.Context, id string) string {
	return localize(ctx, &goi18n.LocalizeConfig{MessageID: id})
}

func TData(ctx context.Context, id string, data map[string]any) string {
	return localize(ctx, &goi18n.LocalizeConfig{MessageID: id, TemplateData: data})
}

func TCount(ctx context.Context, id string, count int, data map[string]any) string {
	withCount := map[string]any{"PluralCount": count}
	for k, v := range data {
		withCount[k] = v
	}
	return localize(ctx, &goi18n.LocalizeConfig{MessageID: id, PluralCount: count, TemplateData: withCount})
}

func localize(ctx context.Context, config *goi18n.LocalizeConfig) string {
	text, err := fromContext(ctx).localizer.Localize(config)
	if err != nil {
		log.Printf("i18n: %v", err)
		return config.MessageID
	}
	return text
}
