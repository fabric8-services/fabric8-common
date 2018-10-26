package configuration

import (
	"fmt"
	"os"
	"strings"

	errs "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// String returns the current configuration as a string
func (c *Registry) String() string {
	allSettings := c.v.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"settings": allSettings,
			"err":      err,
		}).Panicln("Failed to marshall config to string")
	}
	return fmt.Sprintf("%s\n", y)
}

const (
	varLogLevel             = "log.level"
	varDeveloperModeEnabled = "developer.mode.enabled"
	varAuthURL              = "auth.url"
	varEnvironment          = "environment"
	varLogJSON              = "log.json"
	varHTTPAddress          = "http.address"
	varMetricsHTTPAddress   = "metrics.http.address"
	varDiagnoseHTTPAddress  = "diagnose.http.address"

	defaultLogLevel = "info"

	// DevModeRsaPrivateKey for signing JWT Tokens in Dev Mode
	// ssh-keygen -f alm_rsa
	DevModeRsaPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA40yB6SNoU4SpWxTfG5ilu+BlLYikRyyEcJIGg//w/GyqtjvT
/CVo92DRTh/DlrgwjSitmZrhauBnrCOoUBMin0/TXeSo3w2M5tEiiIFPbTDRf2jM
fbSGEOke9O0USCCR+bM2TncrgZR74qlSwq38VCND4zHc89rAzqJ2LVM2aXkuBbO7
TcgLNyooBrpOK9khVHAD64cyODAdJY4esUjcLdlcB7TMDGOgxGGn2RARU7+TUf32
gZZbTMikbuPM5gXuzGlo/22ECbQSKuZpbGwgPIAZ5NN9QA4D1NRz9+KDoiXZ6deZ
TTVCrZykJJ6RyLNfRh+XS+6G5nvcqAmfBpyOWwIDAQABAoIBAE5pBie23zZwfTu+
Z3jNn96/+idLC+DBqq5qsXS3xhpOIlXbLbW98gfkjk+1BXPo9la7wadLlpeX8iuf
4WA+OaNblj69ssO/mOvHGXKdqRixzpN1Q5XZwKX0xYkYf/ahxbmt6P4IfimlX1dB
shsWigU8ZR7rBJ3ayMh/ouTf39ViIbXsHYpEubmACcLaOlXbEuZNr7ofkFQKl/mh
XLWUeOoM97xY6Agw/gv60GIcxIC5OAg7iNqS+XNzhba7f2nf2YqodbN9H1BmEJsf
RRaTTWlZAiQXC8lpZOKwP7DiMLOT78lfmlYtquEBhwRbXazfzsdf67Mr4Kdl2Cej
Jy0EGwECgYEA/DZWB0Lb0tPdT1FmORNrBfGg3PjhX9FOilhbtUgX3nNKp8Zsi3yO
yN6hf0/98qIGlmAQi5C92cXpdhqTiVAGktWD+q0a1W99udIjinS1tFrKgNtOyBWN
uwDBZyhw8RrwpQinMe7B966SVDaphvvOWlB1TadMDh5kReJCYpvRCrMCgYEA5rZj
djCU2UqMw6jIP07nCFjWgxPPjg7jP8aRo07oW2mv1sEA0doCyoZaMrdNeGd3fB0B
sm+IvlQtWD7r0tWZI1GkYpdRkDFurdkIzVPV5pMwH4ByOq/Jf5ZqtjIpoMaRBirA
whJyjmiGU3yDyPDLtEFpNgqM3mIyxS6M6UGKYbkCgYEAg6w+d6YBK+1uQiXGD5BC
tKS0jgjlaOfWcEW3A0qzI3Dfjf3610vdI6OPfu8dLppGhCV9HdAgPdykiQNQ+UQt
WmVcdPgA5WNCqUu7QGK0Joer52AXnkAacYHwdtHXPRkKf66n01rKK2wZexvan91A
m0gcJcFs5IYbZZy9ecvNdB8CgYEAo4JZ5Vay93j1YGnLWcrixDCp/wXYUJbOidGC
QBpZZQf3Hh11JkT7O2uSm2T727yAmw63uC2B3VotNOCLI8ZMHRLsjQ8vOCFAjqdF
rLeg3iQss/bFfkA9b1Y8VNoiVJbGC3fbWu/WDoWXxa12fL/jruG43hsGEUnJL6Q5
K8tOdskCgYABpoHFRxsvJ5Sp9CUS3BBTicVSkpAjoX2O3+cS9XL8IsIqZEMW7VKb
16/H2BRvI0uUq12t+UCc0P0SyrWRGxwGR5zSYHVDOot5EDHqE8aYSbX4jiXtAAiu
qCn3Rug8QWyBjjxnU3CxPRiLSmEllQAAVlzfRWn6kL4RKSyruUhZaA==
-----END RSA PRIVATE KEY-----`
)

// Registry encapsulates the Viper configuration registry which stores the
// configuration data in-memory.
type Registry struct {
	v *viper.Viper
}

// New creates a configuration reader object using a configurable configuration
// file path.
func New(configFilePath string) (*Registry, error) {
	c := Registry{
		v: viper.New(),
	}
	c.v.SetEnvPrefix("F8")
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	if configFilePath != "" {
		c.v.SetConfigType("yaml")
		c.v.SetConfigFile(configFilePath)
		err := c.v.ReadInConfig() // Find and read the config file
		if err != nil {           // Handle errors reading the config file
			return nil, errs.Errorf("Fatal error config file: %s \n", err)
		}
	}
	return &c, nil
}

func getConfigFilePath() string {
	// This was either passed as a env var Or, set inside main.go from --config
	envConfigPath, ok := os.LookupEnv("F8_CONFIG_FILE_PATH")
	if !ok {
		return ""
	}
	return envConfigPath
}

// Get is a wrapper over New() which reads configuration file path from the
// environment variable.
func Get() (*Registry, error) {
	cd, err := New(getConfigFilePath())
	return cd, err
}

func (c *Registry) setConfigDefaults() {
	c.v.SetDefault(varLogLevel, defaultLogLevel)
	c.v.SetDefault(varDeveloperModeEnabled, false)
	c.v.SetDefault(varHTTPAddress, "0.0.0.0:8080")
	c.v.SetDefault(varMetricsHTTPAddress, "0.0.0.0:8080")
}

// GetLogLevel returns the loggging level (as set via config file or environment variable)
func (c *Registry) GetLogLevel() string {
	return c.v.GetString(varLogLevel)
}

// DeveloperModeEnabled returns `true` if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *Registry) DeveloperModeEnabled() bool {
	return c.v.GetBool(varDeveloperModeEnabled)
}

// GetAuthServiceURL returns the Auth Service URL
func (c *Registry) GetAuthServiceURL() string {
	return c.v.GetString(varAuthURL)
}

// GetEnvironment returns the current environment application is deployed in
// like 'production', 'prod-preview', 'local', etc as the value of environment variable
// `F8_ENVIRONMENT` is set.
func (c *Registry) GetEnvironment() string {
	if c.v.IsSet(varEnvironment) {
		return c.v.GetString(varEnvironment)
	}
	return "local"
}

// IsLogJSON returns if we should log json format (as set via config file or environment variable)
func (c *Registry) IsLogJSON() bool {
	if c.v.IsSet(varLogJSON) {
		return c.v.GetBool(varLogJSON)
	}
	if c.DeveloperModeEnabled() {
		return false
	}
	return true
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the wit server binds to (e.g. "0.0.0.0:8080")
func (c *Registry) GetHTTPAddress() string {
	return c.v.GetString(varHTTPAddress)
}

// GetMetricsHTTPAddress returns the address the /metrics endpoing will be mounted.
// By default GetMetricsHTTPAddress is the same as GetHTTPAddress
func (c *Registry) GetMetricsHTTPAddress() string {
	return c.v.GetString(varMetricsHTTPAddress)
}

// GetDiagnoseHTTPAddress returns the address of where to start the gops handler.
// By default GetDiagnoseHTTPAddress is 127.0.0.1:0 in devMode, but turned off in prod mode
// unless explicitly configured
func (c *Registry) GetDiagnoseHTTPAddress() string {
	if c.v.IsSet(varDiagnoseHTTPAddress) {
		return c.v.GetString(varDiagnoseHTTPAddress)
	} else if c.DeveloperModeEnabled() {
		return "127.0.0.1:0"
	}
	return ""
}
