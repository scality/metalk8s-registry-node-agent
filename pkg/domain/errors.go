package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	ErrUnknownError error = &Error{
		Title:    "Unknown Error",
		Status:   000,
		Subclass: 000,
	}
	ErrNotFoundError error = &Error{
		Title:    "Not Found Error",
		Status:   404,
		Subclass: 404000,
	}
	ErrInternalError error = &Error{
		Title:    "Internal Error",
		Status:   500,
		Subclass: 500000,
	}
	ErrConflictError error = &Error{
		Title:    "Conflict Error",
		Status:   409,
		Subclass: 409000,
	}
	ErrBusinessRuleViolationError error = &Error{
		Title:    "Business Rule Violation Error",
		Status:   422,
		Subclass: 422001,
	}
)

var (
	ErrStorageProviderInitError error = &Error{
		Title:    "Init Error",
		Status:   500,
		Subclass: 500000,
	}
	ErrStorageProviderNotFoundError error = &Error{
		Title:    "Not Found Error",
		Status:   404,
		Subclass: 404000,
	}
	ErrStorageProviderInternalError error = &Error{
		Title:    "Internal Error",
		Status:   500,
		Subclass: 500000,
	}
	ErrStorageProviderBusinessRuleViolation error = &Error{
		Title:    "Business Rule Violation Error",
		Status:   422,
		Subclass: 422001,
	}
)

var (
	ErrSessionInitializerInternalError error = &Error{
		Title:    "Internal Error",
		Status:   500,
		Subclass: 500000,
	}
	ErrSessionInitializerNotFoundError error = &Error{
		Title:    "Not Found Error",
		Status:   404,
		Subclass: 404000,
	}
)

var (
	ErrPartUploaderNotFoundError error = &Error{
		Title:    "Not Found Error",
		Status:   404,
		Subclass: 404000,
	}
)

var (
	ErrSessionRemoverNotFoundError error = &Error{
		Title:    "Not Found Error",
		Status:   404,
		Subclass: 404000,
	}
)
var (
	ErrHandlerBadRequestError error = &Error{
		Title:    "Bad Request Error",
		Status:   400,
		Subclass: 400000,
	}
	ErrHandlerMissingRequestParameterError error = &Error{
		Title:    "Missing Request Parameter Error",
		Status:   400,
		Subclass: 400003,
	}
	ErrHandlerMissingRequestHeaderError error = &Error{
		Title:    "Missing Request Header Error",
		Status:   400,
		Subclass: 400002,
	}
	ErrHandlerInvalidRequestHeaderFormatError error = &Error{
		Title:    "Invalid Request Header Format Error",
		Status:   400,
		Subclass: 400006,
	}
)

// Trace is a type that represents a trace of the error.
type Trace struct {
	// Function is the function where the error occurred
	Function string `json:"function,omitempty" yaml:"function,omitempty"`

	// File is the file where the error occurred
	File string `json:"file,omitempty" yaml:"file,omitempty"`

	// Line is the line where the error occurred
	Line int `json:"line,omitempty" yaml:"line,omitempty"`

	// Timestamp is the time when trace was generated
	Timestamp time.Time `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
}

// Error is a RFC 7807 compatible error implementation for Artesca API.
type Error struct {
	// Title as defined in RFC 7807
	Title string `json:"title" yaml:"title"`

	// Type as defined in RFC 7807
	Type string `json:"type" yaml:"type"`

	// Status must be one of HTTP status code
	Status int `json:"status" yaml:"status"`

	// Subclass a specified in apierrors package
	Subclass int `json:"subclass,omitempty" yaml:"subclass,omitempty"`

	// Detail as defined in RFC 7807
	Detail string `json:"detail,omitempty" yaml:"detail,omitempty"`

	// Instance as defined in RFC 7807
	Instance string `json:"instance,omitempty" yaml:"instance,omitempty"`

	// Properties is a map of additional properties
	Properties map[string]any `json:"properties,omitempty" yaml:"properties,omitempty"`

	// Cause is the error that caused this error
	Cause error `json:"-" yaml:"-"`

	// Trace is the trace of the error
	Stack []*Trace `json:"stack,omitempty" yaml:"stack,omitempty"`
}

// Is allow to compare this error with another received as parameter.
func (e *Error) Is(err error) bool {
	other := new(Error)
	if ok := errors.As(err, &other); !ok {
		return false
	}

	return e.Title == other.Title && e.Status == other.Status && e.Subclass == other.Subclass
}

// Unwrap implements the Wrapper interface.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

// Error implements the error interface.
// nolint: revive // No way to get wrong here.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	b := bytes.NewBuffer(nil)

	fmt.Fprintf(
		b,
		"%s (%d):",
		strings.ToLower(e.Title),
		e.Subclass,
	)

	if e.Detail != "" {
		fmt.Fprintf(
			b,
			" %s:",
			strings.TrimSuffix(strings.ToLower(e.Detail), "."),
		)
	}

	if e.Instance != "" {
		fmt.Fprintf(b, " instance='%v',", e.Instance)
	}

	for k, v := range e.Properties {
		fmt.Fprintf(b, " %s='%v',", k, v)
	}

	if len(e.Stack) > 0 {
		tail := e.Stack[len(e.Stack)-1]

		if tail != nil {
			fmt.Fprintf(
				b,
				" at=(func='%s', file='%s', line='%d'),",
				path.Base(tail.Function),
				filepath.Base(tail.File),
				tail.Line,
			)
		}
	}

	if e.Cause != nil {
		fmt.Fprintf(b, " caused by: %v", e.Cause.Error())
	}

	return string(bytes.TrimSuffix(bytes.TrimSuffix(b.Bytes(), []byte(",")), []byte(":")))
}

// String implements the fmt.Stringer interface.
func (e *Error) String() string {
	if e == nil {
		return ""
	}

	b := bytes.NewBuffer(nil)
	json.NewEncoder(b).Encode(e) // nolint: errcheck // No way to get wrong here.

	return b.String()
}

func FromTemplate(err error) *Error {
	var t *Error

	ok := errors.As(err, &t)
	if !ok {
		t, _ = ErrUnknownError.(*Error)
	}

	e := &Error{
		Title:    t.Title,
		Type:     t.Type,
		Status:   t.Status,
		Subclass: t.Subclass,
	}

	if !ok {
		e.Cause = err
	}

	return e
}

func Intercept(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}

	return FromTemplate(err)
}

func Stamp(err error) error {
	trace := trace()

	return Intercept(err).throw(trace)
}

func (e *Error) WithDetail(detail string) *Error {
	e.Detail = detail

	return e
}

func (e *Error) AtInstance(instance string) *Error {
	e.Instance = instance

	return e
}

func (e *Error) WithProperties(properties map[string]any) *Error {
	if e.Properties == nil {
		e.Properties = properties

		return e
	}

	for k, v := range properties {
		e.Properties[k] = v
	}

	return e
}

func (e *Error) AddProperty(key string, value any) *Error {
	if e.Properties == nil {
		e.Properties = make(map[string]any)
	}

	e.Properties[key] = value

	return e
}

func (e *Error) CausedBy(err error) *Error {
	e.Cause = err

	return e
}

func (e *Error) throw(trace *Trace) error {
	if trace == nil {
		return e
	}

	e.Stack = append([]*Trace{trace}, e.Stack...)

	return e
}

// nolint: revive // The couple of public/private methods is a design option.
func (e *Error) Throw() error {
	trace := trace()

	return e.throw(trace)
}

func trace() *Trace {
	// nolint: mnd // 2 is the depth of the caller.
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return nil
	}

	return &Trace{
		Function:  fn.Name(),
		File:      file,
		Line:      line,
		Timestamp: time.Now().UTC(),
	}
}
