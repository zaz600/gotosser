package main

import (
	"fmt"
	"testing"
)

func TestErrorhistoryStore(t *testing.T) {
	hs := newErrorHistoryStore(10)
	if hs == nil {
		t.Error("newErrorHistoryStore == nil")
		return
	}
	hs.Add("test")
	if len(hs.Get()) != 1 {
		t.Error(`len(hs.Get()) != 1`)
		return
	}
	if hs.Get()[0].Msg != "test" {
		t.Error(`hs.Get()[0].Msg != "test"`)
		return
	}
	for i := 0; i < 15; i++ {
		hs.Add(fmt.Sprintf("test%d", i))
	}
	hs2 := hs.Get()
	if hs2[len(hs2)-1].Msg != "test14" {
		t.Error(`hs2[len(hs2)-1].Msg != "test14"`)
		return
	}

}
