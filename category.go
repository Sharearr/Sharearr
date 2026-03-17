package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrCategoryNotFound = errors.New("category not found")

type Category struct {
	ID        int64
	Name      string
	ParentID  sql.NullInt64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetRootByName(ctx context.Context, name string) (*Category, error) {
	c := &Category{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, parent_id, created_at, updated_at
		 FROM categories
		 WHERE lower(name) = lower(?) AND parent_id IS NULL`,
		name,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get root category by name: %w", err)
	}
	return c, nil
}

type CategoryService struct {
	repo *CategoryRepository
}

func NewCategoryService(repo *CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func NewCategoryServiceFromDB(db *sql.DB) *CategoryService {
	return NewCategoryService(NewCategoryRepository(db))
}

type CategoryTree struct {
	Category
	Subcategories []Category
}

func (r *CategoryRepository) ListAll(ctx context.Context) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, parent_id, created_at, updated_at FROM categories ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *CategoryService) ListTree(ctx context.Context) ([]CategoryTree, error) {
	all, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	nodes := make(map[int64]*CategoryTree, len(all))
	for i := range all {
		nodes[all[i].ID] = &CategoryTree{Category: all[i]}
	}
	for _, c := range all {
		if c.ParentID.Valid {
			if parent, ok := nodes[c.ParentID.Int64]; ok {
				parent.Subcategories = append(parent.Subcategories, c)
			}
		}
	}
	var roots []CategoryTree
	for _, c := range all {
		if !c.ParentID.Valid {
			roots = append(roots, *nodes[c.ID])
		}
	}
	return roots, nil
}

func (s *CategoryService) GetRootByName(ctx context.Context, name string) (*Category, error) {
	c, err := s.repo.GetRootByName(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	return c, err
}

func (r *CategoryRepository) ListByIDs(ctx context.Context, ids []int64) ([]Category, error) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, parent_id, created_at, updated_at
		 FROM categories WHERE id IN (`+placeholders(len(ids))+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list categories by ids: %w", err)
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *CategoryService) ListByIDs(ctx context.Context, ids []int64) ([]Category, error) {
	return s.repo.ListByIDs(ctx, ids)
}
