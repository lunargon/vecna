package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func GenerateKeyPair(name string) (privatePath, publicPath string, err error) {
	home, _ := os.UserHomeDir()
	keyDir := filepath.Join(home, ".ssh", "vecna")
	os.MkdirAll(keyDir, 0700)

	privatePath = filepath.Join(keyDir, name)
	publicPath = privatePath + ".pub"

	if _, err := os.Stat(privatePath); err == nil {
		return privatePath, publicPath, nil
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateFile, err := os.OpenFile(privatePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("failed to create private key: %w", err)
	}
	defer privateFile.Close()

	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	if err := os.WriteFile(publicPath, publicKeyBytes, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write public key: %w", err)
	}

	return privatePath, publicPath, nil
}

func DeployPublicKey(host Host, password, publicKeyPath string) error {
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	sshDir := "~/.ssh"
	authorizedKeys := sshDir + "/authorized_keys"

	cmd := fmt.Sprintf(
		"mkdir -p %s && chmod 700 %s && echo '%s' >> %s && chmod 600 %s",
		sshDir, sshDir, string(publicKey), authorizedKeys, authorizedKeys,
	)

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to deploy key: %w", err)
	}

	return nil
}
