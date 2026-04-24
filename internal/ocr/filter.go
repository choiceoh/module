package ocr

import (
	"regexp"
	"strings"
)

// ExtractSerialCandidates returns uppercase alphanumeric substrings of
// reasonable length from each recognized result. OCR often picks up
// adjacent digits (the "NO" column number) as a prefix; regex extraction
// peels those off by matching a strict A-Z0-9 run.
func ExtractSerialCandidates(results []Result, minLen, maxLen int) []string {
	rePat := `[A-Z0-9]{` + itoa(minLen) + `,` + itoa(maxLen) + `}`
	re := regexp.MustCompile(rePat)
	seen := map[string]struct{}{}
	var out []string
	for _, r := range results {
		text := strings.ToUpper(strings.TrimSpace(r.Text))
		for _, m := range re.FindAllString(text, -1) {
			if _, dup := seen[m]; dup {
				continue
			}
			seen[m] = struct{}{}
			out = append(out, m)
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 4)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
