package domain

import (
	"fmt"
	"time"
)

type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	ID               string
	Name             string
	AuthorID         string
	Status           PRStatus
	AssignedReviewers []string
	CreatedAt        time.Time
	MergedAt         *time.Time
}

func NewPullRequest(id, name, authorID string) (*PullRequest, error) {
	if id == "" || name == "" || authorID == "" {
		return nil, fmt.Errorf("empty parameter")
	}

	now := time.Now()

	return &PullRequest{
		ID:               id,
		Name:             name,
		AuthorID:         authorID,
		Status:           PRStatusOpen,
		AssignedReviewers: []string{},
		CreatedAt:        now,
		MergedAt:         nil,
	}, nil
}

func (p *PullRequest) AssignReviewers(reviewers []string) error {
	if len(reviewers) > 2 {
		return fmt.Errorf("too many reviewers: %d", len(reviewers))
	}

	seen := make(map[string]struct{}, len(reviewers))

	for _, id := range reviewers {
		if id == "" {
			return fmt.Errorf("empty reviewer id")
		}
		if id == p.AuthorID {
			return fmt.Errorf("author cannot be reviewer")
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("duplicate reviewer id: %s", id)
		}
		seen[id] = struct{}{}
	}

	p.AssignedReviewers = reviewers
	return nil
}

func (p *PullRequest) MarkMerged() {
	now := time.Now()
	if p.Status == PRStatusMerged {
		if p.MergedAt == nil {
			p.MergedAt = &now
		}
		return
	}

	p.Status = PRStatusMerged
	p.MergedAt = &now
}

func (p *PullRequest) CanModifyReviewers() bool {
	return p.Status == PRStatusOpen
}

func (p *PullRequest) ReplaceReviewer(oldID, newID string) error {
	if !p.CanModifyReviewers() {
		return NewDomainError(ErrorCodePRMerged, "cannot replace reviewers on merged PR")
	}

	if newID == "" {
		return fmt.Errorf("empty new reviewer id")
	}
	if newID == p.AuthorID {
		return fmt.Errorf("author cannot be reviewer")
	}
	if oldID == newID {
		return nil
	}

	for _, id := range p.AssignedReviewers {
		if id == newID && id != oldID {
			return fmt.Errorf("new reviewer already assigned")
		}
	}

	idx := -1
	for i, id := range p.AssignedReviewers {
		if id == oldID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return NewDomainError(ErrorCodeNotAssigned, "reviewer is not assigned to this PR")
	}

	p.AssignedReviewers[idx] = newID
	return nil
}
