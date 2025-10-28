package logging

import (
	"context"
	"goapp/pkg/db"
)

type Store interface {
	Write(ctx context.Context, log *ServiceLog) error
	WriteMany(ctx context.Context, logs []*ServiceLog) error
}

type DBStore struct{}

func NewDBStore() *DBStore {
	return &DBStore{}
}

func (s *DBStore) Write(ctx context.Context, log *ServiceLog) error {
	_, err := db.Get().NewInsert().Model(log).Exec(ctx)
	return err
}

func (s *DBStore) WriteMany(ctx context.Context, logs []*ServiceLog) error {
	_, err := db.Get().NewInsert().Model(logs).Exec(ctx)
	return err
}
