package hash

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	hashbytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return " ", err
	}

	return string(hashbytes), nil
}

func ChekPassword(hash, password string) bool {
	ifequiv := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return ifequiv == nil
}
