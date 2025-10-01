package api

type SecretStore interface {
	Get(owner, key string) (string, error)
}
