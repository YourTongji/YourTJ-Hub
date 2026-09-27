export interface OidcExchangeRequest {
  code: string
  codeVerifier: string
  nonce: string
  redirectUri: string
}

export interface OidcExchangeResult {
  token: string
}

export interface AppleCredentialRequest {
  authorizationCode: string
  identityToken: string
  nonce: string
}
