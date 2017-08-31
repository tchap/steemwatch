package notifications

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/cznic/ql"

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

	db := session.DB("steemwatch")
	mem, err := buildDB(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mem.Close()

	iter := db.C("events").Find(nil).Iter()
	var doc map[string]interface{}
	numRowsByEventKind := make(map[string]int)
	for iter.Next(&doc) {
		kind := doc["kind"].(string)
		for _, v := range doc {
			if v, ok := v.([]interface{}); ok {
				num := numRowsByEventKind[kind]
				num += len(v)
				numRowsByEventKind[kind] = num
			}
		}
	}
	if err := iter.Err(); err != nil {
		t.Fatal(err)
	}

	// Count rows in QL.
	tctx := ql.NewRWCtx()
	for k, v := range numRowsByEventKind {
		table := eventKindToTableName(k)
		rs, _, err := mem.Run(tctx, "SELECT count(*) FROM "+table)
		if err != nil {
			t.Fatal(err)
		}
		row, err := rs[0].FirstRow()
		if err != nil {
			t.Fatal(err)
		}
		count := row[0].(int64)
		if count != int64(v) {
			t.Errorf("row count mismatch for %v: expected %v, got %v", k, v, count)
		}
	}
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

func eventKindToTableName(kind string) string {
	kind = strings.Replace(kind, ".", " ", -1)
	kind = strings.Replace(kind, "_", " ", -1)
	kind = strings.Title(kind)
	kind = strings.Replace(kind, " ", "", -1)
	return kind
}
