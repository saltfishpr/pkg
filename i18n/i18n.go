package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"golang.org/x/text/language"
)

var ErrLanguageNotSupported = errors.New("language not supported")

type Options struct {
	Fallback language.Tag
	Arg      any // argument for [Get] method
}

type Option func(*Options)

func WithFallback(fallback language.Tag) Option {
	return func(o *Options) {
		o.Fallback = fallback
	}
}

func WithArg(arg any) Option {
	return func(o *Options) {
		o.Arg = arg
	}
}

var DefaultOptions = &Options{
	Fallback: language.English,
}

type I18n interface {
	Get(lang language.Tag, options ...Option) (string, error)
}

type SimpleI18n struct {
	data map[language.Tag]string
}

func NewSimpleI18n(data map[language.Tag]string) *SimpleI18n {
	return &SimpleI18n{
		data: data,
	}
}

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

type TextTemplateI18n struct {
	data map[language.Tag]*template.Template
}

func NewTextTemplateI18n() *TextTemplateI18n {
	return &TextTemplateI18n{
		data: make(map[language.Tag]*template.Template),
	}
}

func (i *TextTemplateI18n) MustAdd(lang language.Tag, tpl string) *TextTemplateI18n {
	i.data[lang] = template.Must(template.New("").Parse(tpl))
	return i
}

func (i *TextTemplateI18n) Add(lang language.Tag, tpl string) error {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	i.data[lang] = t
	return nil
}

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
