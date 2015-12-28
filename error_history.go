package main

import (
	"fmt"
	"sync"
	"time"
)

//хранит определенное количество сообщений об ошибках
type errorHistoryStore struct {
	sync.RWMutex
	data     []string
	MaxCount int
}

//Добавляет сообщение в историю ошибок
func (e *errorHistoryStore) Add(s string) {
	tm := time.Now().Format("2006-01-02 15:04:05")
	e.Lock()
	defer e.Unlock()
	e.data = append(e.data, fmt.Sprintf("%s %s", tm, s))
	if len(e.data) > e.MaxCount {
		e.data = e.data[1:]
	}
}

//Возвращает историю ошибок
func (e *errorHistoryStore) Get() []string {
	e.RLock()
	e.RUnlock()
	return e.data
}

//newErrorHistoryStore создает новый объект errorHistoryStore и возвращает его 
func newErrorHistoryStore(MaxCount int) *errorHistoryStore {
	return &errorHistoryStore{MaxCount: MaxCount}
}
