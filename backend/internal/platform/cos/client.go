package cos

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tcos "github.com/tencentyun/cos-go-sdk-v5"
)

// Client 腾讯云 COS 客户端（私有 Bucket，服务端持有 Secret）。
type Client struct {
	cfg    Config
	inner  *tcos.Client
	bucket string
	region string
}

// NewClient 根据配置创建客户端。
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("cos bucket url: %w", err)
	}
	base := &tcos.BaseURL{BucketURL: bucketURL}
	inner := tcos.NewClient(base, &http.Client{
		Transport: &tcos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})
	return &Client{cfg: cfg, inner: inner, bucket: cfg.Bucket, region: cfg.Region}, nil
}

// PutBytes 上传对象正文。
func (c *Client) PutBytes(ctx context.Context, objectKey string, contentType string, body []byte) error {
	if strings.TrimSpace(objectKey) == "" {
		return fmt.Errorf("objectKey is required")
	}
	opt := &tcos.ObjectPutOptions{}
	if contentType != "" {
		opt.ObjectPutHeaderOptions = &tcos.ObjectPutHeaderOptions{ContentType: contentType}
	}
	_, err := c.inner.Object.Put(ctx, objectKey, bytes.NewReader(body), opt)
	return err
}

// PutReader 流式上传（Content-Length 可选，-1 表示未知）。
func (c *Client) PutReader(ctx context.Context, objectKey, contentType string, r io.Reader, contentLength int64) error {
	if strings.TrimSpace(objectKey) == "" {
		return fmt.Errorf("objectKey is required")
	}
	opt := &tcos.ObjectPutOptions{}
	if contentType != "" {
		opt.ObjectPutHeaderOptions = &tcos.ObjectPutHeaderOptions{ContentType: contentType}
	}
	if contentLength >= 0 {
		if opt.ObjectPutHeaderOptions == nil {
			opt.ObjectPutHeaderOptions = &tcos.ObjectPutHeaderOptions{}
		}
		opt.ObjectPutHeaderOptions.ContentLength = contentLength
	}
	_, err := c.inner.Object.Put(ctx, objectKey, r, opt)
	return err
}

// PresignedGetURL 生成短时 GET 预签名 URL；filename 用于 Content-Disposition（可为空）。
func (c *Client) PresignedGetURL(ctx context.Context, objectKey, filename string, ttl time.Duration) (string, error) {
	return c.presignedGetURL(ctx, objectKey, filename, ttl, true)
}

// PresignedInlineGetURL 在线预览用（inline disposition，不含 CI 处理）。图片等非 PDF 类型使用。
func (c *Client) PresignedInlineGetURL(ctx context.Context, objectKey, filename string, ttl time.Duration) (string, error) {
	return c.presignedGetURL(ctx, objectKey, filename, ttl, false)
}

// PresignedPDFInlineGetURL PDF 浏览器预览：同时签名 disposition 与 content-type，便于 CORS fetch + PDF.js。
func (c *Client) PresignedPDFInlineGetURL(ctx context.Context, objectKey, filename string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	opt := &tcos.PresignedURLOptions{Query: &url.Values{}}
	opt.Query.Set(
		"response-content-disposition",
		fmt.Sprintf(`inline; filename="%s"`, sanitizeFilename(filename)),
	)
	opt.Query.Set("response-content-type", "application/pdf")
	u, err := c.inner.Object.GetPresignedURL(ctx, http.MethodGet, objectKey, c.cfg.SecretID, c.cfg.SecretKey, ttl, opt, false)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PresignedDocPreviewURL 腾讯云 CI 文档预览（html）；需 Bucket 开通数据万象 doc-preview。
func (c *Client) PresignedDocPreviewURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	opt := &tcos.PresignedURLOptions{Query: &url.Values{}}
	opt.Query.Set("ci-process", "doc-preview")
	opt.Query.Set("dstType", "html")
	u, err := c.inner.Object.GetPresignedURL(ctx, http.MethodGet, objectKey, c.cfg.SecretID, c.cfg.SecretKey, ttl, opt, false)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) presignedGetURL(ctx context.Context, objectKey, filename string, ttl time.Duration, attachment bool) (string, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	opt := &tcos.PresignedURLOptions{}
	if strings.TrimSpace(filename) != "" {
		opt.Query = &url.Values{}
		kind := "inline"
		if attachment {
			kind = "attachment"
		}
		opt.Query.Set("response-content-disposition", fmt.Sprintf(`%s; filename="%s"`, kind, sanitizeFilename(filename)))
	}
	u, err := c.inner.Object.GetPresignedURL(ctx, http.MethodGet, objectKey, c.cfg.SecretID, c.cfg.SecretKey, ttl, opt, false)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `"`, "'")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	if name == "" {
		return "download"
	}
	return name
}

// Head 检查对象是否存在并返回 Content-Length（若响应头有）。
func (c *Client) Head(ctx context.Context, objectKey string) (size int64, exists bool, err error) {
	resp, err := c.inner.Object.Head(ctx, objectKey, nil)
	if err != nil {
		if isNotFound(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if resp != nil {
		size = resp.ContentLength
	}
	return size, true, nil
}

// GetBytes 下载整个对象（仅用于小文件/探针；大文件后续走流式或签名 URL）。
func (c *Client) GetBytes(ctx context.Context, objectKey string) ([]byte, error) {
	resp, err := c.inner.Object.Get(ctx, objectKey, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ObjectHTTPResponse COS GET 原始响应，供 Gin 流式转发（含 Range / 206）。
type ObjectHTTPResponse struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

// GetObjectHTTP 按 objectKey 拉取对象；rangeHeader 非空时透传 bytes Range（预签名 URL 不支持浏览器 Range 时的服务端替代）。
func (c *Client) GetObjectHTTP(ctx context.Context, objectKey, rangeHeader string) (*ObjectHTTPResponse, error) {
	if strings.TrimSpace(objectKey) == "" {
		return nil, fmt.Errorf("objectKey is required")
	}
	opt := &tcos.ObjectGetOptions{}
	if strings.TrimSpace(rangeHeader) != "" {
		opt.Range = rangeHeader
	}
	resp, err := c.inner.Object.Get(ctx, objectKey, opt)
	if err != nil {
		return nil, err
	}
	status := http.StatusOK
	if resp != nil && resp.StatusCode > 0 {
		status = resp.StatusCode
	}
	return &ObjectHTTPResponse{StatusCode: status, Header: resp.Header, Body: resp.Body}, nil
}

// Delete 删除对象（幂等：不存在时不视为错误）。
func (c *Client) Delete(ctx context.Context, objectKey string) error {
	_, err := c.inner.Object.Delete(ctx, objectKey)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if tcos.IsNotFoundError(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "nosuchkey")
}
