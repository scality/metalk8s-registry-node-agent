package domain

import (
	"github.com/scality/go-errors"
)

var (
	ErrUnknown               error = errors.New("Unknown Error")
	ErrNotFound              error = errors.New("Not Found Error")
	ErrInternal              error = errors.New("Internal Error")
	ErrConflict              error = errors.New("Conflict Error")
	ErrBusinessRuleViolation error = errors.New("Business Rule Violation Error")
)

var (
	ErrStorageProviderInit                  error = errors.New("Init Error")
	ErrStorageProviderNotFound              error = errors.New("Not Found Error")
	ErrStorageProviderInternal              error = errors.New("Internal Error")
	ErrStorageProviderBusinessRuleViolation error = errors.New("Business Rule Violation Error")
)

var (
	ErrSessionInitializerInternal error = errors.New("Internal Error")
	ErrSessionInitializerNotFound error = errors.New("Not Found Error")
)

var (
	ErrPartUploaderNotFound error = errors.New("Not Found Error")
)

var (
	ErrSessionRemoverNotFound error = errors.New("Not Found Error")
)

var (
	ErrGetExternalSolutionArchiveNotFound error = errors.New("Not Found Error")
	ErrGetExternalSolutionArchiveInternal error = errors.New("Internal Error")
)

var (
	ErrExternalDownloaderInternal error = errors.New("Internal Error")
	ErrExternalDownloaderNotFound error = errors.New("Not Found Error")
)

var (
	ErrMountSolutionArchiveInternal       error = errors.New("Internal Error")
	ErrMountSolutionArchiveIncorrectMount error = errors.New("Incorrect Mount Point")
	ErrMountSolutionArchiveNotEmptyDir    error = errors.New("Not Empty Directory")
	ErrMountSolutionArchiveInvalidISO     error = errors.New("Invalid ISO File")
)

var (
	ErrHandlerBadRequest                 error = errors.New("Bad Request Error")
	ErrHandlerMissingRequestParameter    error = errors.New("Missing Request Parameter Error")
	ErrHandlerMissingRequestHeader       error = errors.New("Missing Request Header Error")
	ErrHandlerInvalidRequestHeaderFormat error = errors.New("Invalid Request Header Format Error")
)

var (
	ErrFileEventsInternal error = errors.New("Internal Error")
)

var (
	ErrSolutionArchiveCleanerInternal error = errors.New("Internal Error")
)
