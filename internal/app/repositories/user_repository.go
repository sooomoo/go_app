package repositories

import "github.com/sooomo/niu"

type RepositoryOfUser struct {
	cache *niu.Cache
	db    any
}
