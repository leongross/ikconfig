package test

import (
	"os"
	"testing"

	"github.com/leongross/extract-ikconfig/pkg/ikconfig"
)

func TestParseKonfig(t *testing.T) {
	kernel, err := ikconfig.NewKernelConfig("testdata/vmlinuz-linux", ikconfig.KERNEL_COMPRESSION_TYPE_GZIP)
	if err != nil {
		t.Errorf("Error creating new KernelConfig object: %v", err)
	}

	if _, err := os.Stat(kernel.PathDecompressed()); err != nil {
		t.Errorf("got %q, want nil", err)
	}

	configMap, err := kernel.ParseKernelConfig()
	if err != nil {
		t.Errorf("Error parsing kernel config: %v", err)
	}

	val, err := configMap.Get("CONFIG_CC_VERSION_TEXT")
	if err != nil {
		t.Errorf("Error getting value from config map: %v", err)
	}

	t.Logf("CONFIG_CC_VERSION_TEXT: %v", val)
}

func TestFindMagic(t *testing.T) {
	// the file zeroBin only contains 0 bytes, so this should fail
	offset, err := ikconfig.SearchBytes("testdata/zero.bin", []byte{0x01})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	if offset != 0 {
		t.Errorf("Expected offset 0, got %v", offset)
	}

	// the file zeroBin only contains 0 bytes, so this should work
	offset, err = ikconfig.SearchBytes("testdata/zero.bin", []byte{0x00})
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if offset != 0 {
		t.Errorf("Expected offset 0, got %v", offset)
	}

	// if the sequence is longer, the offset should be the same
	offset2, err := ikconfig.SearchBytes("testdata/zero.bin", []byte{0x00, 0x00})
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if offset2 != offset {
		t.Errorf("Expected offset %v = %v, got %v", offset, offset2, offset2)
	}

	// file magic.bin contains the magic bytes 0x13 0x37 at offset 1022
	offset, err = ikconfig.SearchBytes("testdata/magic.bin", []byte{0x13, 0x37})
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if offset != 1022 {
		t.Errorf("Expected offset 1022, got %v", offset)
	}

}
