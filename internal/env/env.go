package env

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func Setup() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err == nil {
			return
		}
		log.Println("Notice: unable to parse .env")
	} else if _, err := os.Stat("/opt/elasticbeanstalk/bin/get-config"); err == nil {
		var out bytes.Buffer
		// var errBytes bytes.Buffer

		command := exec.Command("/opt/elasticbeanstalk/bin/get-config", "environment")
		command.Stdout = &out
		// command.Stderr = &errBytes

		if err := command.Run(); err != nil {
			log.Printf("Error running get-config: %v\n", err)
			// log.Printf("Stderr: %s\n", errBytes.String())
			return
		}

		var env map[string]string
		if err := json.Unmarshal(out.Bytes(), &env); err != nil {
			log.Printf("Error parsing environment config: %v\n", err)
			return
		}

		for key, value := range env {
			if err := os.Setenv(key, value); err != nil {
				log.Printf("Error setting environment variable %s: %v\n", key, err)
			}
		}
	}
}

func Get[T any](key string, defaultValue T) T {
	value, ok := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return defaultValue
	}

	// coerse the value to the type of defaultValue
	switch any(defaultValue).(type) {
	case string:
		return any(value).(T)
	case int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Fatalf("unable to convert %s: %s into int64\n", key, value)
		}
		return any(int64(v)).(T)
	case int32:
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			log.Fatalf("unable to convert %s: %s into int32\n", key, value)
		}
		return any(int32(v)).(T)
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("unable to convert %s: %s into int\n", key, value)
		}

		return any(v).(T)
	default:
		return any(value).(T)
	}
}
