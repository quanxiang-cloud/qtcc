package injector

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TLSCertFile string `envconfig:"TLS_CERT_FILE" required:"true"`
	TLSKeyFile  string `envconfig:"TLS_KEY_FILE" required:"true"`
}

// GetConfig returns configuration derived from environment variables.
func GetConfig() (Config, error) {
	// get config from environment variables
	c := Config{}
	err := envconfig.Process("", &c)
	if err != nil {
		return c, err
	}

	return c, nil
}
