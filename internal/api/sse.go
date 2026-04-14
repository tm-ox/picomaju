package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

// SSEMergeFragment writes a datastar merge-fragments SSE event that replaces
// the element matched by the fragment's root id.
func SSEMergeFragment(ctx context.Context, w http.ResponseWriter, c templ.Component) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		return err
	}

	fmt.Fprint(w, "event: datastar-merge-fragments\n")
	scanner := bufio.NewScanner(&buf)
	first := true
	for scanner.Scan() {
		if first {
			fmt.Fprintf(w, "data: fragments %s\n", scanner.Text())
			first = false
		} else {
			fmt.Fprintf(w, "data: %s\n", scanner.Text())
		}
	}
	if first {
		fmt.Fprint(w, "data: fragments \n")
	}
	fmt.Fprint(w, "\n")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}
