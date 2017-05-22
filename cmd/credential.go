/*
 * Minio Cloud Storage, (C) 2015, 2016, 2017 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	accessKeyMinLen       = 5
	accessKeyMaxLen       = 20
	secretKeyMinLen       = 8
	secretKeyMaxLenAmazon = 40
	alphaNumericTable     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphaNumericTableLen  = byte(len(alphaNumericTable))
)

var (
	errInvalidAccessKeyLength = errors.New("Invalid access key, access key should be 5 to 20 characters in length")
	errInvalidSecretKeyLength = errors.New("Invalid secret key, secret key should be 8 to 40 characters in length")
)
var secretKeyMaxLen = secretKeyMaxLenAmazon

// isAccessKeyValid - validate access key for right length.
func isAccessKeyValid(accessKey string) bool {
	return len(accessKey) >= accessKeyMinLen && len(accessKey) <= accessKeyMaxLen
}

// isSecretKeyValid - validate secret key for right length.
func isSecretKeyValid(secretKey string) bool {
	return len(secretKey) >= secretKeyMinLen && len(secretKey) <= secretKeyMaxLen
}

// credential container for access and secret keys.
type credential struct {
	AccessKey string    `xml:"AccessKeyId,omitempty" json:"accessKey,omitempty"`
	SecretKey string    `xml:"SecretAccessKey,omitempty" json:"secretKey,omitempty"`
	Expiry    time.Time `xml:"Expiration,omitempty" json:"expiry,omitempty"`

	secretKeyHash []byte
}

// IsValid - returns whether credential is valid or not.
func (cred credential) IsValid() bool {
	return isAccessKeyValid(cred.AccessKey) && isSecretKeyValid(cred.SecretKey)
}

// Equals - returns whether two credentials are equal or not.
func (cred credential) Equal(ccred credential) bool {
	if !ccred.IsValid() {
		return false
	}

	if cred.secretKeyHash == nil {
		secretKeyHash, err := bcrypt.GenerateFromPassword([]byte(cred.SecretKey), bcrypt.DefaultCost)
		if err != nil {
			errorIf(err, "Unable to generate hash of given password")
			return false
		}

		cred.secretKeyHash = secretKeyHash
	}

	return (cred.AccessKey == ccred.AccessKey &&
		bcrypt.CompareHashAndPassword(cred.secretKeyHash, []byte(ccred.SecretKey)) == nil)
}

func createCredentialWithExpiry(accessKey, secretKey string, expiry time.Time) (cred credential, err error) {
	if !isAccessKeyValid(accessKey) {
		err = errInvalidAccessKeyLength
	} else if !isSecretKeyValid(secretKey) {
		err = errInvalidSecretKeyLength
	} else {
		var secretKeyHash []byte
		secretKeyHash, err = bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
		if err == nil {
			cred.AccessKey = accessKey
			cred.SecretKey = secretKey
			cred.secretKeyHash = secretKeyHash
		}
	}
	if !expiry.IsZero() {
		cred.Expiry = expiry
	}
	return cred, err
}

func createCredential(accessKey, secretKey string) (cred credential, err error) {
	return createCredentialWithExpiry(accessKey, secretKey, timeSentinel)
}

func getNewCredentialWithExpiry(expiry time.Time) (credential, error) {
	// Generate access key.
	keyBytes := make([]byte, accessKeyMaxLen)
	_, err := rand.Read(keyBytes)
	fatalIf(err, "Unable to generate access key.")
	for i := 0; i < accessKeyMaxLen; i++ {
		keyBytes[i] = alphaNumericTable[keyBytes[i]%alphaNumericTableLen]
	}
	accessKey := string(keyBytes)

	// Generate secret key.
	keyBytes = make([]byte, secretKeyMaxLen)
	_, err = rand.Read(keyBytes)
	if err != nil {
		return credential{}, err
	}
	secretKey := string([]byte(base64.StdEncoding.EncodeToString(keyBytes))[:secretKeyMaxLen])

	cred, err := createCredentialWithExpiry(accessKey, secretKey, expiry)
	if err != nil {
		return credential{}, err
	}
	return cred, nil
}

// Initialize a new credential object
func mustGetNewCredential() credential {
	cred, err := getNewCredentialWithExpiry(timeSentinel)
	fatalIf(err, "Unable to generate new credential.")
	return cred
}
