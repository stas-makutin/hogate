package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type alexaCertificate struct {
	signing *x509.Certificate
	roots   *x509.CertPool
}

var alexaCertificates map[string]*alexaCertificate
var alexaCertificatesLock sync.Mutex

func init() {
	alexaCertificates = make(map[string]*alexaCertificate)
}

func newAlexaCertificate(pemData []byte) *alexaCertificate {
	if len(pemData) <= 0 {
		return nil
	}

	ac := &alexaCertificate{
		roots: x509.NewCertPool(),
	}
	for len(pemData) > 0 {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}
		if ac.signing == nil {
			ac.signing = certificate
		} else {
			ac.roots.AddCert(certificate)
		}
		pemData = rest
	}

	if !ac.valid() {
		return nil
	}

	return ac
}

func (ac *alexaCertificate) valid() bool {
	if ac.signing == nil {
		return false
	}

	opts := x509.VerifyOptions{
		DNSName: "echo-api.amazon.com",
		Roots:   ac.roots,
	}

	if _, err := ac.signing.Verify(opts); err != nil {
		return false
	}

	return true
}

func validateAlexaRequest(r *http.Request) bool {
	signatureCertUrl := r.Header.Get("SignatureCertChainUrl")
	signature, err := base64.StdEncoding.DecodeString(r.Header.Get("Signature-256"))
	if err != nil {
		return false
	}

	keyAlgorithm, key := signatureCertificate(signatureCertUrl)
	if keyAlgorithm == x509.UnknownPublicKeyAlgorithm {
		return false
	}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 32768))
	if err != nil {
		return false
	}
	bodyHash := sha256.Sum256(body)

	switch keyAlgorithm {
	case x509.RSA:
		if err := rsa.VerifyPKCS1v15(key.(*rsa.PublicKey), crypto.SHA256, bodyHash[:], signature); err != nil {
			return false
		}
	case x509.ECDSA:
		if !ecdsa.VerifyASN1(key.(*ecdsa.PublicKey), bodyHash[:], signature) {
			return false
		}
	case x509.Ed25519:
		if !ed25519.Verify(key.(ed25519.PublicKey), bodyHash[:], signature) {
			return false
		}
	default:
		return false
	}

	return true
}

func signatureCertificate(certificateUrl string) (x509.PublicKeyAlgorithm, any) {
	if certificateUrl == "" {
		return x509.UnknownPublicKeyAlgorithm, nil
	}
	certificateUrl, err := url.JoinPath(certificateUrl)
	if err != nil {
		return x509.UnknownPublicKeyAlgorithm, nil
	}
	url, err := url.Parse(certificateUrl)
	if err != nil {
		return x509.UnknownPublicKeyAlgorithm, nil
	}
	if url.Scheme != "https" {
		return x509.UnknownPublicKeyAlgorithm, nil
	}
	if !strings.EqualFold(url.Hostname(), "s3.amazonaws.com") {
		return x509.UnknownPublicKeyAlgorithm, nil
	}
	if !strings.HasPrefix(url.Path, "/echo.api/") {
		return x509.UnknownPublicKeyAlgorithm, nil
	}
	if url.Port() != "" && url.Port() != "443" {
		return x509.UnknownPublicKeyAlgorithm, nil
	}

	alexaCertificatesLock.Lock()
	defer alexaCertificatesLock.Unlock()

	certificate, ok := alexaCertificates[certificateUrl]
	if !ok || !certificate.valid() {
		response, err := (&http.Client{Timeout: time.Second * 2}).Get(certificateUrl)
		if err != nil {
			return x509.UnknownPublicKeyAlgorithm, nil
		}
		pemData, err := io.ReadAll(io.LimitReader(response.Body, 16384))
		if err != nil || len(pemData) <= 0 {
			return x509.UnknownPublicKeyAlgorithm, nil
		}
		certificate = newAlexaCertificate(pemData)
		if certificate == nil {
			return x509.UnknownPublicKeyAlgorithm, nil
		}
		if len(alexaCertificates) > 10 {
			for k := range alexaCertificates {
				delete(alexaCertificates, k)
				break
			}
		}
		alexaCertificates[certificateUrl] = certificate
	}

	return certificate.signing.PublicKeyAlgorithm, certificate.signing.PublicKey
}
