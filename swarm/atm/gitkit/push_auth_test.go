package gitkit

import (
	"errors"
	// "fmt"
	"io"
	"os"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	transportssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	sshcore "golang.org/x/crypto/ssh"
)

type mockSigner struct{}

func (mockSigner) PublicKey() sshcore.PublicKey {
	return mockPublicKey{}
}

func (mockSigner) Sign(rand io.Reader, _ []byte) (*sshcore.Signature, error) {
	return &sshcore.Signature{Blob: []byte("mock-sig")}, nil
}

type mockPublicKey struct{}

func (mockPublicKey) Type() string {
	return "mock"
}

func (mockPublicKey) Marshal() []byte {
	return []byte("mock-pubkey")
}

func (mockPublicKey) Verify(_ []byte, _ *sshcore.Signature) error {
	return nil
}

func TestPreparePushAuth(t *testing.T) {
	// Backup original functions and envs
	origAgent := sshAgentAuth
	origSigner := sshSigner
	origEnvs := map[string]string{}
	for _, key := range []string{"GIT_TOKEN", "GIT_USERNAME", "GIT_PASSWORD", "GIT_SSH_KEY"} {
		if v := os.Getenv(key); v != "" {
			origEnvs[key] = v
		}
		os.Unsetenv(key)
	}

	// Mock agent to always fail
	sshAgentAuth = func(_ string) (transport.AuthMethod, error) {
		return nil, errors.New("no ssh agent")
	}
	// Mock signer to always succeed
	sshSigner = func(_ []byte) (sshcore.Signer, error) {
		return mockSigner{}, nil
	}

	defer func() {
		sshAgentAuth = origAgent
		sshSigner = origSigner
		for k, v := range origEnvs {
			os.Setenv(k, v)
		}
	}()

	tests := []struct {
		name     string
		remote   string
		params   AuthParams
		setEnvs  map[string]string
		wantType string // "http_token", "http_basic", "ssh_pubkeys", "nil"
		wantErr  bool
	}{
		{
			name:     "HTTPS token param",
			remote:   "https://github.com/user/repo.git",
			params:   AuthParams{Token: "ghp_token123"},
			wantType: "http_token",
			wantErr:  false,
		},
		{
			name:     "HTTPS token env",
			remote:   "https://github.com/user/repo.git",
			params:   AuthParams{},
			setEnvs:  map[string]string{"GIT_TOKEN": "ghp_token456"},
			wantType: "http_token",
			wantErr:  false,
		},
		{
			name:     "HTTPS username/password param",
			remote:   "https://github.com/user/repo.git",
			params:   AuthParams{Username: "user", Password: "pass"},
			wantType: "http_basic",
			wantErr:  false,
		},
		{
			name:     "HTTPS username/password env",
			remote:   "https://github.com/user/repo.git",
			params:   AuthParams{},
			setEnvs:  map[string]string{"GIT_USERNAME": "user", "GIT_PASSWORD": "pass"},
			wantType: "http_basic",
			wantErr:  false,
		},
		{
			name:     "SSH key param",
			remote:   "git@github.com:user/repo.git",
			params:   AuthParams{SSHKey: "dummy-key-content"},
			wantType: "ssh_pubkeys",
			wantErr:  false,
		},
		{
			name:     "SSH GIT_SSH_KEY env",
			remote:   "ssh://git@github.com:user/repo.git",
			params:   AuthParams{},
			setEnvs:  map[string]string{"GIT_SSH_KEY": "dummy-key-content"},
			wantType: "ssh_pubkeys",
			wantErr:  false,
		},
		{
			name:     "SSH no auth",
			remote:   "git@github.com:user/repo.git",
			params:   AuthParams{},
			wantType: "nil",
			wantErr:  false,
		},
		{
			name:     "empty remote",
			remote:   "",
			params:   AuthParams{},
			wantType: "nil",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test envs
			for k, v := range tt.setEnvs {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.setEnvs {
					os.Unsetenv(k)
				}
			}()

			auth, err := preparePushAuth(tt.remote, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			switch tt.wantType {
			case "nil":
				if auth != nil {
					t.Errorf("expected nil auth, got %T", auth)
				}
			case "http_token":
				ba, ok := auth.(*http.BasicAuth)
				if !ok {
					t.Errorf("expected *http.BasicAuth with x-access-token, got %T: %v", auth, auth)
					return
				}
				token := tt.params.Token
				if token == "" {
					token = os.Getenv("GIT_TOKEN")
				}
				if ba.Username != "x-access-token" || ba.Password != token {
					t.Errorf("expected token auth x-access-token/%s, got %s/%s", token, ba.Username, ba.Password)
				}
			case "http_basic":
				ba, ok := auth.(*http.BasicAuth)
				if !ok {
					t.Errorf("expected *http.BasicAuth, got %T", auth)
					return
				}
				username := tt.params.Username
				if username == "" {
					username = os.Getenv("GIT_USERNAME")
				}
				password := tt.params.Password
				if password == "" {
					password = os.Getenv("GIT_PASSWORD")
				}
				if ba.Username != username || ba.Password != password {
					t.Errorf("expected basic auth %s/%s, got %s/%s", username, password, ba.Username, ba.Password)
				}
			case "ssh_pubkeys":
				pk, ok := auth.(*transportssh.PublicKeys)
				if !ok {
					t.Errorf("expected *transportssh.PublicKeys, got %T: %v", auth, auth)
					return
				}
				if pk.User != "git" {
					t.Errorf("expected user 'git', got %q", pk.User)
				}
			}
		})
	}
}
