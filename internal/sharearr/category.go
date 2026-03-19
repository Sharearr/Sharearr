package sharearr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrCategoryNotFound = errors.New("category not found")

type Category struct {
	ID        int64         `db:"id"`
	Name      string        `db:"name"`
	ParentID  sql.NullInt64 `db:"parent_id"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}

type CategoryRepository struct {
	db *sqlx.DB
}

func NewCategoryRepository(db *sqlx.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetRootByName(ctx context.Context, name string) (*Category, error) {
	c := &Category{}
	err := r.db.GetContext(ctx, c,
		`SELECT id, name, parent_id, created_at, updated_at
		 FROM categories
		 WHERE lower(name) = lower(?) AND parent_id IS NULL`,
		name,
	)
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

func NewCategoryServiceFromDB(db *sqlx.DB) *CategoryService {
	return NewCategoryService(NewCategoryRepository(db))
}

type CategoryTree struct {
	Category
	Subcategories []Category
}

func (r *CategoryRepository) ListAll(ctx context.Context) ([]Category, error) {
	var cats []Category
	err := r.db.SelectContext(ctx, &cats,
		`SELECT id, name, parent_id, created_at, updated_at FROM categories ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return cats, nil
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
	query, args, err := sqlx.In(
		`SELECT id, name, parent_id, created_at, updated_at FROM categories WHERE id IN (?)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("build list by ids query: %w", err)
	}
	var cats []Category
	if err := r.db.SelectContext(ctx, &cats, query, args...); err != nil {
		return nil, fmt.Errorf("list categories by ids: %w", err)
	}
	return cats, nil
}

func (s *CategoryService) ListByIDs(ctx context.Context, ids []int64) ([]Category, error) {
	return s.repo.ListByIDs(ctx, ids)
}
