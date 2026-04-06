package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

// ProgressDTO mirrors library.Progress on the wire.
type ProgressDTO struct {
	BookID    int64     `json:"book_id"`
	Position  string    `json:"position"`
	Percent   float64   `json:"percent"`
	Finished  bool      `json:"finished"`
	Theme     string    `json:"theme,omitempty"`
	FontSize  string    `json:"font_size,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetProgressInput is the path param for GET /api/books/{id}/progress.
type GetProgressInput struct {
	ID int64 `path:"id"`
}

// GetProgressOutput is the response.
type GetProgressOutput struct {
	Body ProgressDTO
}

// PutProgressInput carries the path id and request body.
type PutProgressInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Position string  `json:"position" doc:"Opaque position (CFI for EPUB, 'page:N' for PDF)" minLength:"1"`
		Percent  float64 `json:"percent" doc:"0..1 completion" minimum:"0" maximum:"1"`
		Finished bool    `json:"finished"`
		Theme    string  `json:"theme,omitempty"`
		FontSize string  `json:"font_size,omitempty"`
	}
}

// RecentProgressInput is the query param for GET /api/progress/recent.
type RecentProgressInput struct {
	Limit int `query:"limit" default:"12" minimum:"1" maximum:"50"`
}

// RecentProgressOutput is the list of recent-reading entries.
type RecentProgressOutput struct {
	Body struct {
		Entries []RecentProgressDTO `json:"entries"`
	}
}

// RecentProgressDTO pairs a book with its progress for the UI.
type RecentProgressDTO struct {
	Book     BookDTO     `json:"book"`
	Progress ProgressDTO `json:"progress"`
}

// ImportProgressInput is the bulk localStorage → server payload.
type ImportProgressInput struct {
	Body struct {
		Entries []struct {
			BookID   int64   `json:"book_id"`
			Position string  `json:"position" minLength:"1"`
			Percent  float64 `json:"percent" minimum:"0" maximum:"1"`
		} `json:"entries"`
	}
}

// ImportProgressOutput reports how many entries were imported.
type ImportProgressOutput struct {
	Body struct {
		Imported int `json:"imported"`
	}
}

func registerProgress(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "get-progress",
		Method:      http.MethodGet,
		Path:        "/api/books/{id}/progress",
		Summary:     "Get the signed-in user's reading progress for a book",
		Tags:        []string{"progress"},
	}, func(ctx context.Context, in *GetProgressInput) (*GetProgressOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		p, err := d.Store.GetProgress(ctx, u.ID, in.ID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("no progress yet")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("get progress", err)
		}
		return &GetProgressOutput{Body: toProgressDTO(p)}, nil
	})

	// Register PUT for normal saves.
	progressHandler := func(ctx context.Context, in *PutProgressInput) (*GetProgressOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		p := &library.Progress{
			UserID: u.ID, BookID: in.ID,
			Position: in.Body.Position, Percent: in.Body.Percent,
			Finished: in.Body.Finished, Theme: in.Body.Theme, FontSize: in.Body.FontSize,
		}
		if err := d.Store.UpsertProgress(ctx, p); err != nil {
			return nil, huma.Error500InternalServerError("save progress", err)
		}
		// Re-read so returned values include COALESCE'd theme/font_size.
		saved, err := d.Store.GetProgress(ctx, u.ID, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("reload progress", err)
		}
		return &GetProgressOutput{Body: toProgressDTO(saved)}, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "put-progress",
		Method:      http.MethodPut,
		Path:        "/api/books/{id}/progress",
		Summary:     "Save the signed-in user's reading progress",
		Tags:        []string{"progress"},
	}, progressHandler)

	// Also accept POST for the same handler — sendBeacon (used on page
	// unload) always sends POST and can't be changed to PUT.
	huma.Register(api, huma.Operation{
		OperationID: "post-progress",
		Method:      http.MethodPost,
		Path:        "/api/books/{id}/progress",
		Summary:     "Save reading progress (sendBeacon compat)",
		Tags:        []string{"progress"},
	}, progressHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-recent-progress",
		Method:      http.MethodGet,
		Path:        "/api/progress/recent",
		Summary:     "Most recently updated progress across all books",
		Tags:        []string{"progress"},
	}, func(ctx context.Context, in *RecentProgressInput) (*RecentProgressOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		entries, err := d.Store.RecentProgress(ctx, u.ID, in.Limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("list recent", err)
		}
		out := &RecentProgressOutput{}
		out.Body.Entries = make([]RecentProgressDTO, 0, len(entries))
		for _, e := range entries {
			out.Body.Entries = append(out.Body.Entries, RecentProgressDTO{
				Book: toBookDTO(e.Book), Progress: toProgressDTO(&e.Progress),
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "import-progress",
		Method:      http.MethodPost,
		Path:        "/api/progress/import",
		Summary:     "Bulk-import localStorage progress (migration helper)",
		Tags:        []string{"progress"},
	}, func(ctx context.Context, in *ImportProgressInput) (*ImportProgressOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		entries := make([]library.Progress, 0, len(in.Body.Entries))
		for _, e := range in.Body.Entries {
			entries = append(entries, library.Progress{
				BookID: e.BookID, Position: e.Position, Percent: e.Percent,
			})
		}
		n, err := d.Store.BulkImportProgress(ctx, u.ID, entries)
		if err != nil {
			return nil, huma.Error500InternalServerError("import", err)
		}
		out := &ImportProgressOutput{}
		out.Body.Imported = n
		return out, nil
	})
}

func toProgressDTO(p *library.Progress) ProgressDTO {
	return ProgressDTO{
		BookID: p.BookID, Position: p.Position, Percent: p.Percent,
		Finished: p.Finished, Theme: p.Theme, FontSize: p.FontSize,
		UpdatedAt: p.UpdatedAt,
	}
}
