package config

import "log"

func MustNonEmpty(value, envName string) {
	if value == "" {
		log.Fatalf("missing required env %s", envName)
	}
}

func MustNonEmptyBytes(value []byte, envName string) {
	if len(value) == 0 {
		log.Fatalf("missing required env %s", envName)
	}
}
