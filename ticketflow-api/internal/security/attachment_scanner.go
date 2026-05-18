package security

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrMaliciousAttachment = errors.New("attachment contains a malicious signature")

type AttachmentScanner interface {
	ScanFile(path string) error
}

type NoopAttachmentScanner struct{}

func (NoopAttachmentScanner) ScanFile(_ string) error {
	return nil
}

type SignatureAttachmentScanner struct {
	signatures [][]byte
}

func NewSignatureAttachmentScanner() SignatureAttachmentScanner {
	return SignatureAttachmentScanner{
		signatures: [][]byte{
			[]byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`),
		},
	}
}

func (s SignatureAttachmentScanner) ScanFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open attachment for scan: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 26<<20))
	if err != nil {
		return fmt.Errorf("read attachment for scan: %w", err)
	}

	for _, signature := range s.signatures {
		if bytes.Contains(data, signature) {
			return ErrMaliciousAttachment
		}
	}

	return nil
}
