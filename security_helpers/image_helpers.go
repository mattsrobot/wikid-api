package security_helpers

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"log"
	"os"
)

func VideoUrl(file string, width int, height int) (string, error) {

	sourceFile := os.Getenv("FILES_URL") + "/" + file

	options := fmt.Sprintf("%dx%d/0x0/%s", width, height, sourceFile)

	hmac, err := VideoHMAC(options)

	if err != nil {
		return "", err
	}

	return os.Getenv("VID_PROXY") + "/" + hmac + "/" + options, nil
}

func VideoHMAC(path string) (string, error) {

	signer := NewDefaultSigner(os.Getenv("VIDPROXY_KEY"))

	return signer.Sign(path), nil
}

func ImageUrl(file string, width int, height int) (string, error) {

	sourceFile := os.Getenv("PRIVATE_FILES_URL") + "/" + file

	options := fmt.Sprintf("/s:%d:%d:true:true/%s", width, height, base64.RawURLEncoding.EncodeToString([]byte(sourceFile)))

	hmac, err := ImageHMAC(options)

	if err != nil {
		return "", err
	}

	return os.Getenv("IMG_PROXY") + hmac + options, nil
}

func ImageHMAC(path string) (string, error) {

	var keyBin, saltBin []byte
	var err error

	if keyBin, err = hex.DecodeString(os.Getenv("IMGPROXY_KEY")); err != nil {
		log.Fatal(err)
	}

	if saltBin, err = hex.DecodeString(os.Getenv("IMGPROXY_SALT")); err != nil {
		log.Fatal(err)
	}

	mac := hmac.New(sha256.New, keyBin)
	mac.Write(saltBin)
	mac.Write([]byte(path))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("/%s", signature), nil
}

// Signer imagor URL signature signer
type Signer interface {
	Sign(path string) string
}

// NewDefaultSigner default signer using SHA1 with secret
func NewDefaultSigner(secret string) Signer {
	return NewHMACSigner(sha1.New, 0, secret)
}

// NewHMACSigner custom HMAC alg signer with secret and string length based truncate
func NewHMACSigner(alg func() hash.Hash, truncate int, secret string) Signer {
	return &hmacSigner{
		alg:      alg,
		truncate: truncate,
		secret:   []byte(secret),
	}
}

type hmacSigner struct {
	alg      func() hash.Hash
	truncate int
	secret   []byte
}

func (s *hmacSigner) Sign(path string) string {
	h := hmac.New(s.alg, s.secret)
	h.Write([]byte(path))
	sig := base64.URLEncoding.EncodeToString(h.Sum(nil))
	if s.truncate > 0 && len(sig) > s.truncate {
		return sig[:s.truncate]
	}
	return sig
}
