package redisx_test

import (
	"testing"

	"github.com/tozogabee/restaurant-system/pkg/redisx"
)

func TestNamespacedKeysCannotCollideAcrossServices(t *testing.T) {
	menu := redisx.NewNamespace("menu")
	order := redisx.NewNamespace("order")

	if got, want := menu.Key("item", "42"), "menu:item:42"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}
	if menu.Key("item", "42") == order.Key("item", "42") {
		t.Fatal("two services produced the same key for the same id")
	}
}

func TestNamespacedKeyWithNoParts(t *testing.T) {
	if got := redisx.NewNamespace("menu").Key(); got != "menu" {
		t.Fatalf("Key() = %q, want %q", got, "menu")
	}
}

func TestIsMissIgnoresNil(t *testing.T) {
	if redisx.IsMiss(nil) {
		t.Fatal("IsMiss(nil) = true, want false")
	}
}
