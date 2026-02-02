package ssh

import (
	"context"
	"fmt"
	"io"
	"path"

	"golang.org/x/crypto/ssh"
)

type Session interface {
	Run(ctx context.Context, cmd string, stdout, stderr io.Writer) error
	CopyFile(ctx context.Context, content io.Reader, remotePath string, mode uint32) error
	Close() error
}

type session struct {
	conn *ssh.Client
	host Host
}

func newSession(conn *ssh.Client, host Host) (Session, error) {
	return &session{
		conn: conn,
		host: host,
	}, nil
}

func (s *session) Run(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
	sess, err := s.conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sess.Close()

	sess.Stdout = stdout
	sess.Stderr = stderr

	// Run command
	if err := sess.Run(cmd); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (s *session) CopyFile(ctx context.Context, content io.Reader, remotePath string, mode uint32) error {
	// Read all content into memory first
	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	// Use atomic write: write to temp file, then move
	tmpPath := fmt.Sprintf("/tmp/hades-%s", path.Base(remotePath))

	// Create new session for file write
	sess, err := s.conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sess.Close()

	// Write content to temp file using cat
	stdin, err := sess.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	writeCmd := fmt.Sprintf("cat > %s && chmod %o %s", tmpPath, mode, tmpPath)
	if err := sess.Start(writeCmd); err != nil {
		return fmt.Errorf("failed to start write command: %w", err)
	}

	if _, err := stdin.Write(data); err != nil {
		stdin.Close()
		return fmt.Errorf("failed to write data: %w", err)
	}
	stdin.Close()

	if err := sess.Wait(); err != nil {
		return fmt.Errorf("write command failed: %w", err)
	}

	// Move temp file to final location (atomic)
	sess2, err := s.conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session for mv: %w", err)
	}
	defer sess2.Close()

	mvCmd := fmt.Sprintf("mv %s %s", tmpPath, remotePath)
	if err := sess2.Run(mvCmd); err != nil {
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	return nil
}

func (s *session) Close() error {
	// Connection is managed by the client, not individual sessions
	return nil
}
