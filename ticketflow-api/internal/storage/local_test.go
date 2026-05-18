package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLocalStorageSaveAndOpen(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStorage(dir)

	key := "uploads/org/ticket/file.txt"
	if err := store.Save(context.Background(), key, strings.NewReader("ticketflow"), "text/plain"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	file, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(content) != "ticketflow" {
		t.Fatalf("content = %q, want %q", content, "ticketflow")
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	store := NewLocalStorage(t.TempDir())

	err := store.Save(context.Background(), "../escape.txt", strings.NewReader("x"), "text/plain")
	if !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("Save() error = %v, want os.ErrInvalid", err)
	}
}
