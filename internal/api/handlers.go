package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/extract"
	"github.com/annismckenzie/x-article-exporter/internal/jobs"
)

type exportRequest struct {
	URL       string `json:"url"`
	Translate string `json:"translate,omitempty"`
	DarkMode  bool   `json:"dark_mode,omitempty"`
}

type exportResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type statusResponse struct {
	ID        string            `json:"id"`
	Status    string            `json:"status"`
	Error     string            `json:"error,omitempty"`
	Title     string            `json:"title,omitempty"`
	Author    string            `json:"author,omitempty"`
	PageCount int               `json:"page_count,omitempty"`
	WordCount int               `json:"word_count,omitempty"`
	Links     map[string]string `json:"links,omitempty"`
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var req exportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	if _, err := extract.ExtractArticleID(req.URL); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid article URL: %v", err))
		return
	}

	id := s.manager.Submit(req.URL, req.Translate, req.DarkMode)
	writeJSON(w, http.StatusAccepted, exportResponse{ID: id, Status: string(jobs.StatusProcessing)})
}

func (s *Server) handleExportStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job := s.manager.Get(id)
	if job == nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	resp := statusResponse{
		ID:     job.ID,
		Status: string(job.Status),
	}

	switch job.Status {
	case jobs.StatusFailed:
		resp.Error = job.Error
	case jobs.StatusComplete:
		resp.Title = job.Result.Title
		resp.Author = job.Result.Author
		resp.PageCount = job.Result.PageCount
		resp.WordCount = job.Result.WordCount
		resp.Links = map[string]string{
			"pdf": fmt.Sprintf("/export/%s/pdf", job.ID),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleExportPDF(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job := s.manager.Get(id)
	if job == nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	switch job.Status {
	case jobs.StatusProcessing:
		writeJSON(w, http.StatusAccepted, statusResponse{ID: job.ID, Status: string(job.Status)})
	case jobs.StatusFailed:
		writeError(w, http.StatusInternalServerError, job.Error)
	case jobs.StatusComplete:
		filename := sanitizeFilename(job.Result.Title) + ".pdf"
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Write(job.Result.PDFBytes)
	}
}

var unsafeFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizeFilename(title string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		return "article"
	}
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "article"
	}
	return name
}
