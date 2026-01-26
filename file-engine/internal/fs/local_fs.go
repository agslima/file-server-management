package fs
import("context";"io";"os";"path/filepath")
type LocalFS struct{Base string}
func NewLocalFS(b string)*LocalFS{return &LocalFS{Base:b}}
func(l*LocalFS)full(p string)string{return filepath.Join(l.Base,filepath.Clean("/"+p))}
func(l*LocalFS)CreateFolder(ctx context.Context,p string)error{return os.MkdirAll(l.full(p),0755)}
func(l*LocalFS)AtomicWrite(ctx context.Context,p string,r io.Reader)error{f:=l.full(p);t:=f+".tmp";o,_:=os.Create(t);io.Copy(o,r);o.Close();return os.Rename(t,f)}
