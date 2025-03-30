package repositories

import (
	"goapp/internal/app/models"

	"github.com/sooomo/niu"
)

type RepositoryOfUser struct {
	cache *niu.Cache
	db    any
}

func NewRepositoryOfUser(cache *niu.Cache, db any) *RepositoryOfUser {
	return &RepositoryOfUser{
		cache: cache,
		db:    db,
	}
}

func (r *RepositoryOfUser) Upsert(phone string) (*models.ModelOfUser, error) {
	return nil, nil
}

func (r *RepositoryOfUser) GetById(userId int) (*models.ModelOfUser, error) {
	return nil, nil
}
