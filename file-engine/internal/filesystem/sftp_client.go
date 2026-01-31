package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SftpFs holds SFTP client config
type SftpFs struct {
	Addr       string // host:port
	User       string
	Password   string // optional, prefer key
	PrivateKey []byte // optional PEM
	Timeout    time.Duration
	BaseRoot   string // remote base path
	client     *sftp.Client
	sshClient  *ssh.Client
}

// NewSftpFs creates and connects to remote SFTP server
func NewSftpFs(addr, user, password string, privateKey []byte, baseRoot string) (*SftpFs, error) {
	cfg := &SftpFs{
		Addr:       addr,
		User:       user,
		Password:   password,
		PrivateKey: privateKey,
		Timeout:    10 * time.Second,
		BaseRoot:   filepath.Clean(baseRoot),
	}

	if err := cfg.connect(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// hostKeyCallback returns a HostKeyCallback implementation that must
// verify the server's host key against a trusted allow list.
// NOTE: This implementation currently does not have access to any
// configured host key material and therefore fails closed. It should
// be extended to load and check known host keys as appropriate for
// the deployment environment.
func (s *SftpFs) hostKeyCallback() (ssh.HostKeyCallback, error) {
	return nil, errors.New("host key verification is not configured")
}

func (s *SftpFs) connect() error {
	var auth []ssh.AuthMethod
	if len(s.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(s.PrivateKey)
		if err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else if s.Password != "" {
		auth = append(auth, ssh.Password(s.Password))
	} else {
		return errors.New("no auth method provided")
	}

	hostKeyCallback, err := s.hostKeyCallback()
	if err != nil {
		return fmt.Errorf("host key callback: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            s.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         s.Timeout,
	}

	conn, err := ssh.Dial("tcp", s.Addr, sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("sftp new client: %w", err)
	}
	s.client = client
	s.sshClient = conn
	return nil
}

func (s *SftpFs) Close() {
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.sshClient != nil {
		_ = s.sshClient.Close()
	}
}

// sanitize path on remote base
func (s *SftpFs) remoteJoin(parts ...string) string {
	joined := filepath.Join(parts...)
	clean := filepath.Clean(joined)
	if filepath.IsAbs(clean) {
		clean = filepath.Clean(clean)[1:]
	}
	return filepath.Join(s.BaseRoot, clean)
}

// CreateFolder creates directory recursively on remote server
func (s *SftpFs) CreateFolder(ctx context.Context, parts ...string) error {
	full := s.remoteJoin(parts...)
	if err := s.client.MkdirAll(full); err != nil {
		return fmt.Errorf("sftp mkdirall failed: %w", err)
	}
	return nil
}

// AtomicWriteFile writes file atomically on remote by writing to temp file then renaming
func (s *SftpFs) AtomicWriteFile(ctx context.Context, perm os.FileMode, data io.Reader, parts ...string) error {
	full := s.remoteJoin(parts...)
	dir := filepath.Dir(full)
	tmp := fmt.Sprintf("%s.tmp-%d", full, time.Now().UnixNano())
	// ensure dir exists
	if err := s.client.MkdirAll(dir); err != nil {
		return fmt.Errorf("sftp ensure dir: %w", err)
	}

	// write tmp file
	f, err := s.client.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp open tmp: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("sftp write tmp: %w", err)
	}
	if err := f.Chmod(perm); err != nil {
		// best-effort
		_ = err
	}
	// rename to final
	if err := s.client.Rename(tmp, full); err != nil {
		return fmt.Errorf("sftp rename: %w", err)
	}
	return nil
}

// MoveUploadedFile moves file on remote (rename fallback to copy)
func (s *SftpFs) MoveUploadedFile(ctx context.Context, srcParts []string, dstParts []string) error {
	src := s.remoteJoin(srcParts...)
	dst := s.remoteJoin(dstParts...)
	if err := s.client.Rename(src, dst); err == nil {
		return nil
	}
	// fallback copy
	srcF, err := s.client.Open(src)
	if err != nil {
		return fmt.Errorf("sftp open src: %w", err)
	}
	defer srcF.Close()
	dstF, err := s.client.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp open dst: %w", err)
	}
	defer dstF.Close()
	if _, err := io.Copy(dstF, srcF); err != nil {
		return fmt.Errorf("sftp copy: %w", err)
	}
	if err := s.client.Remove(src); err != nil {
		return fmt.Errorf("sftp remove src: %w", err)
	}
	return nil
}
