package notifications

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestBuildDB(t *testing.T) {
	port, cleanup := mongo(t)
	fmt.Println(port)
	cleanup()
}

func mongo(t *testing.T) (string, func()) {
	t.Helper()

	out, err := exec.Command("docker", "run", "-P", "-d", "--rm", "mongo").Output()
	if err != nil {
		t.Fatal(err)
	}
	containerID := string(bytes.TrimSpace(out))
	cleanup := func() {
		exec.Command("docker", "stop", containerID).Run()
	}

	out, err = exec.Command("docker", "port", containerID, "27017").Output()
	if err != nil {
		cleanup()
		t.Fatal(err)
	}

	parts := strings.Split(string(bytes.TrimSpace(out)), ":")
	if len(parts) != 2 {
		cleanup()
		t.Fatal("unexpected docker output encountered")
	}
	port := parts[1]

	return port, cleanup
}
