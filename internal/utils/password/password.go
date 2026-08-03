package password

import (
	"crypto/rand"

	"golang.org/x/crypto/bcrypt"
)

const cost = bcrypt.DefaultCost

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Verify(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// Generate возвращает случайный пароль из n символов.
func Generate(n int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		// ponytail: modulo bias для пароля незначителен
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b), nil
}
