package go_libs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

const bitSize = 4096 // RSA keysize

// Sha256bytes2bytes converts a byte sequence into a SHA-256-based digest of it.
// The output for this application is the same on the commadn line with:
// curl -q localhost:8888 | jq -c .Data | tr -d '\n' | shasum -a256
// The added newline must be removed. Alternatively, gnu-sed can be used instad of tr:
// gsed -Ez 's/\n$//'
// The complete JSON return structure only consists of US-ASCII characters. So potential
// different escaping for special characters do not have to be considered.
func Sha256bytes2bytes(bytes []byte) []byte {
	//return fmt.Sprintf("%x", sha256.Sum256(bytes)) returning type array [32]byte which must usually be converted
	msgHash := sha256.New()
	_, _ = msgHash.Write(bytes) // todo no error handling, but error is very unlike
	return msgHash.Sum(nil)
}

// SignByteArray returns a signature for the given digest or returns an error
func SignByteArray(key *rsa.PrivateKey, digest []byte) ([]byte, error) {
	var opts rsa.PSSOptions
	opts.SaltLength = rsa.PSSSaltLengthAuto
	if key == nil { // no signing
		return nil, nil
	}
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest, &opts)
	if err != nil {
		return nil, errors.New(CurrentFunctionName() + ":" + err.Error())
	}
	return signature, nil
}

// SignByteArray2Base64 signs a byte array by calling SignByteArray but returns the
// signature as a base64-encoded string.
func SignByteArray2Base64(key *rsa.PrivateKey, digest []byte) (string, error) {
	sig, err := SignByteArray(key, digest)
	if err != nil {
		return "", errors.New(CurrentFunctionName() + ":" + err.Error())
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyByteArray verifies a digital signature (digest). If no error is returned,
// then the verification was successful. Furthermore, it recalculates the digest of the
// message. It should result in the same digest as the digitally signed one.
func VerifyByteArray(key *rsa.PublicKey, digest []byte, msg string) error {
	var opts rsa.PSSOptions
	opts.SaltLength = rsa.PSSSaltLengthAuto
	if key == nil {
		return errors.New(CurrentFunctionName() + ":Error, public key is nil")
	}
	if digest == nil {
		return errors.New(CurrentFunctionName() + ":Error, digest is nil")
	}
	plaintestDigest := Sha256bytes2bytes([]byte(msg))
	CondDebugln(CurrentFunctionName() + ", recalculated digest for msg: " + fmt.Sprintf("%x", plaintestDigest))
	return rsa.VerifyPSS(key, crypto.SHA256, plaintestDigest, digest, &opts)
}

// VerifyBase64String accepts a base64 encoded string as the signature.
// It decodes the signature and calls VerifyByteArray.
func VerifyBase64String(key *rsa.PublicKey, b64 string, msg string) error {
	signatureByte, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return errors.New(CurrentFunctionName() + ":Error, decoding base64 string")
	}
	return VerifyByteArray(key, signatureByte, msg)
}

// =======================================================================================
// = Key Loading and Signing

// ParsePrivateKey load a PEM-encoded RSA private key from a buffer. The function does not try
// to read multiple keys from the byte array. Only the first PEM block is processed.
func ParsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(der)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New(CurrentFunctionName() + ":failed to decode PEM block containing private key")
	}
	pub, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New(CurrentFunctionName() + ":failed to parse PEM block:" + err.Error())
	}
	return pub, nil
}

// LoadPrivateKey load a PEM-encoded RSA private key from a file
func LoadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.New(CurrentFunctionName() + ":reading file:" + err.Error())
	}
	return ParsePrivateKey(buf)
}

// ParsePublicKey load a PEM-encoded RSA public key from a buffer. The function does not try
// to read multiple keys from the byte array. Only the first PEM block is processed.
func ParsePublicKey(der []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(der)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New(CurrentFunctionName() + ":failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, errors.New(CurrentFunctionName() + ":failed to parse PEM block:" + err.Error())
	}
	return pub, nil
}

// LoadPublicKey load a PEM-encoded RSA public key from a file
func LoadPublicKey(filename string) (*rsa.PublicKey, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.New(CurrentFunctionName() + ":reading file:" + err.Error())
	}
	return ParsePublicKey(buf)
}

// TODO VerifySignature
// TODO EncryptAES256
// TODO DecryptAES256

// =======================================================================================
// = Keypair Generation

// WritePrivateKey converts the key to PEM format and writes them to a file.
func WritePrivateKey(file *os.File, privKey *rsa.PrivateKey) error {
	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}
	if err := pem.Encode(file, privateKey); err != nil {
		return errors.New(CurrentFunctionName() + ":" + err.Error())
	}
	return nil
}

// WritePublicKey converts the public key to PEM format and writes them to the file.
func WritePublicKey(file *os.File, pubKey *rsa.PublicKey) error {
	asn1Bytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return errors.New(CurrentFunctionName() + ":1:" + err.Error())
	}
	CondDebugln(fmt.Sprintf("Length of Public Key: %d", len(asn1Bytes)))
	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}
	if err := pem.Encode(file, pemkey); err != nil {
		return errors.New(CurrentFunctionName() + ":2:" + err.Error())
	}
	return nil
}

// createRSAKeyPair2 creates the keypair and calls the functions to write the keys to the files
func createRSAKeyPair2(privKeyFile *os.File, pubKeyFile *os.File) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return errors.New(CurrentFunctionName() + "key creation:" + err.Error())
	}
	if err := WritePrivateKey(privKeyFile, privateKey); err != nil {
		return errors.New(CurrentFunctionName() + "private key writing:" + err.Error())
	}
	if err := WritePublicKey(pubKeyFile, &privateKey.PublicKey); err != nil {
		return errors.New(CurrentFunctionName() + "public key writing:" + err.Error())
	}
	return nil
}

// CreateRSAKeyPair2File checks if the 2 required files do not exist and can be created sucessfully. Then,
// it transfers control to createKeyPairError2.
func CreateRSAKeyPair2File(outfileName string) error {
	var privKeyFile *os.File
	var pubKeyFile *os.File
	var err error

	const publicKeyFileSuffix = ".pub"

	if _, err = os.Stat(outfileName); err == nil {
		return errors.New("Private key file " + outfileName + " already exists.")
	}
	if _, err = os.Stat(outfileName + publicKeyFileSuffix); err == nil {
		return errors.New("Public key file " + outfileName + " already exists.")
	}
	if privKeyFile, err = os.Create(outfileName); err != nil {
		return errors.New("Error creating private key file " + outfileName + ":" + err.Error())
	}
	if pubKeyFile, err = os.Create(outfileName + publicKeyFileSuffix); err != nil {
		return errors.New("Error creating private key file " + outfileName + ":" + err.Error())
	}
	defer privKeyFile.Close()
	defer pubKeyFile.Close()
	return createRSAKeyPair2(privKeyFile, pubKeyFile)
}

// CreateRSAKeyPair creates an RSA 4096-bit key-pair. This function makes only partly sense,
// as the private key always contains the public key.
func CreateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, nil, errors.New(CurrentFunctionName() + "key creation:" + err.Error())
	}
	return privateKey, &privateKey.PublicKey, nil
}

// EOF
