package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/unkmonster/go-kit/crypto/pem"
)

// GenerateRS256KeyPairPem 生成 PEM 编码的 RS256 密钥对

func GenerateRS256KeyPairPem() (pub, priv string, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 256*8)
	if err != nil {
		return "", "", fmt.Errorf("generate private key: %w", err)
	}

	privBytes, err := pem.EncodePKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", fmt.Errorf("EncodePKCS8PrivateKey: %w", err)
	}

	pubBytes, err := pem.EncodePKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("EncodePKIXPublicKey: %w", err)
	}

	return string(pubBytes), string(privBytes), nil
}

// GenerateSigningKey 根据 alg 生成一个 secret 或者 pem 格式的密钥对
func GenerateSigningKey(method string) (pubKey, privKey, secret *string, err error) {
	alg := jwt.GetSigningMethod(method)
	if alg == nil {
		return nil, nil, nil, fmt.Errorf("invalid signing method: %s", method)
	}

	if alg == jwt.SigningMethodRS256 {
		pub, priv, err := GenerateRS256KeyPairPem()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("GenerateRS256KeyPairPem: %w", err)
		}
		return &pub, &priv, nil, nil
	}
	err = fmt.Errorf("unsupported signing method: %s", alg.Alg())
	return
}
