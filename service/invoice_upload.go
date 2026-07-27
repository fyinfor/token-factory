package service

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const invoiceUploadSubPrefix = "invoices"

func isPDFUpload(file *multipart.FileHeader) bool {
	if file == nil {
		return false
	}
	if uploadFileExt(file.Filename) != ".pdf" {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(file.Header.Get("Content-Type")))
	if contentType == "" || contentType == "application/octet-stream" {
		return true
	}
	return contentType == "application/pdf"
}

func joinUploadPrefix(base, sub string) string {
	base = strings.Trim(strings.TrimSpace(base), "/")
	sub = strings.Trim(strings.TrimSpace(sub), "/")
	switch {
	case base != "" && sub != "":
		return base + "/" + sub
	case sub != "":
		return sub
	default:
		return base
	}
}

// UploadInvoiceFile 上传电子发票 PDF，按运营设置走本地存储或 OSS。
func UploadInvoiceFile(file *multipart.FileHeader, userID int) (string, error) {
	if !isPDFUpload(file) {
		return "", fmt.Errorf("仅支持 PDF 格式电子发票")
	}
	source, err := file.Open()
	if err != nil {
		return "", err
	}
	header := make([]byte, 5)
	_, readErr := io.ReadFull(source, header)
	closeErr := source.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(header, []byte("%PDF-")) {
		return "", fmt.Errorf("电子发票文件不是有效的 PDF")
	}
	cfg := operation_setting.GetOssSetting()
	if !cfg.Enabled {
		return "", fmt.Errorf("文件上传未启用，请先在运营设置中启用")
	}
	if cfg.StorageType == operation_setting.StorageTypeLocal {
		prefix, err := NormalizeLocalUploadPrefix(joinUploadPrefix(cfg.LocalObjectKeyPrefix, invoiceUploadSubPrefix))
		if err != nil {
			return "", err
		}
		return localUploadMultipartFileWithPrefix(file, userID, prefix)
	}
	if !operation_setting.IsOssUploadReady() {
		return "", ErrOssNotConfigured
	}
	prefix := joinUploadPrefix(cfg.ObjectKeyPrefix, invoiceUploadSubPrefix)
	return ossUploadMultipartFileWithPrefix(file, userID, prefix)
}
