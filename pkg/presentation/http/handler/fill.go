//nolint:dupl,wrapcheck,funlen,lll,revive // Duplication is fine, too much wraps missing
package handler

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type ContentRegexp struct {
	name       string
	regexp     *regexp.Regexp
	submatches int
	format     string
	example    string
}

// contentRangeRegexp is used to parse the Content-Range header.
var contentRangeRegexp = ContentRegexp{
	name: "Content-Range",
	// See RFC9110, 14.4. Content-Range
	regexp: regexp.MustCompile(
		`^bytes (?P<start>[0-9]{1,12})-(?P<end>[0-9]{1,12})\/(?P<total>[0-9]{1,12})$`,
	),
	submatches: 4,
	format:     "bytes <start>-<end>/<total>",
	example:    "bytes 0-19/20",
}

// rangeRegexp is used to parse the Range header.
var rangeRegexp = ContentRegexp{
	name: "Range",
	// See RFC9110, 14.2. Range
	regexp: regexp.MustCompile(
		`^bytes=(?P<start>[0-9]{1,12})-(?P<end>[0-9]{1,12})$`,
	),
	submatches: 3,
	format:     "bytes=<start>-<end>",
	example:    "bytes=0-19",
}

// parseRange() parses the Range header and returns the start,
// size and end values.
// nolint:revive
func parseRange(
	headerRange string,
) (int64, int64, int64, error) {
	return parseContentHeader(
		rangeRegexp,
		headerRange,
	)
}

// parseContentRange() parses the Content-Range header and returns the start,
// size and end values.
// nolint:revive
func parseContentRange(
	headerContentRange string,
) (int64, int64, int64, error) {
	return parseContentHeader(
		contentRangeRegexp,
		headerContentRange,
	)
}

func parseContentHeader(
	contentRegexp ContentRegexp,
	headerContent string,
) (int64, int64, int64, error) {
	var start, end, total int64
	var err error

	// Check if the Content-Range header is missing
	if headerContent == "" {
		err = errors.From(domain.ErrHandlerMissingRequestHeader).
			WithIdentifier(400002).
			WithDetailf("header %s is missing", contentRegexp.name).
			Throw()

		return start, end, total, err
	}

	matched := contentRegexp.regexp.FindStringSubmatch(headerContent)

	if len(matched) != contentRegexp.submatches {
		err = errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetailf("header %s is not in the expected format", contentRegexp.name).
			WithProperty("received_header", headerContent).
			WithProperty("expected_format", contentRegexp.format).
			WithProperty("example_header", contentRegexp.example).
			WithProperty("validation_regex", contentRegexp.regexp.String()).
			Throw()

		return start, end, total, err
	}

	problems := make(map[string]any)

	start, err = strconv.ParseInt(matched[1], 10, 64)
	if err != nil {
		problems["problem_invalid_start"] = err.Error()
	}

	end, err = strconv.ParseInt(matched[2], 10, 64)
	if err != nil {
		problems["problem_invalid_end"] = err.Error()
	}

	if start < 0 {
		problems["problem_invalid_start_less_than_zero"] = fmt.Sprintf("start < 0: %d", start)
	}

	if start > end {
		problems["problem_invalid_start_greater_than_end"] = fmt.Sprintf("start > end: %d > %d", start, end)
	}

	if contentRegexp.submatches == 4 {
		total, err = strconv.ParseInt(matched[3], 10, 64)
		if err != nil {
			problems["problem_invalid_total"] = err.Error()
		}
		size := end - start + 1
		if size > total {
			problems["problem_invalid_size_greater_than_total"] = fmt.Sprintf("size > total: %d > %d", size, total)
		}
	}

	if len(problems) > 0 {
		err = errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetailf("header %s is not in the expected format", contentRegexp.name).
			WithProperties(problems).
			WithProperty("received_header", headerContent).
			WithProperty("expected_format", contentRegexp.format).
			WithProperty("example_header", contentRegexp.example).
			WithProperty("validation_regex", contentRegexp.regexp.String()).
			Throw()

		start, end, total = 0, 0, 0

		return start, end, total, err
	}

	return start, end, total, nil
}

// fillInstanceFromPropertiesMap generates an instance string from a map[string]any.
func fillInstanceFromPropertiesMap(
	dst **string,
	src map[string]any,
) {
	b := bytes.NewBuffer(nil)

	if dst != nil && *dst != nil {
		b.WriteString(**dst)
		b.WriteString(":")
	}

	for key, value := range src {
		if !strings.HasPrefix(key, "problem_") {
			fmt.Fprintf(b, "%s='%v',", key, value)
		}
	}

	if b.Len() > 0 {
		instance := b.String()[:b.Len()-1]
		*dst = &instance
	}
}
