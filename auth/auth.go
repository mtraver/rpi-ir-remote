// Package auth provides types and functions used for JWT generation and validation.
package auth

import (
	"crypto/sha256"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/proto"
)

// Claims is the set of claims, standard and custom additions, that are used in JWTs that authorize Actions.
type Claims struct {
	jwt.StandardClaims
	Device     string `json:"dev"`
	ActionHash string `json:"acsha"`
}

// Sum256 returns the SHA256 hash of the given proto as a hex string. It marshals
// the proto to the wire format and then hashes it.
func Sum256(pb proto.Message) (string, error) {
	b, err := proto.Marshal(pb)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
