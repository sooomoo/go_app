package features_test

import (
	"context"
	"fmt"
	"goapp/internal/app/features/users"
	"goapp/pkg/ids"
	"testing"
)

func TestUserStoreDeleteByID(t *testing.T) {
	store := users.NewUserStore()
	id, _ := ids.NewUIDFromHex("0198bb5b-3509-7a19-94c2-a641d5490679")
	err := store.DeleteByID(context.Background(), id)
	fmt.Println(err)
}
