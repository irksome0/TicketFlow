package security

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSignatureAttachmentScannerRejectsEICAR(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signature.txt")
	content := []byte("log line with TEST-MALWARE-SIGNATURE marker")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	scanner := SignatureAttachmentScanner{
		signatures: [][]byte{[]byte("TEST-MALWARE-SIGNATURE")},
	}
	err := scanner.ScanFile(path)
	if !errors.Is(err, ErrMaliciousAttachment) {
		t.Fatalf("expected ErrMaliciousAttachment, got %v", err)
	}
}

func TestSignatureAttachmentScannerAllowsCleanFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clean.txt")
	if err := os.WriteFile(path, []byte("clean diagnostic log"), 0600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	if err := NewSignatureAttachmentScanner().ScanFile(path); err != nil {
		t.Fatalf("expected clean file, got %v", err)
	}
}
