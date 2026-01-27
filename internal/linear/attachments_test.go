package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestIssueUploadsIncludesCommentUploadsWhenAPIAttachmentsPresent(t *testing.T) {
	bodyData := `{"type":"doc","content":[{"type":"file","attrs":{"href":"https://uploads.linear.app/abc/def/file","name":"Stockout_Reasons_History.sql"}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req gqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch {
		case strings.Contains(req.Query, "attachments(first:"):
			resp := map[string]any{
				"data": map[string]any{
					"issue": map[string]any{
						"description": "",
						"attachments": map[string]any{
							"nodes": []map[string]any{
								{
									"id":        "att-1",
									"title":     "Slack thread",
									"url":       "https://example.com/thread",
									"createdAt": "2026-01-01T00:00:00Z",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case strings.Contains(req.Query, "comments(first:"):
			resp := map[string]any{
				"data": map[string]any{
					"issue": map[string]any{
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"id":        "comment-1",
									"body":      "",
									"bodyData":  bodyData,
									"createdAt": "2026-01-01T00:00:00Z",
									"user": map[string]any{
										"name":  "Tester",
										"email": "test@example.com",
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	client := &Client{
		apiURL: srv.URL,
		http:   srv.Client(),
	}

	attachments, err := client.IssueUploads(context.Background(), "issue-1", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found := false
	for _, attachment := range attachments {
		if attachment.FileName == "Stockout_Reasons_History.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected comment upload filename in uploads")
	}
}

func TestIssueUploadsFiltersNonUploadAttachments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req gqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch {
		case strings.Contains(req.Query, "attachments(first:"):
			resp := map[string]any{
				"data": map[string]any{
					"issue": map[string]any{
						"description": "",
						"attachments": map[string]any{
							"nodes": []map[string]any{
								{
									"id":        "att-1",
									"title":     "Slack thread",
									"url":       "https://example.com/thread",
									"createdAt": "2026-01-01T00:00:00Z",
								},
								{
									"id":        "att-2",
									"title":     "Upload",
									"url":       "https://uploads.linear.app/abc/def/file",
									"createdAt": "2026-01-01T00:00:00Z",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case strings.Contains(req.Query, "comments(first:"):
			resp := map[string]any{
				"data": map[string]any{
					"issue": map[string]any{
						"comments": map[string]any{
							"nodes": []map[string]any{},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	client := &Client{
		apiURL: srv.URL,
		http:   srv.Client(),
	}

	attachments, err := client.IssueUploads(context.Background(), "issue-1", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(attachments))
	}
	if attachments[0].URL != "https://uploads.linear.app/abc/def/file" {
		t.Fatalf("expected upload attachment, got %s", attachments[0].URL)
	}
}
