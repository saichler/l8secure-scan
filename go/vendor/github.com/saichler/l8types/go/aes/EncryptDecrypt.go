/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package aes provides AES-256 encryption and decryption utilities for the Layer 8 system.
// Uses CFB (Cipher Feedback) mode with a random IV (Initialization Vector) for each encryption.
// Encrypted data is returned as base64-encoded strings for safe transmission.
package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// keyChars contains the character set used for generating human-readable AES keys.
var keyChars = []rune("abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// GenerateAES256Key generates a new random 32-character AES-256 key.
// The key uses alphanumeric characters for compatibility with text-based storage.
func GenerateAES256Key() string {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic("failed to generate random key: " + err.Error())
	}
	
	keyRunes := make([]rune, 32)
	for i, b := range key {
		keyRunes[i] = keyChars[int(b)%len(keyChars)]
	}
	return string(keyRunes)
}

// Encrypt encrypts data using AES-256 in CFB mode.
// A random IV is prepended to the ciphertext before base64 encoding.
// Parameters:
//   - dataToEncode: The plaintext data to encrypt
//   - key: A 32-character (256-bit) AES key
//
// Returns the base64-encoded ciphertext (IV + encrypted data).
func Encrypt(dataToEncode []byte, key string) (string, error) {
	keyBytes := []byte(key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	dataLen := len(dataToEncode)
	cipherdata := make([]byte, aes.BlockSize+dataLen)

	iv := cipherdata[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(cipherdata[aes.BlockSize:], dataToEncode)
	return base64.StdEncoding.EncodeToString(cipherdata), nil
}

// Decrypt decrypts a base64-encoded AES-256 ciphertext.
// Expects the IV to be prepended to the ciphertext (as produced by Encrypt).
// Parameters:
//   - stringToDecode: The base64-encoded ciphertext (IV + encrypted data)
//   - key: The same 32-character key used for encryption
//
// Returns the decrypted plaintext data.
func Decrypt(stringToDecode, key string) ([]byte, error) {
	encData, err := base64.StdEncoding.DecodeString(stringToDecode)
	if err != nil {
		return nil, err
	}
	if len(encData) < aes.BlockSize {
		return nil, errors.New("encrypted data does not have an iv spec")
	}
	
	keyBytes := []byte(key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	
	iv := encData[:aes.BlockSize]
	encData = encData[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	data := make([]byte, len(encData))
	cfb.XORKeyStream(data, encData)
	return data, nil
}
