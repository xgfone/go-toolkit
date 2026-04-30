package app

import (
	"testing"
)

func TestSet_Panics_EmptyKey(t *testing.T) {
	defer func() { _ = recover() }()
	New().Set("", "v")
	t.Error("expected panic")
}

func TestSet_Panics_NilValue(t *testing.T) {
	defer func() { _ = recover() }()
	New().Set("k", nil)
	t.Error("expected panic")
}

func TestSetFunc_Panics_EmptyKey(t *testing.T) {
	defer func() { _ = recover() }()
	New().SetFunc("", func() any { return nil })
	t.Error("expected panic")
}

func TestSetFunc_Panics_NilFunc(t *testing.T) {
	defer func() { _ = recover() }()
	New().SetFunc("k", nil)
	t.Error("expected panic")
}

func TestGet_Panics_EmptyKey(t *testing.T) {
	defer func() { _ = recover() }()
	New().Get("")
	t.Error("expected panic")
}

func TestValue_Set_Get(t *testing.T) {
	app := New()
	app.Set("k", "v")
	v, ok := app.Get("k")
	if !ok || v != "v" {
		t.Errorf("unexpected: %v, %v", v, ok)
	}
}

func TestValue_Get_Missing(t *testing.T) {
	_, ok := New().Get("missing")
	if ok {
		t.Error("expected missing")
	}
}

func TestValue_SetFunc(t *testing.T) {
	app := New()
	var counter int
	app.SetFunc("counter", func() any {
		counter++
		return counter
	})

	v1, _ := app.Get("counter")
	v2, _ := app.Get("counter")
	if v1.(int) != 1 || v2.(int) != 2 {
		t.Errorf("expected 1, 2; got %v, %v", v1, v2)
	}
}

func TestValue_MustGet_Success(t *testing.T) {
	orig := DefaultApp
	defer func() { DefaultApp = orig }()
	DefaultApp = New()
	DefaultApp.Set("k", "v")
	v := MustGet[string]("k")
	if v != "v" {
		t.Errorf("unexpected: %v", v)
	}
}

func TestValue_MustGet_Panics_Missing(t *testing.T) {
	orig := DefaultApp
	defer func() { DefaultApp = orig }()
	DefaultApp = New()

	defer func() { _ = recover() }()
	MustGet[string]("missing")
	t.Error("expected panic")
}

func TestValue_MustGet_Panics_TypeMismatch(t *testing.T) {
	orig := DefaultApp
	defer func() { DefaultApp = orig }()
	DefaultApp = New()
	DefaultApp.Set("k", 42)

	defer func() { _ = recover() }()
	MustGet[string]("k")
	t.Error("expected panic")
}

func TestValue_Get_Typed(t *testing.T) {
	orig := DefaultApp
	defer func() { DefaultApp = orig }()
	DefaultApp = New()
	DefaultApp.Set("k", "v")

	v, ok := Get[string]("k")
	if !ok || v != "v" {
		t.Errorf("unexpected: %v, %v", v, ok)
	}

	_, ok = Get[int]("k")
	if ok {
		t.Error("expected type mismatch")
	}

	_, ok = Get[string]("missing")
	if ok {
		t.Error("expected missing")
	}
}
