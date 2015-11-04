package main

import "testing"

func TestPackServiceLink(t *testing.T) {
	realValue := packServiceLink("192.168.0.1", "http", 80)
	expectedValue := "http://192.168.0.1:80"
	if realValue != expectedValue {
		t.Error("PackServiceLink failed expectedValue:" + expectedValue + " got " + realValue)
	}
}
