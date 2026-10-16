package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logincrypto"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/passkeyservice"
	"github.com/gin-gonic/gin"
)

type passkeyEnvelope struct {
	Result      any                   `json:"result"`
	Code        component.Status      `json:"code"`
	MessageCode component.MessageCode `json:"messageCode,omitempty"`
}

func writePasskeySuccess(c *gin.Context, value any, code component.MessageCode) {
	c.JSON(http.StatusOK, passkeyEnvelope{Result: value, Code: component.SUCCESS, MessageCode: code})
}

func writePasskeyFailure(c *gin.Context, code component.MessageCode) {
	c.JSON(http.StatusOK, passkeyEnvelope{Result: nil, Code: component.FAIL, MessageCode: code})
}

func decodePasskeyJSON(c *gin.Context, destination any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writePasskeyFailure(c, component.MessageRequestInvalidFormat)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writePasskeyFailure(c, component.MessageRequestInvalidFormat)
		return false
	}
	return true
}

// PasskeyStatus exposes only the deployment enable bit so clients can hide
// unavailable UI without learning the RP's signing material.
func PasskeyStatus(c *gin.Context) {
	enabled, err := passkeyservice.IsEnabled()
	if err != nil {
		slog.Error("passkey configuration invalid", "error", err)
		enabled = false
	}
	writePasskeySuccess(c, gin.H{"enabled": enabled}, component.MessageOperationSuccess)
}

func AppleWebCredentialsAssociation(c *gin.Context) {
	cfg, err := passkeyservice.LoadConfig()
	if err != nil || cfg.AppleAppID() == "" {
		c.JSON(http.StatusNotFound, gin.H{"webcredentials": gin.H{"apps": []string{}}})
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"webcredentials": gin.H{"apps": []string{cfg.AppleAppID()}}})
}

func AndroidDigitalAssetLinks(c *gin.Context) {
	cfg, err := passkeyservice.LoadConfig()
	if err != nil || cfg.AndroidPackageName == "" || len(cfg.AndroidFingerprints) == 0 {
		c.Header("Cache-Control", "public, max-age=3600")
		c.JSON(http.StatusOK, []any{})
		return
	}
	fingerprints := make([]string, 0, len(cfg.AndroidFingerprints))
	for _, fingerprint := range cfg.AndroidFingerprints {
		compact := strings.ReplaceAll(fingerprint, ":", "")
		if len(compact) != 64 {
			continue
		}
		var formatted strings.Builder
		for i := 0; i < len(compact); i += 2 {
			if i > 0 {
				formatted.WriteByte(':')
			}
			formatted.WriteString(strings.ToUpper(compact[i : i+2]))
		}
		fingerprints = append(fingerprints, formatted.String())
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, []any{gin.H{
		"relation": []string{"delegate_permission/common.get_login_creds"},
		"target": gin.H{
			"namespace":                "android_app",
			"package_name":             cfg.AndroidPackageName,
			"sha256_cert_fingerprints": fingerprints,
		},
	}})
}

func PasskeyRegistrationBegin(c *gin.Context) {
	var body struct {
		Proof string `json:"proof"`
	}
	if !decodePasskeyJSON(c, &body) {
		return
	}
	if body.Proof == "" {
		writePasskeyFailure(c, component.MessageRequestInvalidParams)
		return
	}
	result, err := passkeyservice.BeginRegistration(c.GetUint64("userId"), c.GetString("currentJti"), body.Proof)
	if err != nil {
		writePasskeyServiceError(c, err, false)
		return
	}
	writePasskeySuccess(c, result, component.MessageOperationSuccess)
}

func PasskeyRegistrationFinish(c *gin.Context) {
	var body struct {
		CeremonyID string          `json:"ceremonyId"`
		Name       string          `json:"name"`
		Credential json.RawMessage `json:"credential"`
	}
	if !decodePasskeyJSON(c, &body) {
		return
	}
	request := c.Request.Clone(c.Request.Context())
	request.Body = io.NopCloser(bytesReader(body.Credential))
	result, err := passkeyservice.FinishRegistration(c.GetUint64("userId"), c.GetString("currentJti"), body.CeremonyID, body.Name, request)
	if err != nil {
		writePasskeyServiceError(c, err, false)
		return
	}
	writePasskeySuccess(c, result, component.MessageOperationSuccess)
}

func PasskeyAuthenticationBegin(c *gin.Context) {
	result, err := passkeyservice.BeginAuthentication()
	if err != nil {
		writePasskeyServiceError(c, err, true)
		return
	}
	writePasskeySuccess(c, result, component.MessageOperationSuccess)
}

func PasskeyAuthenticationFinish(c *gin.Context) {
	var body struct {
		CeremonyID string          `json:"ceremonyId"`
		Credential json.RawMessage `json:"credential"`
	}
	if !decodePasskeyJSON(c, &body) {
		return
	}
	if body.CeremonyID == "" || len(body.Credential) == 0 {
		writePasskeyFailure(c, component.MessageAuthLoginInvalidRequest)
		return
	}
	request := c.Request.Clone(c.Request.Context())
	request.Body = io.NopCloser(bytesReader(body.Credential))
	result, err := passkeyservice.FinishAuthentication(body.CeremonyID, request, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		writePasskeyServiceError(c, err, true)
		return
	}
	jwtopt.TokenSetting(c, result.Token)
	writePasskeySuccess(c, gin.H{}, component.MessageAuthLoginSuccess)
}

func PasskeyReauthenticate(c *gin.Context) {
	var body struct {
		EncryptedPassword string `json:"encryptedPassword"`
		TOTPCode          string `json:"totpCode"`
	}
	if !decodePasskeyJSON(c, &body) {
		return
	}
	password, err := logincrypto.DecryptPassword(body.EncryptedPassword)
	if err != nil {
		writePasskeyFailure(c, component.MessageAuthInvalidCredentials)
		return
	}
	proof, err := passkeyservice.Reauthenticate(c.GetUint64("userId"), c.GetString("currentJti"), password, body.TOTPCode)
	if err != nil {
		writePasskeyServiceError(c, err, false)
		return
	}
	writePasskeySuccess(c, gin.H{"proof": proof, "expiresInSeconds": 120}, component.MessageOperationSuccess)
}

func PasskeyCredentialList(c *gin.Context) {
	credentials, err := passkeyservice.ListCredentials(c.GetUint64("userId"))
	if err != nil {
		slog.Error("list passkeys failed", "userId", c.GetUint64("userId"), "error", err)
		writePasskeyFailure(c, component.MessageOperationFailed)
		return
	}
	writePasskeySuccess(c, credentials, component.MessageOperationSuccess)
}

func PasskeyCredentialRename(c *gin.Context) {
	var body struct {
		Proof        string `json:"proof"`
		CredentialID string `json:"credentialId"`
		Name         string `json:"name"`
	}
	if !decodePasskeyJSON(c, &body) {
		return
	}
	if err := passkeyservice.RenameCredential(c.GetUint64("userId"), c.GetString("currentJti"), body.Proof, body.CredentialID, body.Name); err != nil {
		writePasskeyServiceError(c, err, false)
		return
	}
	writePasskeySuccess(c, gin.H{}, component.MessageOperationSuccess)
}

func PasskeyCredentialRevoke(c *gin.Context) {
	var body struct {
		Proof        string `json:"proof"`
		CredentialID string `json:"credentialId"`
	}
	if !decodePasskeyJSON(c, &body) {
		return
	}
	if err := passkeyservice.RevokeCredential(c.GetUint64("userId"), c.GetString("currentJti"), body.Proof, body.CredentialID); err != nil {
		writePasskeyServiceError(c, err, false)
		return
	}
	writePasskeySuccess(c, gin.H{}, component.MessageOperationSuccess)
}

func writePasskeyServiceError(c *gin.Context, err error, login bool) {
	if errors.Is(err, passkeyservice.ErrInvalidRequest) || errors.Is(err, passkeyservice.ErrCounterWarning) {
		if login {
			writePasskeyFailure(c, component.MessageAuthInvalidCredentials)
		} else {
			writePasskeyFailure(c, component.MessageRequestInvalidParams)
		}
		return
	}
	if errors.Is(err, passkeyservice.ErrInvalidProof) {
		writePasskeyFailure(c, component.MessageAuthInvalidCredentials)
		return
	}
	if errors.Is(err, passkeyservice.ErrDisabled) || errors.Is(err, passkeyservice.ErrLimitReached) || errors.Is(err, passkeyservice.ErrNotFound) {
		writePasskeyFailure(c, component.MessageOperationFailed)
		return
	}
	slog.Error("passkey operation failed", "error", err)
	if login {
		writePasskeyFailure(c, component.MessageAuthLoginFailed)
	} else {
		writePasskeyFailure(c, component.MessageOperationFailed)
	}
}

// bytesReader returns a private copy so request parsing cannot alias the input
// buffer retained by the decoder.
func bytesReader(value []byte) io.Reader { return bytes.NewReader(append([]byte(nil), value...)) }
