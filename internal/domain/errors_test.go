package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainError_ErrorFormatting(t *testing.T) {
	err := NewDomainError(ErrorCodePRMerged, "")
	require.EqualError(t, err, "PR_MERGED")

	err2 := NewDomainError(ErrorCodeNotAssigned, "reviewer is not assigned")
	require.EqualError(t, err2, "NOT_ASSIGNED: reviewer is not assigned")
}

func TestAsDomainError_ExtractsCodeAndMessage(t *testing.T) {
	baseErr := NewDomainError(ErrorCodeNoCandidate, "no active replacement candidate")

	wrapped := errors.New("some wrapper")
	wrapped = &withCause{msg: wrapped.Error(), cause: baseErr}

	de, ok := AsDomainError(wrapped)
	require.True(t, ok)
	require.NotNil(t, de)
	require.Equal(t, ErrorCodeNoCandidate, de.Code)
	require.Equal(t, "no active replacement candidate", de.Msg)
}

type withCause struct {
	msg   string
	cause error
}

func (e *withCause) Error() string {
	return e.msg
}

func (e *withCause) Unwrap() error {
	return e.cause
}
