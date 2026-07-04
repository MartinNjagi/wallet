package library

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	cachedCert *x509.Certificate
	certMutex  sync.RWMutex
)

// EncryptInitiatorPasswordS3 fetches the cert from S3 (caching it in memory) and encrypts the password.
func EncryptInitiatorPasswordS3(ctx context.Context, s3Client *s3.Client, bucket, key, password string) (string, error) {
	// 1. Check if we already have it in memory
	certMutex.RLock()
	cert := cachedCert
	certMutex.RUnlock()

	// 2. Cache miss: Fetch from S3
	if cert == nil {
		certMutex.Lock()

		// Double-check locking (in case another goroutine fetched it while we waited for the lock)
		if cachedCert == nil {
			out, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				certMutex.Unlock()
				return "", fmt.Errorf("failed to fetch certificate from S3: %w", err)
			}
			defer out.Body.Close()

			certBytes, err := io.ReadAll(out.Body)
			if err != nil {
				certMutex.Unlock()
				return "", fmt.Errorf("failed to read certificate body: %w", err)
			}

			block, _ := pem.Decode(certBytes)
			if block == nil {
				certMutex.Unlock()
				return "", fmt.Errorf("failed to parse certificate PEM")
			}

			parsedCert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				certMutex.Unlock()
				return "", fmt.Errorf("failed to parse x509 certificate: %w", err)
			}
			cachedCert = parsedCert
		}

		cert = cachedCert
		certMutex.Unlock()
	}

	// 3. Encrypt the password using the public key from the certificate
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, cert.PublicKey.(*rsa.PublicKey), []byte(password))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt password: %w", err)
	}

	// 4. Return as Base64 string
	return base64.StdEncoding.EncodeToString(encrypted), nil
}
