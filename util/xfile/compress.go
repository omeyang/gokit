package xfile

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/omeyang/gokit/middleware/pool"

	"golang.org/x/sync/errgroup"
)

// Compressor 压缩和解压缩接口
type Compressor interface {
	Compress(param *CompressParam) error
	Decompress(param *DecompressParam) error
}

// CompressType 压缩类型
type CompressType string

const (
	// ZipCompressType .zip格式
	ZipCompressType = ".zip"
	// TarGzCompressType .tar.gz 格式
	TarGzCompressType = ".tar.gz"
	// GzCompressType .gz格式
	GzCompressType = ".gz"
)

// CompressParam 压缩参数
type CompressParam struct {
	Source      string
	Destination string
	RenameFunc  func(string, string) string // 重命名函数 防止已经有了同名的目标文件
	Format      CompressType                // 指定压缩格式，例如 ".zip", ".tar.gz", ".gz"
	BufferSize  int                         // 缓冲区大小，以字节为单位
	PoolSize    int                         // 临时对象池大小
}

// DecompressParam 解压缩参数
type DecompressParam struct {
	Source      string
	Destination string
	BufferSize  int // 缓冲区大小，以字节为单位
	PoolSize    int // 临时对象池大小
}

// FileCompressor 文件压缩实例
type FileCompressor struct{}

// Compress 文件压缩方法
func (fc *FileCompressor) Compress(param *CompressParam) error {
	if param.RenameFunc == nil {
		param.RenameFunc = DefaultRenameFunc
	}
	baseName := filepath.Base(param.Source)
	ext := param.Format
	if !filepath.IsAbs(param.Source) {
		param.Source, _ = filepath.Abs(param.Source)
	}
	if param.Destination == "" {
		param.Destination = "."
	}
	baseTargetName := filepath.Join(param.Destination, baseName)
	compressedFilename := param.RenameFunc(baseTargetName, string(ext))
	compressedFile, err := os.OpenFile(compressedFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer compressedFile.Close()
	bufferSize := param.BufferSize
	if bufferSize <= 0 {
		bufferSize = 16 * 1024 // 默认16KB
	}
	poolSize := param.PoolSize
	if poolSize <= 0 {
		poolSize = 10 // 默认临时对象池大小
	}
	bufFactory := pool.NewBufferFactory(bufferSize)
	bufPool := pool.NewBatchPool[[]byte](bufFactory, poolSize) // 显式指定泛型类型
	switch ext {
	case GzCompressType:
		return compressGz(compressedFile, param.Source, bufPool)
	case TarGzCompressType:
		return compressTarGz(compressedFile, param.Source, bufPool)
	case ZipCompressType:
		return compressZip(compressedFile, param.Source, bufPool)
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}
}

// compressGz 压缩为 .gz 格式
func compressGz(compressedFile *os.File, source string, bufPool *pool.BatchPool[[]byte]) error {
	writer := gzip.NewWriter(compressedFile)
	defer writer.Close()
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	buf := bufPool.Get()
	defer bufPool.Put(buf, &pool.BufferFactory{BufferSize: len(buf)})
	_, err = io.CopyBuffer(writer, file, buf)
	return err
}

// compressTarGz 压缩为 .tar.gz 格式
func compressTarGz(compressedFile *os.File, source string, bufPool *pool.BatchPool[[]byte]) error {
	gw := gzip.NewWriter(compressedFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	var g errgroup.Group
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(path, filepath.Dir(source)+string(filepath.Separator))
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.IsDir() {
			path := path // 避免闭包问题
			g.Go(func() error {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				buf := bufPool.Get()
				defer bufPool.Put(buf, &pool.BufferFactory{BufferSize: len(buf)})
				if _, err := io.CopyBuffer(tw, file, buf); err != nil {
					return err
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// compressZip 压缩为 .zip 格式
func compressZip(compressedFile *os.File, source string, bufPool *pool.BatchPool[[]byte]) error {
	zipWriter := zip.NewWriter(compressedFile)
	defer zipWriter.Close()
	var g errgroup.Group
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(path, filepath.Dir(source)+string(filepath.Separator))
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			path := path // 避免闭包问题
			g.Go(func() error {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				buf := bufPool.Get()
				defer bufPool.Put(buf, &pool.BufferFactory{BufferSize: len(buf)})
				if _, err := io.CopyBuffer(writer, file, buf); err != nil {
					return err
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// Decompress 文件解压缩方法
func (fc *FileCompressor) Decompress(param *DecompressParam) error {
	bufferSize := param.BufferSize
	if bufferSize <= 0 {
		bufferSize = 16 * 1024 // 默认16KB
	}
	poolSize := param.PoolSize
	if poolSize <= 0 {
		poolSize = 10 // 默认临时对象池
	}
	bufFactory := pool.NewBufferFactory(bufferSize)
	bufPool := pool.NewBatchPool[[]byte](bufFactory, poolSize)
	if strings.HasSuffix(param.Source, GzCompressType) && !strings.HasSuffix(param.Source, TarGzCompressType) {
		// 修改解压缩路径为文件路径
		return decompressGz(param.Source, param.Destination, bufPool)
	} else if strings.HasSuffix(param.Source, TarGzCompressType) {
		return decompressTarGz(param.Source, param.Destination, bufPool)
	} else if strings.HasSuffix(param.Source, ZipCompressType) {
		return decompressZip(param.Source, param.Destination, bufPool)
	} else {
		return fmt.Errorf("unsupported format: %s", param.Source)
	}
}

// decompressGz 解压缩 .gz 格式
func decompressGz(source, destination string, bufPool *pool.BatchPool[[]byte]) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	outFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer outFile.Close()

	buf := bufPool.Get()
	defer bufPool.Put(buf, &pool.BufferFactory{BufferSize: len(buf)})

	_, err = io.CopyBuffer(outFile, reader, buf)
	return err
}

// decompressTarGz 解压缩 .tar.gz 格式
func decompressTarGz(source, destination string, bufPool *pool.BatchPool[[]byte]) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destination, header.Name)
		// 修正路径分隔符问题
		target = filepath.Clean(target)
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		} else {
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}

			buf := bufPool.Get()
			if _, err := io.CopyBuffer(outFile, tr, buf); err != nil {
				outFile.Close()
				bufPool.Put(buf, pool.NewBufferFactory(len(buf)))
				return err
			}
			bufPool.Put(buf, pool.NewBufferFactory(len(buf)))
			outFile.Close()
		}
	}
	return nil
}

// decompressZip 解压缩 .zip 格式
func decompressZip(source, destination string, bufPool *pool.BatchPool[[]byte]) error {
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destination, f.Name)
		// 修正路径分隔符问题
		fpath = filepath.Clean(fpath)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		buf := bufPool.Get()
		if _, err := io.CopyBuffer(outFile, rc, buf); err != nil {
			outFile.Close()
			rc.Close()
			bufPool.Put(buf, pool.NewBufferFactory(len(buf)))
			return err
		}
		bufPool.Put(buf, pool.NewBufferFactory(len(buf)))
		outFile.Close()
		rc.Close()
	}

	return nil
}
