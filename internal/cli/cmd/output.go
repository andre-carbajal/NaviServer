package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"naviserver/pkg/sdk"
	"net"
	"net/url"
	"os"
	"strings"
)

const (
	exitCodeUnknown    = 1
	exitCodeValidation = 2
	exitCodeNetwork    = 3
	exitCodeAPI        = 4
)

type cliErrorKind int

const (
	cliErrorValidation cliErrorKind = iota
	cliErrorOperation
)

type cliError struct {
	kind cliErrorKind
	msg  string
	err  error
}

func (e *cliError) Error() string {
	if e.msg != "" && e.err != nil {
		return fmt.Sprintf("%s: %v", e.msg, e.err)
	}
	if e.msg != "" {
		return e.msg
	}
	if e.err != nil {
		return e.err.Error()
	}
	return "unknown error"
}

func (e *cliError) Unwrap() error {
	return e.err
}

func newValidationError(msg string) error {
	return &cliError{kind: cliErrorValidation, msg: msg}
}

func isJSONOutput() bool {
	return strings.EqualFold(outputFormat, "json")
}

func printJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func printTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	fmt.Fprintln(os.Stdout, renderTableRow(headers, widths))
	fmt.Fprintln(os.Stdout, renderTableSeparator(widths))
	for _, row := range rows {
		cells := make([]string, len(headers))
		copy(cells, row)
		fmt.Fprintln(os.Stdout, renderTableRow(cells, widths))
	}
}

func renderTableRow(cells []string, widths []int) string {
	rendered := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		rendered[i] = fmt.Sprintf("%-*s", widths[i], cell)
	}
	return strings.Join(rendered, "  ")
}

func renderTableSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width)
	}
	return strings.Join(parts, "  ")
}

func formatMegabytes(sizeBytes int64) string {
	return fmt.Sprintf("%.2f", float64(sizeBytes)/1024/1024)
}

func printCommandError(err error) {
	if err == nil {
		return
	}

	if isJSONOutput() {
		jsonErr := printJSON(map[string]string{"error": err.Error()})
		if jsonErr == nil {
			return
		}
	}

	fmt.Fprintln(os.Stderr, "Error:", err)
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}

	if cErr, ok := errors.AsType[*cliError](err); ok && cErr.kind == cliErrorValidation {
		return exitCodeValidation
	}

	if _, ok := errors.AsType[*sdk.APIError](err); ok {
		return exitCodeAPI
	}

	if _, ok := errors.AsType[*url.Error](err); ok {
		return exitCodeNetwork
	}

	if _, ok := errors.AsType[net.Error](err); ok {
		return exitCodeNetwork
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "unknown command") ||
		strings.Contains(errMsg, "unknown flag") ||
		strings.Contains(errMsg, "accepts") ||
		strings.Contains(errMsg, "requires") ||
		strings.Contains(errMsg, "invalid argument") ||
		strings.Contains(errMsg, "flag needs an argument") {
		return exitCodeValidation
	}

	return exitCodeUnknown
}
