package dotenv

import (
	"fmt"
	"os"
	"strings"
)

var env map[string]string

func Load() {
	data, err := os.ReadFile(".env")
	env = make(map[string]string)

	if err != nil {
		panic(err)
	}

	for line := range strings.SplitSeq(string(data), "\n") {

		key_value := strings.Split(line, "=")

		if len(key_value) != 2 {
			fmt.Println("Invalid line in .env file:", line)
			continue // or log warning if preferred
		}

		env[key_value[0]] = strings.TrimSpace(key_value[1])
	}

}

func Get(key string) string {
	return env[key]
}
