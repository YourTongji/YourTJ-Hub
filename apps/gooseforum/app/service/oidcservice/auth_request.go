package oidcservice

import (
	"strconv"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/models/forum/oidcAuthRequests"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// authRequest adapts the persisted auth request row to op.AuthRequest.
type authRequest struct {
	entity *oidcAuthRequests.Entity
}

func (r *authRequest) GetID() string          { return r.entity.RequestId }
func (r *authRequest) GetACR() string         { return "" }
func (r *authRequest) GetAMR() []string       { return nil }
func (r *authRequest) GetAudience() []string  { return []string{r.entity.ClientId} }
func (r *authRequest) GetAuthTime() time.Time { return r.entity.AuthTime }
func (r *authRequest) GetClientID() string    { return r.entity.ClientId }
func (r *authRequest) GetCodeChallenge() *oidc.CodeChallenge {
	if r.entity.CodeChallenge == "" {
		return nil
	}
	return &oidc.CodeChallenge{Challenge: r.entity.CodeChallenge, Method: oidc.CodeChallengeMethodS256}
}
func (r *authRequest) GetNonce() string       { return r.entity.Nonce }
func (r *authRequest) GetRedirectURI() string { return r.entity.RedirectUri }
func (r *authRequest) GetResponseType() oidc.ResponseType {
	return oidc.ResponseType(r.entity.ResponseType)
}
func (r *authRequest) GetResponseMode() oidc.ResponseMode { return oidc.ResponseModeQuery }
func (r *authRequest) GetScopes() []string {
	if r.entity.Scopes == "" {
		return nil
	}
	return strings.Fields(r.entity.Scopes)
}
func (r *authRequest) GetState() string   { return r.entity.State }
func (r *authRequest) GetSubject() string { return strconv.FormatUint(r.entity.Subject, 10) }
func (r *authRequest) Done() bool         { return r.entity.Done }
