package i18n_test

import (
	"fmt"
	"log"

	"golang.org/x/text/language"

	"github.com/saltfishpr/pkg/i18n"
)

func ExampleNew() {
	translator := i18n.New(i18n.Dictionary{
		language.English: {
			"greeting": "Hello",
			"welcome":  "Welcome, {{.Name}}!",
		},
		language.SimplifiedChinese: {
			"greeting": "你好",
		},
	})

	greeting, err := translator.T(language.SimplifiedChinese, "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(greeting)

	welcome, err := translator.F(language.SimplifiedChinese, "welcome", map[string]string{"Name": "Ada"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(welcome)

	// Output:
	// 你好
	// Welcome, Ada!
}

func ExampleWithFallback() {
	translator := i18n.New(i18n.Dictionary{
		language.SimplifiedChinese: {
			"greeting": "你好",
		},
	}, i18n.WithFallback(language.SimplifiedChinese))

	greeting, err := translator.T(language.Japanese, "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(greeting)

	// Output:
	// 你好
}
