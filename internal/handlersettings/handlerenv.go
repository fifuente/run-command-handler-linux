package handlersettings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/run-command-handler-linux/internal/types"
)

// HandlerEnvFileName is the file name of the Handler Environment as placed by the
// Azure Linux Guest Agent.
const HandlerEnvFileName = "HandlerEnvironment.json"

// GetHandlerEnv locates the HandlerEnvironment.json file by assuming it lives
// next to or one level above the extension handler (read: this) executable,
// reads, parses and returns it.
func GetHandlerEnv() (he types.HandlerEnvironment, _ error) {
	dir, err := scriptDir()
	if err != nil {
		return he, fmt.Errorf("vmextension: cannot find base directory of the running process: %v", err)
	}
	paths := []string{
		filepath.Join(dir, HandlerEnvFileName),       // this level (i.e. executable is in [EXT_NAME]/.)
		filepath.Join(dir, "..", HandlerEnvFileName), // one up (i.e. executable is in [EXT_NAME]/bin/.)
	}
	var b []byte
	for _, p := range paths {
		o, err := os.ReadFile(p)
		if err != nil && !os.IsNotExist(err) {
			return he, fmt.Errorf("vmextension: error examining HandlerEnvironment at '%s': %v", p, err)
		} else if err == nil {
			b = o
			break
		}
	}
	if b == nil {
		return he, fmt.Errorf("vmextension: Cannot find HandlerEnvironment at paths: %s", strings.Join(paths, ", "))
	}
	return ParseHandlerEnv(b)
}

// scriptDir returns the absolute path of the running process.
func scriptDir() (string, error) {
	p, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

// ParseHandlerEnv parses the
// /var/lib/waagent/[extension]/HandlerEnvironment.json format.
func ParseHandlerEnv(b []byte) (he types.HandlerEnvironment, _ error) {
	var hf []types.HandlerEnvironment

	if err := json.Unmarshal(b, &hf); err != nil {
		return he, fmt.Errorf("vmextension: failed to parse handler env: %v", err)
	}
	if len(hf) != 1 {
		return he, fmt.Errorf("vmextension: expected 1 config in parsed HandlerEnvironment, found: %v", len(hf))
	}
	return hf[0], nil
}
