package main

import (
	"testing"
)

func TestProcessingCache(t *testing.T) {
	p := newProcessingCache()
	if p == nil {
		t.Error("newProcessingCache == nil")
		return
	}

	file := "c:\\1.txt"
	p.add(file)
	if p.check(file) == false {
		t.Error("check == false")
		return
	}

	p.del(file)
	if p.check(file) == true {
		t.Error("check == true")
		return
	}
}
