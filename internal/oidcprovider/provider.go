package oidcprovider

import (
	"context"
	"crypto/rsa"
	"net/url"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/token/jwt"
)

func allowHTTPRedirectURI(_ context.Context, u *url.URL) bool {
	return u != nil && (u.Scheme == "http" || u.Scheme == "https")
}

// NewOAuth2Provider builds a fosite OAuth2Provider with OpenID Connect,
// PKCE, token revocation and introspection enabled.
func NewOAuth2Provider(store fosite.Storage, secret []byte, privateKey *rsa.PrivateKey, issuer string) fosite.OAuth2Provider {
	config := &fosite.Config{
		AccessTokenLifespan:         time.Hour,
		AuthorizeCodeLifespan:       10 * time.Minute,
		IDTokenLifespan:             time.Hour,
		RefreshTokenLifespan:        720 * time.Hour,
		IDTokenIssuer:               issuer,
		GlobalSecret:                secret,
		ClientSecretsHasher:         PlainSecretHasher{},
		ScopeStrategy:               fosite.HierarchicScopeStrategy,
		AudienceMatchingStrategy:    fosite.DefaultAudienceMatchingStrategy,
		RedirectSecureChecker:       allowHTTPRedirectURI,
		EnforcePKCEForPublicClients: true,
	}

	getPrivateKey := func(_ context.Context) (interface{}, error) {
		return privateKey, nil
	}

	strategy := &compose.CommonStrategy{
		CoreStrategy:               compose.NewOAuth2HMACStrategy(config),
		OpenIDConnectTokenStrategy: compose.NewOpenIDConnectStrategy(getPrivateKey, config),
		Signer:                     &jwt.DefaultSigner{GetPrivateKey: getPrivateKey},
	}

	return compose.Compose(
		config,
		store,
		strategy,
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OAuth2ClientCredentialsGrantFactory,
		compose.OAuth2RefreshTokenGrantFactory,
		compose.OpenIDConnectExplicitFactory,
		compose.OpenIDConnectRefreshFactory,
		compose.OAuth2PKCEFactory,
		compose.OAuth2TokenRevocationFactory,
		compose.OAuth2TokenIntrospectionFactory,
	)
}
