package notify

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/renyongfengs/Gobase/log"
)

var notifyCache map[string]chan interface{}

var mu sync.Mutex

var ErrorCommandTimeOut = errors.New("command time out")

func init() {
	notifyCache = make(map[string]chan interface{})
}

func NewNotify(notifyName string) chan interface{} {
	mu.Lock()
	defer mu.Unlock()
	result := make(chan interface{})
	notifyCache[notifyName] = result
	return result
}

func CloseNotify(notifyName string) {
	mu.Lock()
	defer mu.Unlock()
	if resultChan, ok := notifyCache[notifyName]; ok {
		close(resultChan)
	}
	delete(notifyCache, notifyName)
}

func InsertNotifyData(notifyName string, data interface{}) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	mu.Lock()
	defer mu.Unlock()
	if resultChan, ok := notifyCache[notifyName]; ok {
		select {
		case resultChan <- data:
		case <-ticker.C:
			log.Warn("Insert notify to channel:%s with data:%+v timeout", notifyName, data)
		}
	}
}

func InsertNotifyDataWithLikedName(likedName string, data interface{}) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	mu.Lock()
	defer mu.Unlock()
	for notifyName, d := range notifyCache {
		if strings.Contains(notifyName, likedName) {
			select {
			case d <- data:
			case <-ticker.C:
				log.Warn("Insert notify data %s timeout", notifyName)
			}
		}
	}
}

func WaitNotifyResponse(notifyName string, timeout time.Duration) (interface{}, error) {
	resultChan := NewNotify(notifyName)
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	defer CloseNotify(notifyName)
	select {
	case d := <-resultChan:
		switch d.(type) {
		case error:
			return nil, d.(error)
		default:
			return d, nil
		}
	case <-ticker.C:
		return nil, ErrorCommandTimeOut
	}
}
