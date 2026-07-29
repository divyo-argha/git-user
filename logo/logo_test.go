package logo

import (
	"strings"
	"testing"
)

func TestNewSmallPixelLines_NotEmpty(t *testing.T) {
	if len(NewSmallPixelLines) == 0 {
		t.Fatal("NewSmallPixelLines should not be empty")
	}
}

func TestNewSmallPixelLines_Content(t *testing.T) {
	for i, line := range NewSmallPixelLines {
		if line == "" {
			t.Errorf("NewSmallPixelLines[%d] should not be empty", i)
		}
	}
}

func TestIsInlineGraphicsSupported(t *testing.T) {
	if IsInlineGraphicsSupported() {
		t.Fatal("IsInlineGraphicsSupported should return false")
	}
}

func TestGetTrimmedLogo(t *testing.T) {
	trimmed := GetTrimmedLogo()
	if len(trimmed) == 0 {
		t.Fatal("GetTrimmedLogo should return non-empty result")
	}
	for i, line := range trimmed {
		if strings.TrimSpace(line) == "" {
			t.Errorf("GetTrimmedLogo[%d] should not be empty after trim", i)
		}
	}
}

func TestGetTrimmedLogo_RemovesEmptyLines(t *testing.T) {
	result := GetTrimmedLogo()
	for _, line := range result {
		if strings.TrimSpace(line) == "" {
			t.Error("GetTrimmedLogo should remove empty/whitespace-only lines")
		}
	}
}

func TestGetTrimmedLogo_ContainsNewSmallPixelLines(t *testing.T) {
	result := GetTrimmedLogo()
	combinedResult := strings.Join(result, "")
	combinedOriginal := strings.Join(NewSmallPixelLines, "")
	if combinedResult == "" && combinedOriginal != "" {
		t.Error("GetTrimmedLogo should return at least some content from NewSmallPixelLines")
	}
}
