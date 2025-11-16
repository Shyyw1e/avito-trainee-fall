package usecase

import (
	"testing"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
)

type fakeRand struct {
	seq []int
	pos int
}

func (f *fakeRand) Intn(n int) int {
	if len(f.seq) == 0 {
		return 0
	}
	v := f.seq[f.pos%len(f.seq)]
	f.pos++
	if n <= 0 {
		return 0
	}
	if v < 0 {
		v = -v
	}
	return v % n
}

func TestChooseReviewers_NoCandidates(t *testing.T) {
	res := chooseReviewers(nil, nil)
	if len(res) != 0 {
		t.Fatalf("expected 0 reviewers, got %d", len(res))
	}
}

func TestChooseReviewers_OneCandidate(t *testing.T) {
	candidates := []domain.User{
		{ID: "u1"},
	}

	res := chooseReviewers(candidates, nil)
	if len(res) != 1 || res[0] != "u1" {
		t.Fatalf("expected [u1], got %#v", res)
	}
}

func TestChooseReviewers_TwoOrMore_NoRand(t *testing.T) {
	candidates := []domain.User{
		{ID: "u1"},
		{ID: "u2"},
		{ID: "u3"},
	}

	res := chooseReviewers(candidates, nil)
	if len(res) != 2 {
		t.Fatalf("expected 2 reviewers, got %d", len(res))
	}
	if res[0] != "u1" || res[1] != "u2" {
		t.Fatalf("expected [u1 u2], got %#v", res)
	}
}

func TestChooseReviewers_WithRand_Distinct(t *testing.T) {
	candidates := []domain.User{
		{ID: "u1"},
		{ID: "u2"},
		{ID: "u3"},
		{ID: "u4"},
	}

	r := &fakeRand{seq: []int{0, 1}}

	res := chooseReviewers(candidates, r)
	if len(res) != 2 {
		t.Fatalf("expected 2 reviewers, got %d", len(res))
	}
	if res[0] == res[1] {
		t.Fatalf("expected distinct reviewers, got %v and %v", res[0], res[1])
	}
}

func TestChooseOne_NoRand(t *testing.T) {
	candidates := []domain.User{
		{ID: "u10"},
		{ID: "u20"},
	}

	res := chooseOne(candidates, nil)
	if res != "u10" {
		t.Fatalf("expected u10, got %s", res)
	}
}

func TestChooseOne_WithRand(t *testing.T) {
	candidates := []domain.User{
		{ID: "u1"},
		{ID: "u2"},
		{ID: "u3"},
	}

	r := &fakeRand{seq: []int{2}}
	res := chooseOne(candidates, r)
	if res != "u3" {
		t.Fatalf("expected u3, got %s", res)
	}
}
