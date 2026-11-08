package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"hash"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	saltCharacters          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	saltLength              = 16
	scryptCost              = 32768
	scryptBlockSize         = 8
	scryptParallelism       = 1
	scryptKeyLength         = 64
	defaultPBKDF2Iterations = 600000
)

func HashPassword(password string) (string, error) {
	salt, err := randomSalt()
	if err != nil {
		return "", err
	}
	digest, err := scrypt.Key([]byte(password), []byte(salt), scryptCost, scryptBlockSize, scryptParallelism, scryptKeyLength)
	if err != nil {
		return "", err
	}
	method := "scrypt:" + strconv.Itoa(scryptCost) + ":" + strconv.Itoa(scryptBlockSize) + ":" + strconv.Itoa(scryptParallelism)
	return method + "$" + salt + "$" + hex.EncodeToString(digest), nil
}

func CheckPassword(storedHash string, password string) bool {
	parts := strings.SplitN(storedHash, "$", 3)
	if len(parts) != 3 {
		return false
	}
	method, salt, expectedHex := parts[0], parts[1], parts[2]
	computed, ok := computeHash(method, salt, password)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHex)) == 1
}

func computeHash(method string, salt string, password string) (string, bool) {
	arguments := strings.Split(method, ":")
	switch arguments[0] {
	case "scrypt":
		cost, blockSize, parallelism := scryptCost, scryptBlockSize, scryptParallelism
		if len(arguments) == 4 {
			var errors [3]error
			cost, errors[0] = strconv.Atoi(arguments[1])
			blockSize, errors[1] = strconv.Atoi(arguments[2])
			parallelism, errors[2] = strconv.Atoi(arguments[3])
			for _, err := range errors {
				if err != nil {
					return "", false
				}
			}
		}
		digest, err := scrypt.Key([]byte(password), []byte(salt), cost, blockSize, parallelism, scryptKeyLength)
		if err != nil {
			return "", false
		}
		return hex.EncodeToString(digest), true
	case "pbkdf2":
		hashName := "sha256"
		iterations := defaultPBKDF2Iterations
		if len(arguments) > 1 {
			hashName = arguments[1]
		}
		if len(arguments) > 2 {
			parsed, err := strconv.Atoi(arguments[2])
			if err != nil {
				return "", false
			}
			iterations = parsed
		}
		newHash, size, ok := hashFunction(hashName)
		if !ok {
			return "", false
		}
		return hex.EncodeToString(pbkdf2.Key([]byte(password), []byte(salt), iterations, size, newHash)), true
	}
	return "", false
}

func hashFunction(name string) (func() hash.Hash, int, bool) {
	switch name {
	case "sha256":
		return sha256.New, sha256.Size, true
	case "sha512":
		return sha512.New, sha512.Size, true
	case "sha1":
		return sha1.New, sha1.Size, true
	}
	return nil, 0, false
}

func randomSalt() (string, error) {
	salt := make([]byte, saltLength)
	limit := big.NewInt(int64(len(saltCharacters)))
	for index := range salt {
		choice, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		salt[index] = saltCharacters[choice.Int64()]
	}
	return string(salt), nil
}
