package config

// Options contains all settings read from the configuration file
type Options struct {
	Redis struct {
		URI      string
		Password string
	}
	Mutex struct {
		Prefix     string
		DefaultTTL int
	}
}
