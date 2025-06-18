package services_test

import (
	"fmt"
	"goapp/internal/pkg/services"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestIdService(t *testing.T) {
	idSvc := services.NewDefaultIDService(1)
	userId := idSvc.GenUserID()
	orderId := idSvc.GenOrderID()
	fmt.Println(userId, strings.ToUpper(strconv.FormatInt(userId, 36)))
	fmt.Println(orderId, strings.ToUpper(strconv.FormatInt(orderId, 36)))

	// return
	userIdMP := sync.Map{}
	wg := sync.WaitGroup{}
	wg.Add(100)
	for range 100 {
		go func() {
			defer wg.Done()
			for range 1000 {
				id := idSvc.GenOrderID()
				if _, loaded := userIdMP.LoadOrStore(id, struct{}{}); loaded {
					t.Errorf("duplicate id: %d", id)
				}
			}
		}()
	}
	wg.Wait()
}
