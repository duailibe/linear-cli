package linear

import "testing"

func TestParseUploadsLinks(t *testing.T) {
	text := "See [file.sql](https://uploads.linear.app/abc/def/ghi)"
	attachments := parseUploadsLinks(text)
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Title != "file.sql" {
		t.Fatalf("expected title from markdown")
	}
	if attachments[0].FileName != "file.sql" {
		t.Fatalf("expected filename from title, got %s", attachments[0].FileName)
	}
}

func TestSanitizeFileNameDots(t *testing.T) {
	if got := sanitizeFileName("."); got != "attachment" {
		t.Fatalf("expected attachment, got %s", got)
	}
	if got := sanitizeFileName(".."); got != "attachment" {
		t.Fatalf("expected attachment, got %s", got)
	}
}
