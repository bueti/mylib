package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

// BookDTO is the wire representation of a book.
type BookDTO struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	SortTitle   string      `json:"sort_title"`
	Subtitle    string      `json:"subtitle,omitempty"`
	Description string      `json:"description,omitempty"`
	Authors     []AuthorDTO `json:"authors"`
	Series      *SeriesDTO  `json:"series,omitempty"`
	SeriesIndex *float64    `json:"series_index,omitempty"`
	Language    string      `json:"language,omitempty"`
	ISBN        string      `json:"isbn,omitempty"`
	Publisher   string      `json:"publisher,omitempty"`
	PublishedAt string      `json:"published_at,omitempty"`
	Format      string      `json:"format"`
	SizeBytes   int64       `json:"size_bytes"`
	Tags        []string    `json:"tags,omitempty"`
	HasCover    bool        `json:"has_cover"`
	AddedAt     time.Time   `json:"added_at"`
}

// AuthorDTO is the wire representation of an author.
type AuthorDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	SortName string `json:"sort_name"`
}

// SeriesDTO is the wire representation of a series.
type SeriesDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ListBooksInput is the query for GET /books.
type ListBooksInput struct {
	Q        string `query:"q" doc:"Full-text search query"`
	AuthorID int64  `query:"author_id" doc:"Filter by author id"`
	SeriesID int64  `query:"series_id" doc:"Filter by series id"`
	Tag      string `query:"tag" doc:"Filter by tag name"`
	Format   string `query:"format" doc:"Filter by format (epub, pdf, mobi, azw3)"`
	Sort     string `query:"sort" doc:"Sort key: title, -title, added, -added" enum:"title,-title,added,-added"`
	Limit    int    `query:"limit" doc:"Page size (max 500)" default:"50" minimum:"1" maximum:"500"`
	Offset   int    `query:"offset" doc:"Page offset" minimum:"0"`
}

// ListBooksOutput is the response for GET /books.
type ListBooksOutput struct {
	Body struct {
		Books  []BookDTO `json:"books"`
		Total  int       `json:"total"`
		Limit  int       `json:"limit"`
		Offset int       `json:"offset"`
	}
}

// GetBookInput is the path params for GET /books/{id}.
type GetBookInput struct {
	ID int64 `path:"id" doc:"Book id"`
}

// GetBookOutput is the response for GET /books/{id}.
type GetBookOutput struct {
	Body BookDTO
}

func registerBooks(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "list-books",
		Method:      http.MethodGet,
		Path:        "/api/books",
		Summary:     "List books",
		Tags:        []string{"books"},
	}, func(ctx context.Context, in *ListBooksInput) (*ListBooksOutput, error) {
		filter := library.BookFilter{
			Query:  in.Q,
			Tag:    in.Tag,
			Format: in.Format,
			Sort:   in.Sort,
			Limit:  in.Limit,
			Offset: in.Offset,
		}
		if in.AuthorID > 0 {
			filter.AuthorID = &in.AuthorID
		}
		if in.SeriesID > 0 {
			filter.SeriesID = &in.SeriesID
		}
		books, total, err := d.Store.ListBooks(ctx, filter)
		if err != nil {
			return nil, huma.Error500InternalServerError("list books", err)
		}
		out := &ListBooksOutput{}
		out.Body.Books = make([]BookDTO, 0, len(books))
		for _, b := range books {
			out.Body.Books = append(out.Body.Books, toBookDTO(b))
		}
		out.Body.Total = total
		out.Body.Limit = filter.Limit
		if out.Body.Limit == 0 {
			out.Body.Limit = 50
		}
		out.Body.Offset = filter.Offset
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-book",
		Method:      http.MethodGet,
		Path:        "/api/books/{id}",
		Summary:     "Get book by id",
		Tags:        []string{"books"},
	}, func(ctx context.Context, in *GetBookInput) (*GetBookOutput, error) {
		b, err := d.Store.GetBook(ctx, in.ID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("book not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("get book", err)
		}
		return &GetBookOutput{Body: toBookDTO(b)}, nil
	})
}

// toBookDTO projects a library.Book onto the wire type. Slice fields
// are always non-nil so JSON clients get [] instead of null.
func toBookDTO(b *library.Book) BookDTO {
	out := BookDTO{
		ID:          b.ID,
		Title:       b.Title,
		SortTitle:   b.SortTitle,
		Subtitle:    b.Subtitle,
		Description: b.Description,
		SeriesIndex: b.SeriesIndex,
		Language:    b.Language,
		ISBN:        b.ISBN,
		Publisher:   b.Publisher,
		PublishedAt: b.PublishedAt,
		Format:      b.Format,
		SizeBytes:   b.SizeBytes,
		Tags:        []string{},
		Authors:     []AuthorDTO{},
		HasCover:    b.CoverPath != "",
		AddedAt:     b.AddedAt,
	}
	out.Tags = append(out.Tags, b.Tags...)
	for _, a := range b.Authors {
		out.Authors = append(out.Authors, AuthorDTO{ID: a.ID, Name: a.Name, SortName: a.SortName})
	}
	if b.SeriesID != nil {
		out.Series = &SeriesDTO{ID: *b.SeriesID, Name: b.SeriesName}
	}
	return out
}
