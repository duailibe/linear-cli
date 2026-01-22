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

func TestParseUploadsLinksPrefersTitleWithoutExtension(t *testing.T) {
	text := "See [lin-attachment](https://uploads.linear.app/abc/def/ghi)"
	attachments := parseUploadsLinks(text)
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].FileName != "lin-attachment" {
		t.Fatalf("expected filename from title, got %s", attachments[0].FileName)
	}
}

func TestParseUploadsBodyData(t *testing.T) {
	bodyData := `{"type":"doc","content":[{"type":"file","attrs":{"href":"https://uploads.linear.app/abc/def/ghi","name":"report.csv"}}]}`
	attachments := parseUploadsBodyData(bodyData)
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Title != "report.csv" {
		t.Fatalf("expected title from bodyData, got %s", attachments[0].Title)
	}
	if attachments[0].FileName != "report.csv" {
		t.Fatalf("expected filename from bodyData, got %s", attachments[0].FileName)
	}
}
