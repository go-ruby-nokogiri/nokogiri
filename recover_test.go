// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"errors"
	"strings"
	"testing"
)

func TestRecoverEvalError(t *testing.T) {
	// nil -> nil (no panic)
	if recoverEvalError(nil) != nil {
		t.Fatal("nil should return nil")
	}
	// an evalError is classified as the returned error
	if got := recoverEvalError(evalError("boom")); got == nil || got.Error() != "boom" {
		t.Fatalf("evalError: %v", got)
	}
	// a non-evalError value is re-raised
	defer func() {
		r := recover()
		if r != "surprise" {
			t.Fatalf("re-raise: %v", r)
		}
	}()
	recoverEvalError("surprise")
	t.Fatal("expected re-panic")
}

func TestRecoverParseError(t *testing.T) {
	if recoverParseError(nil) != nil {
		t.Fatal("nil should return nil")
	}
	if got := recoverParseError(parseError("bad")); got == nil || got.Error() != "bad" {
		t.Fatalf("parseError: %v", got)
	}
	defer func() {
		if r := recover(); r != 42 {
			t.Fatalf("re-raise: %v", r)
		}
	}()
	recoverParseError(42)
	t.Fatal("expected re-panic")
}

// failingReader always errors, to exercise the parser error-return paths.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failure") }

func TestHTMLReaderError(t *testing.T) {
	if _, err := HTMLReader(failingReader{}); err == nil {
		t.Error("HTMLReader should propagate a read error")
	}
	if _, err := HTMLFragmentReader(failingReader{}); err == nil {
		t.Error("HTMLFragmentReader should propagate a read error")
	}
}

func TestHTMLReaderOK(t *testing.T) {
	// the reader variants parse normal input too
	d, err := HTMLReader(strings.NewReader(`<p>hi</p>`))
	if err != nil {
		t.Fatal(err)
	}
	if d.AtCSSMust(t, "p").Text() != "hi" {
		t.Fatal("HTMLReader content")
	}
}
