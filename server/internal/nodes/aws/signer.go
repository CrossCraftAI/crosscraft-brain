// Package aws provides AWS service integration nodes with SigV4 request signing.
package aws

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Signer implements AWS Signature Version 4 signing for HTTP requests.
// Reference: https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
type Signer struct {
	AccessKey string
	SecretKey string
	Region    string
	Service   string
}

// ServiceSigner creates a Signer pre-configured for a specific AWS service.
type ServiceSigner struct {
	Signer
}

// Sign signs an HTTP request with AWS SigV4.
func (s *Signer) Sign(req *http.Request, body []byte) error {
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	// Compute payload hash
	payloadHash := sha256Hex(body)

	// Set required headers
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.Host)
	if req.Header.Get("Content-Type") == "" && len(body) > 0 {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	// Canonical request
	canonicalHeaders, signedHeaders := canonicalHeadersAndSigned(req)
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL.Path),
		canonicalQuery(req.URL.RawQuery),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign
	credentialScope := dateStamp + "/" + s.Region + "/" + s.Service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Signing key
	signingKey := s.deriveSigningKey(dateStamp)

	// Signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Authorization header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.AccessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// deriveSigningKey computes the AWS signing key.
func (s *Signer) deriveSigningKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+s.SecretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(s.Region))
	kService := hmacSHA256(kRegion, []byte(s.Service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

// SignRequest is a convenience method that reads the body, signs, and rewinds.
func (s *Signer) SignRequest(req *http.Request) error {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	return s.Sign(req, body)
}

// SigningRoundTripper wraps an http.RoundTripper and signs AWS requests.
type SigningRoundTripper struct {
	Base http.RoundTripper
	Sig  *Signer
}

func (srt *SigningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid consuming the body
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		req.Body.Close()
	}
	req2 := req.Clone(req.Context())
	if body != nil {
		req2.Body = io.NopCloser(bytes.NewReader(body))
		req2.ContentLength = int64(len(body))
	}
	if body == nil {
		body = []byte{}
	}

	if err := srt.Sig.Sign(req2, body); err != nil {
		return nil, fmt.Errorf("aws sign: %w", err)
	}

	base := srt.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}

// --- helpers ---------------------------------------------------------------

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	// Encode path segments but not slashes
	parts := strings.Split(path, "/")
	encoded := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		encoded = append(encoded, urlEncodePath(p))
	}
	return "/" + strings.Join(encoded, "/")
}

func canonicalQuery(query string) string {
	if query == "" {
		return ""
	}
	parts := strings.Split(query, "&")
	sort.Strings(parts)
	encoded := make([]string, 0, len(parts))
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		key := urlEncode(kv[0])
		val := ""
		if len(kv) > 1 {
			val = urlEncode(kv[1])
		}
		encoded = append(encoded, key+"="+val)
	}
	return strings.Join(encoded, "&")
}

func canonicalHeadersAndSigned(req *http.Request) (string, string) {
	// Collect headers (lowercase)
	headers := map[string]string{}
	for k, v := range req.Header {
		lower := strings.ToLower(k)
		headers[lower] = strings.TrimSpace(strings.Join(v, ","))
	}

	// Sort by header name
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonParts, signedParts []string
	for _, k := range keys {
		canonParts = append(canonParts, k+":"+headers[k])
		signedParts = append(signedParts, k)
	}
	return strings.Join(canonParts, "\n") + "\n", strings.Join(signedParts, ";")
}

func urlEncode(s string) string {
	// Custom percent-encoding per AWS SigV4 spec
	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			buf.WriteByte(c)
		} else {
			buf.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return buf.String()
}

func urlEncodePath(s string) string {
	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' || c == '/' {
			buf.WriteByte(c)
		} else {
			buf.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return buf.String()
}
