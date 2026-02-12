package render

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
)

//go:embed fonts/OpenSans-Regular.woff2
var openSansRegular []byte

//go:embed fonts/OpenSans-Italic.woff2
var openSansItalic []byte

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
