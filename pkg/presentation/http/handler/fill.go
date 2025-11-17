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

// contentRangeRegexp is a regular expression to parse the Content-Range header.
var contentRangeRegexp = regexp.MustCompile(
	`bytes (?P<begin>[0-9]{1,12})-(?P<end>[0-9]{1,12})\/(?P<total>[0-9]{1,12})`,
)

const expectedContentRangeSubmatches = 4

// parseContentRange() parses the UploadChunkRange object and returns the start,
// size and end values.
// nolint:revive
func parseContentRange(
	contentRange string,
) (int64, int64, int64, error) {
	var start, end, total int64
	var err error

	// Check if the Content-Range header is missing
	if contentRange == "" {
		err = errors.From(domain.ErrHandlerMissingRequestHeader).
			WithIdentifier(400002).
			WithDetail("Content-Range header is missing.").
			Throw()

		return start, end, total, err
	}

	matched := contentRangeRegexp.FindStringSubmatch(contentRange)

	if len(matched) != expectedContentRangeSubmatches {
		err = errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetail("Content-Range header is not in the expected format.").
			WithProperty("received_content_range", contentRange).
			WithProperty("expected_content_range_format", "bytes <start>-<end>/<total>").
			WithProperty("example_content_range", "bytes 0-19/20").
			WithProperty("validation_regex", contentRangeRegexp.String()).
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

	total, err = strconv.ParseInt(matched[3], 10, 64)
	if err != nil {
		problems["problem_invalid_total"] = err.Error()
	}

	if start < 0 {
		problems["problem_invalid_start_less_than_zero"] = fmt.Sprintf("start < 0: %d", start)
	}

	if start > end {
		problems["problem_invalid_start_greater_than_end"] = fmt.Sprintf("start > end: %d > %d", start, end)
	}

	size := end - start + 1
	if size > total {
		problems["problem_invalid_size_greater_than_total"] = fmt.Sprintf("size > total: %d > %d", size, total)
	}

	if len(problems) > 0 {
		err = errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetail("Content-Range header is not in the expected format.").
			WithProperties(problems).
			WithProperty("received_content_range", contentRange).
			WithProperty("expected_content_range_format", "bytes <start>-<end>/<total>").
			WithProperty("example_content_range", "bytes 0-19/20").
			WithProperty("validation_regex", contentRangeRegexp.String()).
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

	if dst != nil {
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
