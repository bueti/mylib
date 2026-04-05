package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

// ScanJobDTO is the wire representation of a scan job.
type ScanJobDTO struct {
	ID           int64      `json:"id"`
	Root         string     `json:"root"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	FilesSeen    int        `json:"files_seen"`
	FilesAdded   int        `json:"files_added"`
	FilesUpdated int        `json:"files_updated"`
	FilesRemoved int        `json:"files_removed"`
	Error        string     `json:"error,omitempty"`
}

// TriggerScanOutput is returned by POST /scan.
type TriggerScanOutput struct {
	Body ScanJobDTO
}

// GetScanInput has the scan job id.
type GetScanInput struct {
	ID int64 `path:"id"`
}

// GetScanOutput returns a scan job.
type GetScanOutput struct {
	Body ScanJobDTO
}

func registerScan(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID:   "trigger-scan",
		Method:        http.MethodPost,
		Path:          "/scan",
		Summary:       "Start (or join) a library scan",
		Tags:          []string{"scan"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, _ *struct{}) (*TriggerScanOutput, error) {
		id, err := d.Scanner.ScanAll(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("start scan", err)
		}
		job, err := d.Store.GetScanJob(ctx, id)
		if err != nil {
			return nil, huma.Error500InternalServerError("load scan job", err)
		}
		return &TriggerScanOutput{Body: toScanJobDTO(job)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-scan",
		Method:      http.MethodGet,
		Path:        "/scan/{id}",
		Summary:     "Get a scan job",
		Tags:        []string{"scan"},
	}, func(ctx context.Context, in *GetScanInput) (*GetScanOutput, error) {
		job, err := d.Store.GetScanJob(ctx, in.ID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("scan job not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("get scan job", err)
		}
		return &GetScanOutput{Body: toScanJobDTO(job)}, nil
	})
}

func toScanJobDTO(j *library.ScanJob) ScanJobDTO {
	return ScanJobDTO{
		ID:           j.ID,
		Root:         j.Root,
		Status:       j.Status,
		StartedAt:    j.StartedAt,
		FinishedAt:   j.FinishedAt,
		FilesSeen:    j.FilesSeen,
		FilesAdded:   j.FilesAdded,
		FilesUpdated: j.FilesUpdated,
		FilesRemoved: j.FilesRemoved,
		Error:        j.Error,
	}
}
