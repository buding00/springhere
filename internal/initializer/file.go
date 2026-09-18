package initializer

import (
	"os"
	"strings"
)

func patchFile(path, from, to string) error {
	if from == "" || from == to {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	body := string(data)
	if !strings.Contains(body, from) {
		return nil
	}
	return os.WriteFile(path, []byte(strings.ReplaceAll(body, from, to)), 0o644)
}
