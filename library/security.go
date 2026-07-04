package library

import (
	"crypto/rand"
	"errors"
	"github.com/speps/go-hashids/v2"
	"math/big"
)

const salt = "24def0aa4033f2e36c6ae149f1362ca5cfba8cff"

func GenerateFriendlyCode(id uint64) (string, error) {
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 6                                 // E.g., ensures ID 1 doesn't just become "A"
	hd.Alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Optional: custom alphabet

	h, err := hashids.NewWithData(hd)
	if err != nil {
		return "", err
	}

	return h.EncodeInt64([]int64{int64(id)})
}

func DecodeFriendlyCode(hash string) (uint64, error) {
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 6
	hd.Alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

	h, err := hashids.NewWithData(hd)
	if err != nil {
		return 0, err
	}

	decoded, err := h.DecodeInt64WithError(hash)
	if err != nil || len(decoded) == 0 {
		return 0, errors.New("invalid code")
	}

	return uint64(decoded[0]), nil
}

// GenerateSecureRef creates a secure 7-character string
func GenerateSecureRef(length int) string {
	charset := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}
