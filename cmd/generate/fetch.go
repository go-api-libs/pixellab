package main

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"
)

const (
	originalURL = "https://api.pixellab.ai/v2/openapi.json"
	llmsURL     = "https://api.pixellab.ai/v2/llms.txt"
	llmsPath    = "api/llms.txt"
)

// isClaudeCode is set in a Claude Code session, whose sandbox may not reach PixelLab: the copies in api/ are used as
// they are.
var isClaudeCode = os.Getenv("CLAUDECODE") != ""

// fetchUpstream replaces the copies of PixelLab's specification and LLM documentation in api/ with their newest
// versions. The specification is indented, so that a diff of it shows what changed upstream.
func fetchUpstream(ctx context.Context) error {
	if isClaudeCode {
		return nil
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		b, err := fetch(ctx, originalURL)
		if err != nil {
			return err
		}

		v := jsontext.Value(b)
		if err := v.Indent(jsontext.WithIndent("  ")); err != nil {
			return fmt.Errorf("%s: %w", originalURL, err)
		}

		return os.WriteFile(originalPath, append(v, '\n'), 0o644)
	})

	eg.Go(func() error {
		b, err := fetch(ctx, llmsURL)
		if err != nil {
			return err
		}

		return os.WriteFile(llmsPath, b, 0o644)
	})

	return eg.Wait()
}

func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, rsp.Status)
	}

	return io.ReadAll(rsp.Body)
}
