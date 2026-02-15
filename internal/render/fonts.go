package render

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed fonts/OpenSans-Regular.woff2
var openSansRegular []byte

//go:embed fonts/OpenSans-Italic.woff2
var openSansItalic []byte

//go:embed fonts/OpenSans-Regular.ttf
var openSansRegularTTF []byte

//go:embed fonts/OpenSans-Bold.ttf
var openSansBoldTTF []byte

//go:embed fonts/OpenSans-Italic.ttf
var openSansItalicTTF []byte

//go:embed fonts/OpenSans-BoldItalic.ttf
var openSansBoldItalicTTF []byte

//go:embed fonts/NotoSansSymbols2-Regular.ttf
var notoSansSymbols2TTF []byte

// writeFontsToDir writes embedded TTF font files to a directory for Typst.
func writeFontsToDir(dir string) error {
	fonts := []struct {
		name string
		data []byte
	}{
		{"OpenSans-Regular.ttf", openSansRegularTTF},
		{"OpenSans-Bold.ttf", openSansBoldTTF},
		{"OpenSans-Italic.ttf", openSansItalicTTF},
		{"OpenSans-BoldItalic.ttf", openSansBoldItalicTTF},
		{"NotoSansSymbols2-Regular.ttf", notoSansSymbols2TTF},
	}
	for _, f := range fonts {
		if err := os.WriteFile(filepath.Join(dir, f.name), f.data, 0644); err != nil {
			return fmt.Errorf("writing font %s: %w", f.name, err)
		}
	}
	return nil
}

// fontFaceCSS returns @font-face declarations with base64-embedded OpenSans.
func fontFaceCSS() string {
	var buf strings.Builder
	writeFontFace(&buf, "Open Sans", "normal", openSansRegular)
	writeFontFace(&buf, "Open Sans", "italic", openSansItalic)
	return buf.String()
}

func writeFontFace(buf *strings.Builder, family, style string, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	fmt.Fprintf(buf, `@font-face {
  font-family: '%s';
  font-style: %s;
  font-weight: 100 900;
  font-display: swap;
  src: url(data:font/woff2;base64,%s) format('woff2');
}
`, family, style, encoded)
}
