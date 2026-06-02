package library

import (
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// contentFilter represents a filter for the content listing.
type ContentFilter func(fs.DirEntry) bool

// filterContent apply the given filter to a list of entries and returns the
// only the entries that match the filter.
func FilterContent(
	entries []fs.DirEntry,
	filter ContentFilter,
) []fs.DirEntry {
	filteredEntries := make([]fs.DirEntry, 0)

	for _, entry := range entries {
		if filter(entry) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	return filteredEntries
}

/*
 * Filters
 */

func NewRegexNormalFileFilter(
	regex *regexp.Regexp,
) ContentFilter {
	if regex == nil {
		return normalFileFilter
	}

	regexFilter := newRegexFilter(regex)

	return func(entry fs.DirEntry) bool {
		return normalFileFilter(entry) && regexFilter(entry)
	}
}

// listDirContentNames lists the content of a given location of the storage,
// applies a filter and returns the names of the entries that match the filter.
func ListDirContentNames(
	location string,
	filter ContentFilter,
) ([]string, error) {
	entries, err := ListDirContent(location, filter)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return ExtractNames(entries), nil
}

func BucketFilter(
	entry fs.DirEntry,
) bool {
	return entry.IsDir() && strings.HasPrefix(entry.Name(), FileSystemBucketPrefix)
}

func normalFileFilter(
	entry fs.DirEntry,
) bool {
	return !entry.IsDir() &&
		!MultipartMetaFilter(entry) &&
		!multipartPartsFilter(entry) &&
		!multipartRecipientFilter(entry)
}

func MultipartMetaFilter(
	entry fs.DirEntry,
) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), FileSystemMultipartMetaSuffix)
}

func multipartPartsFilter(
	entry fs.DirEntry,
) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), FileSystemMultipartPartsSuffix)
}

func multipartRecipientFilter(
	entry fs.DirEntry,
) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), FileSystemMultipartRecipientSuffix)
}

func newRegexFilter(regex *regexp.Regexp) ContentFilter {
	return func(entry fs.DirEntry) bool {
		return regex.MatchString(entry.Name())
	}
}

// ExtractNames extracts the names of a list of entries.
func ExtractNames(
	entries []fs.DirEntry,
) []string {
	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}

// listDirContent lists the content of a given location of the storage,
// applies a filter and returns the entries that match the filter.
func ListDirContent(
	location string,
	filter ContentFilter,
) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(location)
	if err != nil {
		return nil, errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(7),
			errors.WithDetail("unexpected error while listing the content"),
			errors.WithProperty("location_path", location),
			errors.CausedBy(err),
		)
	}

	return FilterContent(entries, filter), nil
}
