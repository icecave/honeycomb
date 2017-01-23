package loader

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"path"

	minio "github.com/minio/minio-go"
)

const defaultEndpoint = "s3.amazonaws.com"

// S3Loader loads certificates and keys from an Amazon S3 bucket (or compatible
// object store).
type S3Loader struct {
	Bucket   string
	BasePath string
	S3Client *minio.Client
	Logger   *log.Logger
}

// LoadCertificate reads an x509 certificate from a file in PEM format and
// returns the parsed certificate.
func (loader *S3Loader) LoadCertificate(_ context.Context, filePath string) (*x509.Certificate, error) {
	bytes, err := loader.readPEM(filePath)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(bytes)
}

// LoadPrivateKey reads an RSA certificate from a file in PEM formt and
// returns the parsed key.
func (loader *S3Loader) LoadPrivateKey(_ context.Context, filePath string) (*rsa.PrivateKey, error) {
	bytes, err := loader.readPEM(filePath)
	if err != nil {
		return nil, err
	}

	return x509.ParsePKCS1PrivateKey(bytes)
}

func (loader *S3Loader) readPEM(filePath string) ([]byte, error) {
	p := path.Join(loader.BasePath, filePath)

	obj, err := loader.S3Client.GetObject(loader.Bucket, p)
	if err != nil {
		loader.logError(p, err)
		return nil, err
	}

	raw, err := ioutil.ReadAll(obj)
	if err != nil {
		loader.logError(p, err)
		return nil, err
	}

	block, unused := pem.Decode(raw)
	if len(unused) > 0 {
		err := errors.New("unused data in PEM file")
		loader.logError(p, err)
		return nil, err
	}

	if loader.Logger != nil {
		loader.Logger.Printf(
			"Loaded '%s' from '%s' S3 bucket",
			filePath,
			loader.Bucket,
		)
	}

	return block.Bytes, nil
}

func (loader *S3Loader) logError(filePath string, err error) {
	if loader.Logger != nil {
		loader.Logger.Printf(
			"Unable to load '%s' from '%s' S3 bucket, %s",
			filePath,
			loader.Bucket,
			err,
		)
	}
}
