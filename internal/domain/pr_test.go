package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewPullRequest_Valid(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)
	require.NotNil(t, pr)

	require.Equal(t, "pr-1", pr.ID)
	require.Equal(t, "Add feature", pr.Name)
	require.Equal(t, "u1", pr.AuthorID)
	require.Equal(t, PRStatusOpen, pr.Status)
	require.Empty(t, pr.AssignedReviewers)
	require.False(t, pr.CreatedAt.IsZero())
	require.Nil(t, pr.MergedAt)
}

func TestNewPullRequest_InvalidParams(t *testing.T) {
	_, err := NewPullRequest("", "name", "u1")
	require.Error(t, err)

	_, err = NewPullRequest("pr-1", "", "u1")
	require.Error(t, err)

	_, err = NewPullRequest("pr-1", "name", "")
	require.Error(t, err)
}

func TestAssignReviewers_TooMany(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u3", "u4"})
	require.Error(t, err)
}

func TestAssignReviewers_AuthorInReviewers(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u1"})
	require.Error(t, err)
}

func TestAssignReviewers_DuplicateReviewers(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u2"})
	require.Error(t, err)
}

func TestAssignReviewers_OK(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u3"})
	require.NoError(t, err)
	require.Equal(t, []string{"u2", "u3"}, pr.AssignedReviewers)
}

func TestMarkMerged_Idempotent(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	require.Equal(t, PRStatusOpen, pr.Status)
	require.Nil(t, pr.MergedAt)

	pr.MarkMerged()
	require.Equal(t, PRStatusMerged, pr.Status)
	require.NotNil(t, pr.MergedAt)
	firstMergedAt := *pr.MergedAt

	time.Sleep(10 * time.Millisecond)
	pr.MarkMerged()
	require.Equal(t, PRStatusMerged, pr.Status)
	require.NotNil(t, pr.MergedAt)
	_ = firstMergedAt
}

func TestReplaceReviewer_OnMergedPR_ReturnsPRMergedError(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u3"})
	require.NoError(t, err)

	pr.Status = PRStatusMerged

	err = pr.ReplaceReviewer("u2", "u4")
	require.Error(t, err)

	de, ok := AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodePRMerged, de.Code)
}

func TestReplaceReviewer_NotAssigned_ReturnsNotAssignedError(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u3"})
	require.NoError(t, err)

	err = pr.ReplaceReviewer("u5", "u4")
	require.Error(t, err)

	de, ok := AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeNotAssigned, de.Code)
}

func TestReplaceReviewer_NewIsAuthor_Error(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2"})
	require.NoError(t, err)

	err = pr.ReplaceReviewer("u2", "u1")
	require.Error(t, err)

	_, ok := AsDomainError(err)
	require.False(t, ok)
}

func TestReplaceReviewer_NewAlreadyAssigned_Error(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u3"})
	require.NoError(t, err)

	err = pr.ReplaceReviewer("u2", "u3")
	require.Error(t, err)
	_, ok := AsDomainError(err)
	require.False(t, ok)
}

func TestReplaceReviewer_OK(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2", "u3"})
	require.NoError(t, err)

	err = pr.ReplaceReviewer("u2", "u4")
	require.NoError(t, err)

	require.Equal(t, []string{"u4", "u3"}, pr.AssignedReviewers)
}

func TestReplaceReviewer_NoChangeWhenSameID(t *testing.T) {
	pr, err := NewPullRequest("pr-1", "Add feature", "u1")
	require.NoError(t, err)

	err = pr.AssignReviewers([]string{"u2"})
	require.NoError(t, err)

	err = pr.ReplaceReviewer("u2", "u2")
	require.NoError(t, err)
	require.Equal(t, []string{"u2"}, pr.AssignedReviewers)
}
