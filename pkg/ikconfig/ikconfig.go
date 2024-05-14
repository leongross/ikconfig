// This package re-implements https://github.com/torvalds/linux/blob/master/scripts/extract-ikconfig in
// pure golang to be imported in other golang projects.

package ikconfig

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

var (
	// 'IKCFG_ST\037\213\010'
	KERNEL_CONFIG_MAGIC = []byte{'I', 'K', 'C', 'F', 'G'}
)

type KernelConfigEnabled int

const (
	KERNEL_CONFIG_BUILT_IN KernelConfigEnabled = iota // =y
	KERNEL_CONFIG_LOADABLE                            // =m
)

// Represents the
type KernelConfigMap map[string]string

func (k *KernelConfigMap) Get(key string) (string, error) {
	val, ok := (*k)[key]
	if !ok {
		return "", fmt.Errorf("key %q not found", key)
	}
	return val, nil
}

// All supported kernel compression types for kernel version 6.7.1
// CONFIG_HAVE_KERNEL_GZIP
// CONFIG_HAVE_KERNEL_BZIP2
// CONFIG_HAVE_KERNEL_LZMA
// CONFIG_HAVE_KERNEL_XZ
// CONFIG_HAVE_KERNEL_LZO
// CONFIG_HAVE_KERNEL_LZ4
// CONFIG_HAVE_KERNEL_ZSTD
type KernelCompressionType int

const (
	KERNEL_COMPRESSION_TYPE_GZIP KernelCompressionType = iota // None
	KERNEL_COMPRESSION_TYPE_BZIP2
	KERNEL_COMPRESSION_TYPE_LZMA
	KERNEL_COMPRESSION_TYPE_XZ
	KERNEL_COMPRESSION_TYPE_LZO
	KERNEL_COMPRESSION_TYPE_LZ4
	KERNEL_COMPRESSION_TYPE_ZSTD
	KERNEL_COMPRESSION_TYPE_NONE
	KERNEL_COMPRESSION_TYPE_UNKNOWN
)

func (k KernelCompressionType) Magic() []byte {
	return [...][]byte{
		{0x1f, 0x8b}, // GZIP
		{0x5c, 0x33, 0x37, 0x35, 0x37, 0x7a, 0x58, 0x59, 0x00}, // XZ
		{0x42, 0x5a, 0x68}, // bunzip2
	}[k]
}

type KernelConfig struct {
	path             string                `json:"path,omitempty"`
	pathDecompressed string                `json:"path_decompressed,omitempty"`
	compressionType  KernelCompressionType `json:"compression_type,omitempty"`
}

func (k *KernelConfig) Path() string {
	return k.path
}

func (k *KernelConfig) PathDecompressed() string {
	return k.pathDecompressed
}

func (k *KernelConfig) CompressionType() KernelCompressionType {
	return k.compressionType
}

// Find a magic string in the provided file
// Return the offset of the magic string, err if not found or unknown
func (k *KernelConfig) findKernelConfigMagic() (uint, error) {
	return SearchBytes(k.path, KERNEL_CONFIG_MAGIC)
}

func (k *KernelConfig) findCompressionMagic() (uint, error) {
	return SearchBytes(k.path, k.compressionType.Magic())
}

// Create a new KernelConfig object based on the provided path
// If the compression algorithm is known it can be passed to the constructor
// to speed up the later decompression time
// The path to the decompressed file is stored in the object
func (k *KernelConfig) decompress() error {
	// if the type is unknown, try to guess it

	tmpdir, err := os.MkdirTemp("", "ikconfig")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %w", err)
	}

	k.pathDecompressed = tmpdir + "/vmlinux"
	decomp, err := os.Create(k.pathDecompressed)
	if err != nil {
		return fmt.Errorf("error opening decompressed file: %w", err)
	}
	defer decomp.Close()

	kernel, err := os.Open(k.path)
	if err != nil {
		return fmt.Errorf("error opening kernel file: %w", err)
	}

	// find offset of magic values
	// pos, err := k.findMagic()
	if err != nil {
		return fmt.Errorf("error finding magic string for %v: %w", k.compressionType, err)
	}

	switch k.compressionType {
	case KERNEL_COMPRESSION_TYPE_NONE:
		os.Link(k.path, k.pathDecompressed)

	case KERNEL_COMPRESSION_TYPE_UNKNOWN:
		// try all the compression types
	case KERNEL_COMPRESSION_TYPE_GZIP:
		gzipReader, err := gzip.NewReader(kernel)
		if err != nil {
			return fmt.Errorf("error creating gzip reader: %w", err)
		}
		// write the decompressed file to the decompressed path
		_, err = io.Copy(decomp, gzipReader)
		if err != nil {
			return fmt.Errorf("error decompressing gzip: %w", err)
		}

	case KERNEL_COMPRESSION_TYPE_BZIP2:
		bzipReader := bzip2.NewReader(kernel)
		_, err := io.Copy(decomp, bzipReader)
		if err != nil {
			return fmt.Errorf("error decompressing bzip2: %w", err)
		}

	case KERNEL_COMPRESSION_TYPE_LZMA:
	case KERNEL_COMPRESSION_TYPE_XZ:
		buf := bytes.Buffer{}
		_, err := io.Copy(&buf, kernel)
		if err != nil {
			return fmt.Errorf("error copying kernel file to buffer: %w", err)
		}

		xzReader, err := xz.NewReader(&buf)
		if err != nil {
			return fmt.Errorf("error creating xz reader: %w", err)
		}
		if _, err := io.Copy(decomp, xzReader); err != nil {
			return fmt.Errorf("error decompressing xz: %w", err)
		}

	case KERNEL_COMPRESSION_TYPE_ZSTD:
		zstdReader, err := zstd.NewReader(kernel)
		if err != nil {
			return fmt.Errorf("error creating zstd reader: %w", err)
		}
		if _, err := io.Copy(decomp, zstdReader); err != nil {
			return fmt.Errorf("error decompressing zstd: %w", err)
		}
	case KERNEL_COMPRESSION_TYPE_LZO:
	case KERNEL_COMPRESSION_TYPE_LZ4:
	default:
		return fmt.Errorf("unknown/unsupported compression type: %v", k.compressionType)
	}
	return nil
}

func NewKernelConfig(path string, compression KernelCompressionType) (*KernelConfig, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("file %q not found: %w", path, err)
	}
	return &KernelConfig{
		path:            path,
		compressionType: compression,
	}, nil
}

// Parse the provided kernel config as a map of enabled features
func (k *KernelConfig) ParseKernelConfig() (*KernelConfigMap, error) {
	var configBuf bytes.Buffer
	err := k.decompress()
	if err != nil {
		return nil, fmt.Errorf("error decompressing kernel config: %w", err)
	}

	// after the file is decompressed, get the kernel config from the end of the file
	magicOffset, err := k.findKernelConfigMagic()
	if err != nil {
		return nil, fmt.Errorf("error finding magic string in kernel config: %w", err)
	}

	// the config file is a zip file with the kernel config at the end
	// extract the zip from the end and decompress it
	configDecompressed, err := os.ReadFile(k.pathDecompressed)
	if err != nil {
		return nil, fmt.Errorf("error reading kernel config: %w", err)
	}
	configBuf.Write(configDecompressed[magicOffset+8:])

	// unzip and write to file $PWD/config
	out, err := os.Open("config")
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}

	gzipReader, err := gzip.NewReader(&configBuf)
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %w", err)
	}

	io.Copy(out, gzipReader)
	return &KernelConfigMap{}, nil
}
