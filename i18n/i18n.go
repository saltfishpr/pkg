// Package i18n provides simple internationalization support for Go applications.
// It offers two implementations: a simple key-value lookup (SimpleI18n) and a
// template-based system (TextTemplateI18n) that supports dynamic content.
package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"golang.org/x/text/language"
)

// ErrLanguageNotSupported is returned when a requested language is not available.
var ErrLanguageNotSupported = errors.New("language not supported")

// Options configures the behavior of Get operations.
type Options struct {
	Fallback language.Tag // fallback language when requested language is not found
	Arg      any          // argument passed to template execution (for TextTemplateI18n)
}

// Option is a function that modifies Options.
type Option func(*Options)

// WithFallback sets the fallback language.
func WithFallback(fallback language.Tag) Option {
	return func(o *Options) {
		o.Fallback = fallback
	}
}

// WithArg sets the argument for template execution.
func WithArg(arg any) Option {
	return func(o *Options) {
		o.Arg = arg
	}
}

// DefaultOptions is the default options used by Get methods.
var DefaultOptions = &Options{
	Fallback: language.English,
}

// I18n is the interface for internationalization operations.
type I18n interface {
	Get(lang language.Tag, options ...Option) (string, error)
}

// SimpleI18n provides a simple key-value lookup based i18n implementation.
type SimpleI18n struct {
	data map[language.Tag]string
}

// NewSimpleI18n creates a new SimpleI18n instance with the provided language data.
func NewSimpleI18n(data map[language.Tag]string) *SimpleI18n {
	return &SimpleI18n{
		data: data,
	}
}

// Get retrieves the localized string for the given language.
// If the language is not found, it falls back to the fallback language.
// Returns ErrLanguageNotSupported if neither the requested nor fallback language exists.
func (i *SimpleI18n) Get(lang language.Tag, options ...Option) (string, error) {
	opts := *DefaultOptions // copy default options
	for _, option := range options {
		option(&opts)
	}

	if s, ok := i.data[lang]; ok {
		return s, nil
	}
	if s, ok := i.data[opts.Fallback]; ok {
		return s, nil
	}
	return "", fmt.Errorf("language %s not supported: %w", lang, ErrLanguageNotSupported)
}

// TextTemplateI18n provides a template-based i18n implementation supporting dynamic content.
type TextTemplateI18n struct {
	data map[language.Tag]*template.Template
}

// NewTextTemplateI18n creates a new empty TextTemplateI18n instance.
func NewTextTemplateI18n() *TextTemplateI18n {
	return &TextTemplateI18n{
		data: make(map[language.Tag]*template.Template),
	}
}

// MustAdd adds a language template and panics on error.
// Useful for static initialization when template errors should be detected early.
func (i *TextTemplateI18n) MustAdd(lang language.Tag, tpl string) *TextTemplateI18n {
	i.data[lang] = template.Must(template.New("").Parse(tpl))
	return i
}

// Add adds a language template.
func (i *TextTemplateI18n) Add(lang language.Tag, tpl string) error {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	i.data[lang] = t
	return nil
}

// Get retrieves and executes the localized template for the given language.
// If the language is not found, it falls back to the fallback language.
// The template is executed with the argument provided via WithArg option.
// Returns ErrLanguageNotSupported if neither the requested nor fallback language exists.
func (i *TextTemplateI18n) Get(lang language.Tag, options ...Option) (string, error) {
	opts := *DefaultOptions // copy default options
	for _, option := range options {
		option(&opts)
	}

	tpl, ok := i.data[lang]
	if !ok {
		tpl, ok = i.data[opts.Fallback]
		if !ok {
			return "", fmt.Errorf("language %s not supported: %w", lang, ErrLanguageNotSupported)
		}
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, opts.Arg); err != nil {
		return "", fmt.Errorf("execute template %s: %w", tpl.Name(), err)
	}
	return buf.String(), nil
}
