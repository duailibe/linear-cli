package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/duailibe/linear-cli/internal/linear"
)

type IssueAttachmentsCmd struct {
	IssueID   string `arg:"" name:"issue-id" help:"Issue ID"`
	Dir       string `help:"Directory to save attachments" default:"attachments"`
	Limit     int    `help:"Maximum number of comments to scan" default:"50"`
	Overwrite bool   `help:"Overwrite existing files"`
}

func (c *IssueAttachmentsCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}

	attachments, err := client.IssueAttachments(ctx, c.IssueID, c.Limit)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	if len(attachments) == 0 {
		_, _ = fmt.Fprintln(cmdCtx.deps.Out, "No attachments found")
		return nil
	}

	dir := c.Dir
	if dir == "" {
		dir = "attachments"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return exitError(1, fmt.Errorf("create dir: %w", err))
	}

	apiKey, _, _ := cmdCtx.resolveAPIKey()

	results := make([]attachmentDownload, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.URL == "" {
			continue
		}
		name := attachmentFileName(attachment)
		path := uniquePath(filepath.Join(dir, name), c.Overwrite)
		if err := downloadToFile(ctx, attachment.URL, path, apiKey, cmdCtx.global.Timeout); err != nil {
			return exitError(1, err)
		}
		results = append(results, attachmentDownload{
			Attachment: attachment,
			Path:       path,
		})
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(results)
	}
	rows := make([][]string, 0, len(results))
	for _, item := range results {
		rows = append(rows, []string{item.ID, item.Title, item.Path})
	}
	return out.PrintTable([]string{"ID", "Title", "Path"}, rows)
}

type attachmentDownload struct {
	linear.Attachment
	Path string `json:"path"`
}

func attachmentFileName(attachment linear.Attachment) string {
	if attachment.FileName != "" {
		return sanitizeFileName(attachment.FileName)
	}
	if attachment.Title != "" {
		return sanitizeFileName(attachment.Title)
	}
	if attachment.URL != "" {
		if parsed, err := url.Parse(attachment.URL); err == nil {
			base := filepath.Base(parsed.Path)
			if base != "." && base != "/" && base != "" {
				return sanitizeFileName(base)
			}
		}
	}
	return fmt.Sprintf("attachment-%s", attachment.ID)
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	if name == "" {
		return "attachment"
	}
	return name
}

func uniquePath(path string, overwrite bool) string {
	if overwrite {
		return path
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func downloadToFile(ctx context.Context, urlStr, path, apiKey string, timeout time.Duration) (err error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	if apiKey != "" && shouldSendAuth(parsed.Host) {
		req.Header.Set("Authorization", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".linear-attachment-*")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err = tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func shouldSendAuth(host string) bool {
	host = strings.ToLower(host)
	return strings.HasSuffix(host, "linear.app")
}
