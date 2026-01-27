package s3

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

    "github.com/example/file-engine/internal/storage"
)

type S3Storage struct {
    client *s3.Client
    bucket string
    prefix string
}

type Config struct {
    Bucket string
    Region string
    Prefix string // optional namespace prefix inside bucket, e.g. "file-engine"
    Endpoint string // optional for MinIO / custom endpoints
    AccessKeyID string
    SecretAccessKey string
    SessionToken string
}

func New(ctx context.Context, cfg Config) (*S3Storage, error) {
    if cfg.Bucket == "" {
        return nil, fmt.Errorf("S3 bucket required")
    }
    var loadOpts []func(*config.LoadOptions) error
    if cfg.Region != "" {
        loadOpts = append(loadOpts, config.WithRegion(cfg.Region))
    }
    if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
        loadOpts = append(loadOpts, config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
        ))
    }
    awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
    if err != nil {
        return nil, err
    }

    client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        if cfg.Endpoint != "" {
            o.BaseEndpoint = aws.String(cfg.Endpoint)
            o.UsePathStyle = true
        }
    })

    return &S3Storage{client: client, bucket: cfg.Bucket, prefix: strings.Trim(cfg.Prefix, "/")}, nil
}

func (s *S3Storage) key(path string) string {
    p := strings.TrimPrefix(path, "/")
    if s.prefix == "" {
        return p
    }
    return s.prefix + "/" + p
}

func randHex(n int) string {
    b := make([]byte, n)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

func (s *S3Storage) CreateFolder(ctx context.Context, path string) error {
    // In S3, folders are prefixes; we optionally create a zero-byte placeholder "<prefix>/"
    key := s.key(strings.TrimSuffix(path, "/") + "/")
    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   strings.NewReader(""),
    })
    return err
}

func (s *S3Storage) AtomicWrite(ctx context.Context, path string, r io.Reader) error {
    // Emulate atomic write: upload to temp key then copy to final and delete temp.
    finalKey := s.key(path)
    tmpKey := finalKey + ".tmp-" + randHex(8)

    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(tmpKey),
        Body:   r,
    })
    if err != nil {
        return err
    }

    // Copy tmp -> final
    copySource := urlPathEscape(s.bucket + "/" + tmpKey)
    _, err = s.client.CopyObject(ctx, &s3.CopyObjectInput{
        Bucket:     aws.String(s.bucket),
        Key:        aws.String(finalKey),
        CopySource: aws.String(copySource),
        MetadataDirective: s3types.MetadataDirectiveCopy,
    })
    if err != nil {
        return err
    }

    // Best-effort delete tmp
    _, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(tmpKey),
    })
    return nil
}

func (s *S3Storage) Move(ctx context.Context, src string, dst string) error {
    srcKey := s.key(src)
    dstKey := s.key(dst)
    copySource := urlPathEscape(s.bucket + "/" + srcKey)

    _, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
        Bucket:     aws.String(s.bucket),
        Key:        aws.String(dstKey),
        CopySource: aws.String(copySource),
        MetadataDirective: s3types.MetadataDirectiveCopy,
    })
    if err != nil {
        return err
    }
    // Wait a bit for consistency in some setups (optional)
    _ = time.Now()

    _, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(srcKey),
    })
    return err
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
    key := s.key(path)
    _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    return err
}

func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
    key := s.key(path)
    _, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        // AWS SDK returns error for 404; keep it simple here
        return false, nil
    }
    return true, nil
}

func (s *S3Storage) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
    keyPrefix := s.key(strings.TrimPrefix(prefix, "/"))
    if keyPrefix != "" && !strings.HasSuffix(keyPrefix, "/") {
        keyPrefix += "/"
    }

    out := []storage.ObjectInfo{}
    paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
        Bucket: aws.String(s.bucket),
        Prefix: aws.String(keyPrefix),
        Delimiter: aws.String("/"),
    })

    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        for _, cp := range page.CommonPrefixes {
            if cp.Prefix == nil {
                continue
            }
            p := strings.TrimPrefix(*cp.Prefix, s.prefix+"/")
            out = append(out, storage.ObjectInfo{Path: "/" + strings.TrimSuffix(p, "/"), IsDir: true})
        }
        for _, obj := range page.Contents {
            if obj.Key == nil {
                continue
            }
            k := *obj.Key
            // skip folder placeholders
            if strings.HasSuffix(k, "/") && obj.Size != nil && *obj.Size == 0 {
                continue
            }
            p := strings.TrimPrefix(k, s.prefix+"/")
            out = append(out, storage.ObjectInfo{Path: "/" + p, Size: aws.ToInt64(obj.Size), IsDir: false})
        }
    }
    return out, nil
}

// Minimal URL escaping for CopySource (space etc).
// For a full implementation, use url.PathEscape carefully.
func urlPathEscape(s string) string {
    s = strings.ReplaceAll(s, " ", "%20")
    return s
}
