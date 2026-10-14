// pkg/tls/tlsutil.go

package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"pkg/config"

	"google.golang.org/grpc/credentials"
)

func load(
	cfg config.TLSConfig,
) (
	tls.Certificate,
	*x509.CertPool,
	error,
) {

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("load keypair: %w", err)
	}

	ca, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("read ca: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return tls.Certificate{}, nil, fmt.Errorf("invalid ca pem")
	}

	return cert, pool, nil
}

// ServerCredentials: mTLS؛ کلاینت بدون گواهی معتبر رد می‌شود.
func ServerCredentials(
	cfg config.TLSConfig,
) (
	credentials.TransportCredentials,
	error,
) {

	cert, pool, err := load(cfg)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}), nil
}

// ClientTLSConfig را برمی‌گرداند؛ ServerName در Dial برای هر مقصد تنظیم می‌شود.
func ClientTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	cert, pool, err := load(cfg)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}
