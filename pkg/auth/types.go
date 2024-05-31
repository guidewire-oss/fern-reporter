package auth

type JWKs struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	K   string `json:"k,omitempty"`
}

type Claims struct {
	Scopes map[string]string
}

type Scopes struct {
	Scopes map[string]int
}
