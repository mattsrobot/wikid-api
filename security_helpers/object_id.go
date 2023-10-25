package security_helpers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+os.Getenv("SALT")))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password+os.Getenv("SALT")), 10)
	return string(bytes), err
}

func GetAESDecrypted(encrypted string) ([]byte, error) {
	key := os.Getenv("AES_KEY")
	iv := os.Getenv("AES_IV")

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)

	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher([]byte(key))

	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("block size cant be zero")
	}

	mode := cipher.NewCBCDecrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, ciphertext)
	// ciphertext = PKCS5UnPadding(ciphertext)

	return ciphertext, nil
}

func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func GetAESEncrypted(plaintext string) (string, error) {
	key := os.Getenv("AES_KEY")
	iv := os.Getenv("AES_IV")

	var plainTextBlock []byte
	length := len(plaintext)

	if length%16 != 0 {
		extendBlock := 16 - (length % 16)
		plainTextBlock = make([]byte, length+extendBlock)
		copy(plainTextBlock[length:], bytes.Repeat([]byte{uint8(extendBlock)}, extendBlock))
	} else {
		plainTextBlock = make([]byte, length)
	}

	copy(plainTextBlock, plaintext)
	block, err := aes.NewCipher([]byte(key))

	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, len(plainTextBlock))
	mode := cipher.NewCBCEncrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, plainTextBlock)
	str := base64.StdEncoding.EncodeToString(ciphertext)

	return str, nil
}

func Decode(encoded string) (uint64, string) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)

	if err != nil {
		slog.Error("Decode error for authorization 💀",
			slog.String("error", err.Error()))

		return 0, ""
	}

	decrypted, err := GetAESDecrypted(string(decoded[:]))

	if err != nil {
		slog.Error("Decode error for authorization 💀",
			slog.String("error", err.Error()))

		return 0, ""
	}

	split := strings.Split(string(decrypted[:]), "/")

	if len(split) != 3 {
		slog.Error("Decode error for authorization 💀 len error")

		return 0, ""
	}

	id, err := strconv.ParseUint(split[0], 10, 64)

	if err != nil {
		slog.Error("Decode error for authorization 💀",
			slog.String("error", err.Error()))

		return 0, ""
	}

	return id, split[1]
}

func Encode(id uint64, object string, objectSalt string) string {
	fullId := fmt.Sprintf("%d/%s/%s", id, object, objectSalt)
	encrypted, err := GetAESEncrypted(fullId)

	if err != nil {
		slog.Error("Encode error for authorization 💀",
			slog.String("error", err.Error()))

		return ""
	}

	encoded := base64.RawURLEncoding.EncodeToString([]byte(encrypted))

	return string(encoded[:])
}
