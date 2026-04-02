// Package i18n provides simple internationalization support for Go
// applications. It ships two implementations:
//   - [SimpleI18n]: direct key-value lookup by language tag.
//   - [TextTemplateI18n]: template-based lookup using [text/template],
//     supporting dynamic content via the [WithArg] option.
//
// Both implementations fall back to a configurable language (default English)
// when the requested language is unavailable.
package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"golang.org/x/text/language"
)

// ErrLanguageNotSupported is returned when neither the requested language
// nor the fallback language is available.
var ErrLanguageNotSupported = errors.New("language not supported")

// Options configures the behavior of a Get call.
type Options struct {
	Fallback language.Tag // language to try when the requested one is missing
	Arg      any          // data passed to template execution (TextTemplateI18n only)
}

// Option is a functional option for Get methods.
type Option func(*Options)

// WithFallback sets the fallback language tag.
func WithFallback(fallback language.Tag) Option {
	return func(o *Options) {
		o.Fallback = fallback
	}
}

// WithArg sets the data argument passed to template execution in
// [TextTemplateI18n.Get].
func WithArg(arg any) Option {
	return func(o *Options) {
		o.Arg = arg
	}
}

// DefaultOptions is the baseline configuration copied by every Get call.
var DefaultOptions = &Options{
	Fallback: language.English,
}

// I18n is the common interface for retrieving localized strings.
type I18n interface {
	Get(lang language.Tag, options ...Option) (string, error)
}

// SimpleI18n maps language tags directly to static strings.
type SimpleI18n struct {
	data map[language.Tag]string
}

// NewSimpleI18n creates a [SimpleI18n] from a pre-populated language→string map.
func NewSimpleI18n(data map[language.Tag]string) *SimpleI18n {
	return &SimpleI18n{
		data: data,
	}
}

// Get returns the string for lang, falling back to [Options.Fallback] if
// lang is not found. It returns [ErrLanguageNotSupported] if neither exists.
func (i *SimpleI18n) Get(lang language.Tag, options ...Option) (string, error) {
	opts := *DefaultOptions
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

// TextTemplateI18n maps language tags to [text/template] templates, enabling
// dynamic content interpolation.
type TextTemplateI18n struct {
	data map[language.Tag]*template.Template
}

// NewTextTemplateI18n creates an empty [TextTemplateI18n].
// Use [TextTemplateI18n.Add] or [TextTemplateI18n.MustAdd] to register
// language templates.
func NewTextTemplateI18n() *TextTemplateI18n {
	return &TextTemplateI18n{
		data: make(map[language.Tag]*template.Template),
	}
}

// MustAdd registers a template for lang, panicking on parse error.
// It returns the receiver for fluent chaining during package initialization.
func (i *TextTemplateI18n) MustAdd(lang language.Tag, tpl string) *TextTemplateI18n {
	i.data[lang] = template.Must(template.New("").Parse(tpl))
	return i
}

// Add registers a template for lang, returning any parse error.
func (i *TextTemplateI18n) Add(lang language.Tag, tpl string) error {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	i.data[lang] = t
	return nil
}

// Get executes the template for lang (or the fallback) with the argument
// supplied via [WithArg]. It returns [ErrLanguageNotSupported] if neither
// the requested nor fallback language is registered.
func (i *TextTemplateI18n) Get(lang language.Tag, options ...Option) (string, error) {
	opts := *DefaultOptions
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
