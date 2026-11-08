// Package i18n provides keyed translations with optional text/template
// rendering. Languages are matched with [golang.org/x/text/language.Matcher];
// pluralization and locale-aware formatting are out of scope.
package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"sync"
	"text/template"

	"golang.org/x/text/language"
)

// ErrTranslationNotFound is returned when a key has no translation in either
// the matched language or the configured fallback language.
var ErrTranslationNotFound = errors.New("translation not found")

// Dictionary stores translations by language and message key. It is retained
// by [New] and must not be modified afterwards.
type Dictionary map[language.Tag]map[string]string

// Translator looks up translations from a Dictionary. It is safe for
// concurrent use.
type Translator struct {
	dictionary Dictionary
	fallback   language.Tag
	tags       []language.Tag
	matcher    language.Matcher

	templateMu sync.RWMutex
	templates  map[string]*template.Template
}

// Option configures a Translator during construction.
type Option func(*Translator)

// WithFallback sets the language used when no supported language matches or
// the matched language lacks a key.
func WithFallback(lang language.Tag) Option {
	return func(t *Translator) {
		t.fallback = lang
	}
}

// New creates a Translator. English is the default fallback; override it with
// [WithFallback]. Requests are matched against the dictionary keys with
// [language.Matcher].
func New(dictionary Dictionary, options ...Option) *Translator {
	t := &Translator{
		dictionary: dictionary,
		fallback:   language.English,
		templates:  make(map[string]*template.Template),
	}
	for _, option := range options {
		option(t)
	}

	t.tags = make([]language.Tag, 0, len(t.dictionary))
	for tag := range t.dictionary {
		t.tags = append(t.tags, tag)
	}
	sort.Slice(t.tags, func(i, j int) bool {
		return t.tags[i].String() < t.tags[j].String()
	})
	if len(t.tags) > 0 {
		t.matcher = language.NewMatcher(t.tags)
	}
	return t
}

// Match returns the supported language that best matches tags in order of
// preference. It returns the fallback language when none of the tags is
// supported.
func (t *Translator) Match(tags ...language.Tag) language.Tag {
	if t.matcher == nil || len(tags) == 0 {
		return t.fallback
	}

	_, index, confidence := t.matcher.Match(tags...)
	if confidence == language.No {
		return t.fallback
	}

	return t.tags[index]
}

// T returns the plain-text translation for key in lang, which is matched to the
// closest supported language. Missing keys fall back to the configured fallback
// language.
func (t *Translator) T(lang language.Tag, key string) (string, error) {
	return t.lookup(lang, key)
}

// F executes the text/template translation for key in lang with data, using the
// same matching and fallback rules as [Translator.T]. Missing fields in data are
// an error.
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

	tpl, err := template.New("").Option("missingkey=error").Parse(text)
	if err != nil {
		return nil, fmt.Errorf("parse translation template %q: %w", key, err)
	}
	t.templates[text] = tpl
	return tpl, nil
}

func (t *Translator) lookup(lang language.Tag, key string) (string, error) {
	matched := t.Match(lang)
	if text, ok := t.dictionary[matched][key]; ok {
		return text, nil
	}
	if text, ok := t.dictionary[t.fallback][key]; ok {
		return text, nil
	}
	return "", fmt.Errorf("translation %q for language %s: %w", key, lang, ErrTranslationNotFound)
}
