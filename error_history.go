package main

import (
	"sync"
	"time"
)

//errorHistoryItem запись с текстом ошибки и временем её возникновения
type errorHistoryItem struct {
	Time time.Time
	Msg  string
}

//хранит определенное количество сообщений об ошибках
type errorHistoryStore struct {
	sync.RWMutex
	data     []errorHistoryItem //[]string
	MaxCount int
}

//Добавляет сообщение в историю ошибок
func (e *errorHistoryStore) Add(s string) {
	e.Lock()
	defer e.Unlock()
	e.data = append(e.data, errorHistoryItem{time.Now(), s})
	if len(e.data) > e.MaxCount {
		e.data = e.data[1:]
	}
}

//Возвращает историю ошибок
func (e *errorHistoryStore) Get() []errorHistoryItem {
	e.RLock()
	e.RUnlock()
	return e.data
}

//newErrorHistoryStore создает новый объект errorHistoryStore и возвращает его
func newErrorHistoryStore(MaxCount int) *errorHistoryStore {
	return &errorHistoryStore{MaxCount: MaxCount}
}
