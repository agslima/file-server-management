package fsadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SftpFs is an adapter implementation backed by an SFTP remote filesystem.
type SftpFs struct {
	client   *sftp.Client
	sshConn  *ssh.Client
	baseRoot string
}

func NewSftpFs(addr, user, password string, privateKey []byte, baseRoot string) (*SftpFs, error) {
	auth, err := sftpAuth(password, privateKey)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := hostKeyCallback()
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("sftp client: %w", err)
	}

	return &SftpFs{client: client, sshConn: conn, baseRoot: filepath.Clean(baseRoot)}, nil
}

func sftpAuth(password string, privateKey []byte) ([]ssh.AuthMethod, error) {
	if len(privateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if password != "" {
		return []ssh.AuthMethod{ssh.Password(password)}, nil
	}
	return nil, errors.New("no sftp auth provided")
}

func hostKeyCallback() (ssh.HostKeyCallback, error) {
	return nil, errors.New("host key verification is not configured")
}

func (s *SftpFs) Close() {
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.sshConn != nil {
		_ = s.sshConn.Close()
	}
}

func (s *SftpFs) full(parts ...string) string {
	return filepath.Join(append([]string{s.baseRoot}, parts...)...)
}

func (s *SftpFs) CreateFolder(_ context.Context, parts ...string) error {
	return s.client.MkdirAll(s.full(parts...))
}

func (s *SftpFs) AtomicWriteFile(_ context.Context, perm uint32, data []byte, parts ...string) error {
	target := s.full(parts...)
	tmp := fmt.Sprintf("%s.tmp-%d", target, time.Now().UnixNano())
	f, err := s.client.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	_ = f.Chmod(os.FileMode(perm))
	if err := f.Close(); err != nil {
		return err
	}
	return s.client.Rename(tmp, target)
}

func (s *SftpFs) MoveUploadedFile(_ context.Context, src, dst []string) error {
	return s.client.Rename(s.full(src...), s.full(dst...))
}

func (s *SftpFs) Exists(parts ...string) (bool, error) {
	_, err := s.client.Stat(s.full(parts...))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
