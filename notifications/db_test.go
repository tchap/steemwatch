package notifications

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	mgo "gopkg.in/mgo.v2"
)

func TestBuildDB(t *testing.T) {
	port, cleanup := mongo(t)
	defer cleanup()

	mongorestore(t, port)

	session, err := mgo.Dial("localhost:" + port)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	mem, err := buildDB(session.DB("steemwatch"))
	if err != nil {
		t.Fatal(err)
	}
	defer mem.Close()
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

func mongorestore(t *testing.T, port string) {
	t.Helper()

	err := exec.Command("mongorestore", "--port", port, "testdata/steemwatch.dump").Run()
	if err != nil {
		t.Fatal(err)
	}
}
