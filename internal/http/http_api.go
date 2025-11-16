package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Shyyw1e/avito-trainee-fall/internal/domain"
	"github.com/Shyyw1e/avito-trainee-fall/internal/platform/log"
	"github.com/Shyyw1e/avito-trainee-fall/internal/repository"
	"github.com/Shyyw1e/avito-trainee-fall/internal/usecase"
)

type Server struct {
	mux       *http.ServeMux
	teams     *usecase.TeamService
	users     *usecase.UserService
	prs       *usecase.PRService
	stats     *usecase.StatsService
	db        repository.DBExecutor 
	logger    log.Logger
	baseCtxFn func() context.Context
}

func NewServer(
	teams *usecase.TeamService,
	users *usecase.UserService,
	prs *usecase.PRService,
	stats *usecase.StatsService,
	db repository.DBExecutor,
	logger log.Logger,
) *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		teams:     teams,
		users:     users,
		prs:       prs,
		stats:     stats,
		db:        db,
		logger:    logger,
		baseCtxFn: context.Background,
	}

	s.registerRoutes()
	return s
}

// Handler возвращает http.Handler для http.Server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ===== DTO (transport-слой) =====

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type teamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type teamDTO struct {
	TeamName string          `json:"team_name"`
	Members  []teamMemberDTO `json:"members"`
}

type teamAddResponse struct {
	Team teamDTO `json:"team"`
}

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type userDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type setIsActiveResponse struct {
	User userDTO `json:"user"`
}

type createPRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type pullRequestDTO struct {
	ID                string     `json:"pull_request_id"`
	Name              string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type pullRequestResponse struct {
	PR pullRequestDTO `json:"pr"`
}

type mergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type reassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type reassignResponse struct {
	PR         pullRequestDTO `json:"pr"`
	ReplacedBy string         `json:"replaced_by"`
}

type pullRequestShortDTO struct {
	ID       string `json:"pull_request_id"`
	Name     string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
	Status   string `json:"status"`
}

type userReviewsResponse struct {
	UserID       string                `json:"user_id"`
	PullRequests []pullRequestShortDTO `json:"pull_requests"`
}

type assignmentsStatItem struct {
	UserID string `json:"user_id"`
	Count  int    `json:"count"`
}

type assignmentsStatResponse struct {
	Assignments []assignmentsStatItem `json:"assignments"`
}


func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /team/add", s.handleTeamAdd)
	s.mux.HandleFunc("GET /team/get", s.handleTeamGet)

	s.mux.HandleFunc("POST /users/setIsActive", s.handleSetIsActive)
	s.mux.HandleFunc("GET /users/getReview", s.handleGetUserReview)

	s.mux.HandleFunc("POST /pullRequest/create", s.handleCreatePR)
	s.mux.HandleFunc("POST /pullRequest/merge", s.handleMergePR)
	s.mux.HandleFunc("POST /pullRequest/reassign", s.handleReassign)

	s.mux.HandleFunc("GET /stats/assignments", s.handleStatsAssignments)

	s.mux.HandleFunc("GET /health", s.handleHealth)
}


func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if v == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("write_json_failed", "err", err)
	}
}

func (s *Server) writeDomainError(w http.ResponseWriter, err error) {
	var derr *domain.DomainError
	if errors.As(err, &derr) {
		body := errorResponse{}
		body.Error.Code = string(derr.Code)
		body.Error.Message = derr.Error()

		status := http.StatusBadRequest

		switch derr.Code {
		case domain.ErrorCodeTeamExists:
			status = http.StatusBadRequest // 400
		case domain.ErrorCodePRExists:
			status = http.StatusConflict // 409
		case domain.ErrorCodePRMerged:
			status = http.StatusConflict // 409
		case domain.ErrorCodeNotAssigned:
			status = http.StatusConflict // 409
		case domain.ErrorCodeNoCandidate:
			status = http.StatusConflict // 409
		case domain.ErrorCodeNotFound:
			status = http.StatusNotFound // 404
		default:
			status = http.StatusBadRequest
		}

		s.writeJSON(w, status, body)
		return
	}

	// Неизвестная ошибка — 500
	body := errorResponse{}
	body.Error.Code = "INTERNAL"
	body.Error.Message = "internal server error"

	s.logger.Error("internal_error", "err", err)
	s.writeJSON(w, http.StatusInternalServerError, body)
}

// Простая утилита для чтения JSON
func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dest); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

// ===== маппинг domain → DTO =====

func teamToDTO(t *domain.Team) teamDTO {
	members := make([]teamMemberDTO, 0, len(t.Members))
	for _, m := range t.Members {
		members = append(members, teamMemberDTO{
			UserID:   m.ID,
			Username: m.Name,
			IsActive: m.IsActive,
		})
	}
	return teamDTO{
		TeamName: t.Name,
		Members:  members,
	}
}

func userToDTO(u *domain.User) userDTO {
	return userDTO{
		UserID:   u.ID,
		Username: u.Name,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
}

func prToDTO(p *domain.PullRequest) pullRequestDTO {
	var created *time.Time
	if !p.CreatedAt.IsZero() {
		t := p.CreatedAt
		created = &t
	}

	var merged *time.Time
	if !p.MergedAt.IsZero() {
		t := p.MergedAt
		merged = t
	}

	return pullRequestDTO{
		ID:                p.ID,
		Name:              p.Name,
		AuthorID:          p.AuthorID,
		Status:            string(p.Status),
		AssignedReviewers: append([]string(nil), p.AssignedReviewers...),
		CreatedAt:         created,
		MergedAt:          merged,
	}
}

func prShortToDTO(p domain.PullRequest) pullRequestShortDTO {
	return pullRequestShortDTO{
		ID:       p.ID,
		Name:     p.Name,
		AuthorID: p.AuthorID,
		Status:   string(p.Status),
	}
}


// POST /team/add
func (s *Server) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req teamDTO
	if !s.decodeJSON(w, r, &req) {
		return
	}

	members := make([]domain.User, 0, len(req.Members))
	for _, m := range req.Members {
		u, err := domain.NewUser(m.UserID, m.Username, req.TeamName, m.IsActive)
		if err != nil {
			http.Error(w, "bad user in request: "+err.Error(), http.StatusBadRequest)
			return
		}
		members = append(members, *u)
	}

	ctx := r.Context()
	team, err := s.teams.AddTeam(ctx, req.TeamName, members)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	resp := teamAddResponse{Team: teamToDTO(team)}
	s.writeJSON(w, http.StatusCreated, resp)
}

// GET /team/get?team_name=...
func (s *Server) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "team_name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	team, err := s.teams.GetTeam(ctx, teamName, s.db)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	resp := teamToDTO(team)
	s.writeJSON(w, http.StatusOK, resp)
}

// POST /users/setIsActive
func (s *Server) handleSetIsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req setIsActiveRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user, err := s.users.SetUserIsActive(ctx, s.db, req.UserID, req.IsActive)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	resp := setIsActiveResponse{User: userToDTO(user)}
	s.writeJSON(w, http.StatusOK, resp)
}

// GET /users/getReview?user_id=...
func (s *Server) handleGetUserReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	uid, prs, err := s.users.GetUserReviews(ctx, s.db, userID)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	out := make([]pullRequestShortDTO, 0, len(prs))
	for _, p := range prs {
		out = append(out, prShortToDTO(p))
	}

	resp := userReviewsResponse{
		UserID:       uid,
		PullRequests: out,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// POST /pullRequest/create
func (s *Server) handleCreatePR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req createPRRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	if req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		http.Error(w, "pull_request_id, pull_request_name and author_id are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	pr, err := s.prs.CreatePRWithAutoAssign(ctx, req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	resp := pullRequestResponse{PR: prToDTO(pr)}
	s.writeJSON(w, http.StatusCreated, resp)
}

// POST /pullRequest/merge
func (s *Server) handleMergePR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req mergePRRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	if req.PullRequestID == "" {
		http.Error(w, "pull_request_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	pr, err := s.prs.MergePR(ctx, req.PullRequestID)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	resp := pullRequestResponse{PR: prToDTO(pr)}
	s.writeJSON(w, http.StatusOK, resp)
}

// POST /pullRequest/reassign
func (s *Server) handleReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req reassignRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	if req.PullRequestID == "" || req.OldUserID == "" {
		http.Error(w, "pull_request_id and old_user_id are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	pr, newID, err := s.prs.ReassignReviewer(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	resp := reassignResponse{
		PR:         prToDTO(pr),
		ReplacedBy: newID,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// GET /stats/assignments  (доп. задание)
func (s *Server) handleStatsAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	stats, err := s.stats.GetAssignmentsByUser(ctx, s.db)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}

	out := make([]assignmentsStatItem, 0, len(stats))
	for uid, cnt := range stats {
		out = append(out, assignmentsStatItem{
			UserID: uid,
			Count:  cnt,
		})
	}

	resp := assignmentsStatResponse{Assignments: out}
	s.writeJSON(w, http.StatusOK, resp)
}

// GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
