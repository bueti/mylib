package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// ListAuthorsOutput is the response for GET /authors.
type ListAuthorsOutput struct {
	Body struct {
		Authors []AuthorDTO `json:"authors"`
	}
}

// ListSeriesOutput is the response for GET /series.
type ListSeriesOutput struct {
	Body struct {
		Series []SeriesDTO `json:"series"`
	}
}

// ListTagsOutput is the response for GET /tags.
type ListTagsOutput struct {
	Body struct {
		Tags []string `json:"tags"`
	}
}

func registerTaxonomy(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "list-authors",
		Method:      http.MethodGet,
		Path:        "/authors",
		Summary:     "List all authors",
		Tags:        []string{"taxonomy"},
	}, func(ctx context.Context, _ *struct{}) (*ListAuthorsOutput, error) {
		as, err := d.Store.ListAuthors(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("list authors", err)
		}
		out := &ListAuthorsOutput{}
		out.Body.Authors = make([]AuthorDTO, 0, len(as))
		for _, a := range as {
			out.Body.Authors = append(out.Body.Authors, AuthorDTO{ID: a.ID, Name: a.Name, SortName: a.SortName})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-series",
		Method:      http.MethodGet,
		Path:        "/series",
		Summary:     "List all series",
		Tags:        []string{"taxonomy"},
	}, func(ctx context.Context, _ *struct{}) (*ListSeriesOutput, error) {
		ss, err := d.Store.ListSeries(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("list series", err)
		}
		out := &ListSeriesOutput{}
		out.Body.Series = make([]SeriesDTO, 0, len(ss))
		for _, s := range ss {
			out.Body.Series = append(out.Body.Series, SeriesDTO{ID: s.ID, Name: s.Name})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-tags",
		Method:      http.MethodGet,
		Path:        "/tags",
		Summary:     "List all tags",
		Tags:        []string{"taxonomy"},
	}, func(ctx context.Context, _ *struct{}) (*ListTagsOutput, error) {
		ts, err := d.Store.ListTags(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("list tags", err)
		}
		out := &ListTagsOutput{}
		out.Body.Tags = ts
		if out.Body.Tags == nil {
			out.Body.Tags = []string{}
		}
		return out, nil
	})
}
