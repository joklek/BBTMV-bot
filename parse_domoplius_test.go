package main

import (
	"testing"
)

type DomopliusData struct {
	Provided string
	Expected string
}

var DomopliusTestData = []DomopliusData{
	{
		Provided: "zzKzM3MCA2NjYgNjY2NjY=",
		Expected: "+37066666666",
	},
	{
		Provided: "asODYyMjIyMjIy",
		Expected: "+37062222222",
	},
}

func TestDomopliusDecodeNumber(t *testing.T) {
	for _, v := range DomopliusTestData {
		if res := domopliusDecodeNumber(v.Provided); res != v.Expected {
			t.Errorf("Result is incorrect, got: '%s', want: '%s'.", res, v.Expected)
		}
	}
}
