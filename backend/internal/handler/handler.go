package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"module-backend/internal/ai"
	"module-backend/internal/excel"
	"module-backend/internal/schema"
)

type Handler struct {
	AI *ai.Client
}

func New(aiClient *ai.Client) *Handler {
	return &Handler{AI: aiClient}
}

type extractResponse struct {
	Results []extractResult `json:"results"`
}

type extractResult struct {
	Filename string         `json:"filename"`
	Module   *schema.Module `json:"module,omitempty"`
	Error    string         `json:"error,omitempty"`
}

const maxUploadBytes = 20 << 20 // 20MB per file

func (h *Handler) Extract(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		httpError(w, http.StatusBadRequest, "multipart parse failed: "+err.Error())
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		httpError(w, http.StatusBadRequest, "no 'images' field in form")
		return
	}

	out := extractResponse{Results: make([]extractResult, 0, len(files))}
	for _, fh := range files {
		res := extractResult{Filename: fh.Filename}
		if fh.Size > maxUploadBytes {
			res.Error = fmt.Sprintf("file too large (max %d bytes)", maxUploadBytes)
			out.Results = append(out.Results, res)
			continue
		}
		f, err := fh.Open()
		if err != nil {
			res.Error = "open: " + err.Error()
			out.Results = append(out.Results, res)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes))
		f.Close()
		if err != nil {
			res.Error = "read: " + err.Error()
			out.Results = append(out.Results, res)
			continue
		}

		mime := fh.Header.Get("Content-Type")
		if mime == "" {
			mime = "image/png"
		}

		mod, err := h.AI.ExtractModule(r.Context(), data, mime)
		if err != nil {
			log.Printf("extract %s: %v", fh.Filename, err)
			res.Error = err.Error()
			out.Results = append(out.Results, res)
			continue
		}
		res.Module = mod
		out.Results = append(out.Results, res)
	}

	writeJSON(w, http.StatusOK, out)
}

type exportRequest struct {
	Modules []schema.Module `json:"modules"`
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if len(req.Modules) == 0 {
		httpError(w, http.StatusBadRequest, "modules is empty")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="modules.xlsx"`)
	if err := excel.WriteModules(w, req.Modules); err != nil {
		log.Printf("excel write: %v", err)
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
