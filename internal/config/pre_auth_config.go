package config

type PreAuthConfig struct {
	Enabled           bool             `yaml:"enabled" json:"enabled"`
	PassThroughHeader string           `yaml:"pass_through_header" json:"pass_through_header"`
	PassThroughValue  string           `yaml:"pass_through_value" json:"pass_through_value"`
	OAuth2            OAuth2AuthConfig `yaml:"oauth2" json:"oauth2"`
	Basic             BasicAuthConfig  `yaml:"basic" json:"basic"`
	JWT               JWTAuthConfig    `yaml:"jwt" json:"jwt"`
}

type OAuth2AuthConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	TokenHeader      string   `yaml:"token_header" json:"token_header"`
	IntrospectionURL string   `yaml:"introspection_url" json:"introspection_url"`
	ClientID         string   `yaml:"client_id" json:"client_id"`
	ClientSecret     string   `yaml:"client_secret" json:"client_secret"`
	TimeoutSeconds   int      `yaml:"timeout_seconds" json:"timeout_seconds"`
	RequiredScopes   []string `yaml:"required_scopes" json:"required_scopes"`
	AccessTokens     []string `yaml:"access_tokens" json:"access_tokens"`
}

type BasicAuthConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Users   []BasicUserConfig `yaml:"users" json:"users"`
}

type BasicUserConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

type JWTAuthConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	TokenHeader    string   `yaml:"token_header" json:"token_header"`
	Secret         string   `yaml:"secret" json:"secret"`
	Issuer         string   `yaml:"issuer" json:"issuer"`
	Audience       string   `yaml:"audience" json:"audience"`
	AllowedMethods []string `yaml:"allowed_methods" json:"allowed_methods"`
}
