// Package i18n provides keyed translations with optional text/template rendering.
package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"text/template"

	"golang.org/x/text/language"
)

// ErrTranslationNotFound is returned when a key has no translation in either
// the requested language or the configured fallback language.
var ErrTranslationNotFound = errors.New("translation not found")

// Dictionary stores translations by language and message key.
type Dictionary map[language.Tag]map[string]string

// Translator looks up translations from a Dictionary.
type Translator struct {
	dictionary Dictionary
	fallback   language.Tag

	templateMu sync.RWMutex
	templates  map[string]*template.Template
}

// Option configures a Translator.
type Option func(*Translator)

// WithFallback sets the language used when a requested translation is missing.
func WithFallback(lang language.Tag) Option {
	return func(t *Translator) {
		t.fallback = lang
	}
}

// New creates a Translator. English is used as the fallback language unless
// overridden with WithFallback.
func New(dictionary Dictionary, options ...Option) *Translator {
	t := &Translator{
		dictionary: dictionary,
		fallback:   language.English,
		templates:  make(map[string]*template.Template),
	}
	for _, option := range options {
		option(t)
	}
	return t
}

// T returns the plain-text translation for key in lang. It falls back to the
// configured fallback language when the requested translation is unavailable.
func (t *Translator) T(lang language.Tag, key string) (string, error) {
	return t.lookup(lang, key)
}

// F executes the text/template translation for key in lang with data. It uses
// the same fallback behavior as T.
func (t *Translator) F(lang language.Tag, key string, data any) (string, error) {
	text, err := t.lookup(lang, key)
	if err != nil {
		return "", err
	}

	tpl, err := t.template(key, text)
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	if err := tpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("execute translation template %q: %w", key, err)
	}
	return output.String(), nil
}

func (t *Translator) template(key, text string) (*template.Template, error) {
	t.templateMu.RLock()
	tpl := t.templates[text]
	t.templateMu.RUnlock()
	if tpl != nil {
		return tpl, nil
	}

	t.templateMu.Lock()
	defer t.templateMu.Unlock()
	if tpl = t.templates[text]; tpl != nil {
		return tpl, nil
	}

	tpl, err := template.New("").Parse(text)
	if err != nil {
		return nil, fmt.Errorf("parse translation template %q: %w", key, err)
	}
	t.templates[text] = tpl
	return tpl, nil
}

func (t *Translator) lookup(lang language.Tag, key string) (string, error) {
	if text, ok := t.dictionary[lang][key]; ok {
		return text, nil
	}
	if text, ok := t.dictionary[t.fallback][key]; ok {
		return text, nil
	}
	return "", fmt.Errorf("translation %q for language %s: %w", key, lang, ErrTranslationNotFound)
}
