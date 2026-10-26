// Package validator owns the shared, immutable-after-startup validation engine.
package validator

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	v "github.com/go-playground/validator/v10"
	zhtranslations "github.com/go-playground/validator/v10/translations/zh"
)

// Errors keeps API field keys separate from translated display names.
type Errors map[string]string

func (e Errors) Error() string {
	// Do not include submitted values in errors or logs.
	keys := make([]string, 0, len(e))
	for key := range e {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	messages := make([]string, 0, len(e))
	for _, key := range keys {
		messages = append(messages, key+": "+e[key])
	}
	return strings.Join(messages, "; ")
}

var once sync.Once
var engine *v.Validate
var translator ut.Translator
var initErr error

// Init registers rules and translations once, before serving requests.
func Init() error {
	once.Do(func() {
		engine = v.New(v.WithRequiredStructEnabled())
		engine.RegisterTagNameFunc(func(f reflect.StructField) string {
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				return f.Name
			}
			return name
		})
		locale := zh.New()
		translator, _ = ut.New(locale, locale).GetTranslator("zh")
		initErr = zhtranslations.RegisterDefaultTranslations(engine, translator)
	})
	return initErr
}

// Struct validates request fields using registered rules.
func Struct(value any) error {
	if engine == nil {
		return errors.New("validator is not initialized")
	}
	if initErr != nil {
		return initErr
	}
	err := engine.Struct(value)
	if err != nil {
		var fields v.ValidationErrors
		if !errors.As(err, &fields) {
			return err
		}
		out := Errors{}
		for _, field := range fields {
			key := field.Namespace()
			if i := strings.IndexByte(key, '.'); i >= 0 {
				key = key[i+1:]
			}
			message := field.Translate(translator)
			label := fieldLabel(reflect.TypeOf(value), field.StructNamespace())
			if label != "" {
				message = strings.Replace(message, field.Field(), label, 1)
			}
			if _, exists := out[key]; !exists {
				out[key] = message
			}
		}
		return out
	}
	return nil
}

func fieldLabel(t reflect.Type, namespace string) string {
	parts := strings.Split(namespace, ".")
	for index, name := range parts[1:] {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return ""
		}
		if i := strings.IndexByte(name, '['); i >= 0 {
			name = name[:i]
		}
		field, ok := t.FieldByName(name)
		if !ok {
			return ""
		}
		t = field.Type
		if index == len(parts)-2 {
			return field.Tag.Get("label")
		}
	}
	return ""
}
