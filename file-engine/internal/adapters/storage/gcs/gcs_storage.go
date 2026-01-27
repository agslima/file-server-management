package gcs

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io"
    "strings"

    "cloud.google.com/go/storage"
    "google.golang.org/api/iterator"

    istorage "github.com/example/file-engine/internal/storage"
)

type GCSStorage struct {
    client *storage.Client
    bucket string
    prefix string
}

type Config struct {
    Bucket string
    Prefix string // optional namespace prefix inside bucket
}

func New(ctx context.Context, cfg Config) (*GCSStorage, error) {
    if cfg.Bucket == "" {
        return nil, fmt.Errorf("GCS bucket required")
    }
    c, err := storage.NewClient(ctx)
    if err != nil {
        return nil, err
    }
    return &GCSStorage{client: c, bucket: cfg.Bucket, prefix: strings.Trim(cfg.Prefix, "/")}, nil
}

func (g *GCSStorage) key(path string) string {
    p := strings.TrimPrefix(path, "/")
    if g.prefix == "" {
        return p
    }
    return g.prefix + "/" + p
}

func randHex(n int) string {
    b := make([]byte, n)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

func (g *GCSStorage) CreateFolder(ctx context.Context, path string) error {
    // Create a zero-byte placeholder object ending with "/"
    key := g.key(strings.TrimSuffix(path, "/") + "/")
    w := g.client.Bucket(g.bucket).Object(key).NewWriter(ctx)
    if err := w.Close(); err != nil {
        return err
    }
    return nil
}

func (g *GCSStorage) AtomicWrite(ctx context.Context, path string, r io.Reader) error {
    finalKey := g.key(path)
    tmpKey := finalKey + ".tmp-" + randHex(8)

    // upload tmp
    tw := g.client.Bucket(g.bucket).Object(tmpKey).NewWriter(ctx)
    if _, err := io.Copy(tw, r); err != nil {
        _ = tw.Close()
        return err
    }
    if err := tw.Close(); err != nil {
        return err
    }

    // copy to final
    src := g.client.Bucket(g.bucket).Object(tmpKey)
    dst := g.client.Bucket(g.bucket).Object(finalKey)
    if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
        return err
    }

    // best-effort delete tmp
    _ = src.Delete(ctx)
    return nil
}

func (g *GCSStorage) Move(ctx context.Context, src string, dst string) error {
    srcKey := g.key(src)
    dstKey := g.key(dst)

    srcObj := g.client.Bucket(g.bucket).Object(srcKey)
    dstObj := g.client.Bucket(g.bucket).Object(dstKey)

    if _, err := dstObj.CopierFrom(srcObj).Run(ctx); err != nil {
        return err
    }
    return srcObj.Delete(ctx)
}

func (g *GCSStorage) Delete(ctx context.Context, path string) error {
    key := g.key(path)
    return g.client.Bucket(g.bucket).Object(key).Delete(ctx)
}

func (g *GCSStorage) Exists(ctx context.Context, path string) (bool, error) {
    key := g.key(path)
    _, err := g.client.Bucket(g.bucket).Object(key).Attrs(ctx)
    if err != nil {
        return false, nil
    }
    return true, nil
}

func (g *GCSStorage) List(ctx context.Context, prefix string) ([]istorage.ObjectInfo, error) {
    keyPrefix := g.key(strings.TrimPrefix(prefix, "/"))
    if keyPrefix != "" && !strings.HasSuffix(keyPrefix, "/") {
        keyPrefix += "/"
    }

    it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: keyPrefix})
    out := []istorage.ObjectInfo{}

    for {
        obj, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, err
        }
        k := obj.Name
        // Skip placeholder dirs
        if strings.HasSuffix(k, "/") && obj.Size == 0 {
            continue
        }
        p := k
        if g.prefix != "" {
            p = strings.TrimPrefix(p, g.prefix+"/")
        }
        out = append(out, istorage.ObjectInfo{Path: "/" + p, Size: obj.Size, IsDir: false})
    }
    return out, nil
}
