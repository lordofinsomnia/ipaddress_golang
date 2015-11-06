package main

import "testing"

func TestPackServiceLink(t *testing.T) {
	realValue := packServiceLink("192.168.0.1", "http", 80)
	expectedValue := "http://192.168.0.1:80"
	if realValue != expectedValue {
		t.Error("PackServiceLink failed expectedValue:" + expectedValue + " got " + realValue)
	}
}

//BenchmarkPackServiceLink-8	2000000000	         0.00 ns/op
func BenchmarkPackServiceLink(b *testing.B) {
	packServiceLink("192.168.0.1", "http", 80)

}
