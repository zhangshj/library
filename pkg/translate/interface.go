// Package translate defines the unified interface for machine translation drivers.
package translate

import (
	"context"
	"fmt"
)

// ErrInvalidInput indicates the input parameters are invalid.
var ErrInvalidInput = fmt.Errorf("invalid input")

// Translator defines the contract for text translation across cloud providers.
type Translator interface {
	// Translate converts text from source language to target language.
	// Language codes follow BCP 47 / ISO 639-1, e.g. "zh", "en", "ja".
	Translate(ctx context.Context, text string, sourceLang, targetLang string) (string, error)

	// TranslateBatch converts a batch of texts from source language to target language.
	// It preserves the order of input texts in the output slice.
	// Language codes follow BCP 47 / ISO 639-1, e.g. "zh", "en", "ja".
	TranslateBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error)
}
