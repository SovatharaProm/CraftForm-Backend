package service

import (
	"context"

	"github.com/sovatharaprom/craftform-backend/internal/model"
	"github.com/sovatharaprom/craftform-backend/internal/repository"
)

type FormService struct {
	repo *repository.FormRepo
}

func NewFormService(repo *repository.FormRepo) *FormService {
	return &FormService{repo: repo}
}

func (s *FormService) List(ctx context.Context, ownerID string, filter model.FormFilter) ([]model.Form, error) {
	return s.repo.ListByOwner(ctx, ownerID, filter)
}

func (s *FormService) ListPublic(ctx context.Context, query string) ([]model.Form, error) {
	return s.repo.ListPublic(ctx, query)
}

func (s *FormService) Create(ctx context.Context, ownerID string, req model.FormRequest) (*model.Form, error) {
	if req.Status == "" {
		req.Status = model.FormStatusDraft
	}
	return s.repo.Create(ctx, ownerID, req)
}

func (s *FormService) GetByID(ctx context.Context, formID, requesterID string) (*model.Form, error) {
	form, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form.Status == model.FormStatusDraft && form.OwnerID != requesterID {
		return nil, model.ErrNotFound
	}
	return form, nil
}

func (s *FormService) Update(ctx context.Context, formID, ownerID string, req model.FormRequest) (*model.Form, error) {
	if err := s.assertOwner(ctx, formID, ownerID); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, formID, req)
}

func (s *FormService) Delete(ctx context.Context, formID, ownerID string) error {
	if err := s.assertOwner(ctx, formID, ownerID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, formID)
}

func (s *FormService) Duplicate(ctx context.Context, formID, ownerID string) (*model.Form, error) {
	if err := s.assertOwner(ctx, formID, ownerID); err != nil {
		return nil, err
	}
	return s.repo.Duplicate(ctx, formID, ownerID)
}

func (s *FormService) assertOwner(ctx context.Context, formID, ownerID string) error {
	form, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return err
	}
	if form.OwnerID != ownerID {
		return model.ErrForbidden
	}
	return nil
}
