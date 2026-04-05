package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

// CollectionDTO is the wire representation of a Collection.
type CollectionDTO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	BookCount int       `json:"book_count"`
	CreatedAt time.Time `json:"created_at"`
}

type ListCollectionsOutput struct {
	Body struct {
		Collections []CollectionDTO `json:"collections"`
	}
}

type CreateCollectionInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100"`
	}
}

type CreateCollectionOutput struct {
	Body CollectionDTO
}

type GetCollectionInput struct {
	ID int64 `path:"id"`
}

type GetCollectionOutput struct {
	Body CollectionDTO
}

type RenameCollectionInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100"`
	}
}

type DeleteCollectionInput struct {
	ID int64 `path:"id"`
}

type ModifyCollectionBookInput struct {
	CollectionID int64 `path:"id"`
	BookID       int64 `path:"book_id"`
}

type EmptyOutput struct {
	Body struct{}
}

func registerCollections(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "list-collections",
		Method:      http.MethodGet,
		Path:        "/api/collections",
		Summary:     "List the signed-in user's collections",
		Tags:        []string{"collections"},
	}, func(ctx context.Context, _ *struct{}) (*ListCollectionsOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		cs, err := d.Store.ListCollections(ctx, u.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("list collections", err)
		}
		out := &ListCollectionsOutput{}
		out.Body.Collections = make([]CollectionDTO, 0, len(cs))
		for _, c := range cs {
			out.Body.Collections = append(out.Body.Collections, toCollectionDTO(c))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-collection",
		Method:        http.MethodPost,
		Path:          "/api/collections",
		Summary:       "Create a new collection",
		Tags:          []string{"collections"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateCollectionInput) (*CreateCollectionOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		c, err := d.Store.CreateCollection(ctx, u.ID, in.Body.Name)
		if err != nil {
			return nil, huma.Error409Conflict("could not create collection", err)
		}
		return &CreateCollectionOutput{Body: toCollectionDTO(c)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-collection",
		Method:      http.MethodGet,
		Path:        "/api/collections/{id}",
		Summary:     "Get a collection by id",
		Tags:        []string{"collections"},
	}, func(ctx context.Context, in *GetCollectionInput) (*GetCollectionOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		c, err := d.Store.GetCollection(ctx, u.ID, in.ID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("collection not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("get collection", err)
		}
		return &GetCollectionOutput{Body: toCollectionDTO(c)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "rename-collection",
		Method:      http.MethodPatch,
		Path:        "/api/collections/{id}",
		Summary:     "Rename a collection",
		Tags:        []string{"collections"},
	}, func(ctx context.Context, in *RenameCollectionInput) (*GetCollectionOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		err := d.Store.RenameCollection(ctx, u.ID, in.ID, in.Body.Name)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("collection not found")
		}
		if err != nil {
			return nil, huma.Error409Conflict("could not rename", err)
		}
		c, err := d.Store.GetCollection(ctx, u.ID, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("reload", err)
		}
		return &GetCollectionOutput{Body: toCollectionDTO(c)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-collection",
		Method:        http.MethodDelete,
		Path:          "/api/collections/{id}",
		Summary:       "Delete a collection",
		Tags:          []string{"collections"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *DeleteCollectionInput) (*EmptyOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		err := d.Store.DeleteCollection(ctx, u.ID, in.ID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("collection not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("delete", err)
		}
		return &EmptyOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "add-to-collection",
		Method:        http.MethodPost,
		Path:          "/api/collections/{id}/books/{book_id}",
		Summary:       "Add a book to a collection",
		Tags:          []string{"collections"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *ModifyCollectionBookInput) (*EmptyOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		err := d.Store.AddBookToCollection(ctx, u.ID, in.CollectionID, in.BookID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("collection not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("add book", err)
		}
		return &EmptyOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "remove-from-collection",
		Method:        http.MethodDelete,
		Path:          "/api/collections/{id}/books/{book_id}",
		Summary:       "Remove a book from a collection",
		Tags:          []string{"collections"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *ModifyCollectionBookInput) (*EmptyOutput, error) {
		u := UserFromContext(ctx)
		if u == nil {
			return nil, huma.Error401Unauthorized("login required")
		}
		err := d.Store.RemoveBookFromCollection(ctx, u.ID, in.CollectionID, in.BookID)
		if errors.Is(err, library.ErrNotFound) {
			return nil, huma.Error404NotFound("collection not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("remove book", err)
		}
		return &EmptyOutput{}, nil
	})
}

func toCollectionDTO(c *library.Collection) CollectionDTO {
	return CollectionDTO{ID: c.ID, Name: c.Name, BookCount: c.BookCount, CreatedAt: c.CreatedAt}
}
