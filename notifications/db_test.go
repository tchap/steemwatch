package notifications

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/cznic/ql"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

	tctx := ql.NewRWCtx()

	iter := db.C("events").Find(nil).Iter()
	var doc map[string]interface{}
	for iter.Next(&doc) {
		ownerID := doc["ownerId"].(bson.ObjectId)
		kind := doc["kind"].(string)
		for k, v := range doc {
			if v, ok := v.([]interface{}); ok {
				for _, item := range v {
					var (
						rs  []ql.Recordset
						err error
					)
					if kind == "descendant.published" {
						continue
					} else {
						rs, _, err = mem.Run(
							tctx,
							fmt.Sprintf(`
							SELECT count(*)
							FROM %v
							WHERE UserID == $1 AND %v == $2
						`, eventKindToTableName(kind), fieldToRowName(k)),
							ownerID.Hex(), item)
					}
					if err != nil {
						t.Fatal(err)
					}
					row, err := rs[0].FirstRow()
					if err != nil {
						t.Fatal(err)
					}
					count := row[0].(int64)
					if count != 1 {
						t.Errorf("row missing [ownerID = %v, kind = %v, %v = %v]", ownerID, kind, k, item)
					}
				}
			}
		}
	}
	if err := iter.Err(); err != nil {
		t.Fatal(err)
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

func fieldToRowName(fieldName string) string {
	switch fieldName {
	case "authorBlacklist":
		return "BLAuthor"
	case "from":
		return "FromAccount"
	case "to":
		return "ToAccount"
	}

	switch {
	case strings.HasSuffix(fieldName, "es"):
		fieldName = strings.TrimSuffix(fieldName, "es")
	case strings.HasSuffix(fieldName, "s"):
		fieldName = strings.TrimSuffix(fieldName, "s")
	}

	fieldName = strings.Title(fieldName)
	return fieldName
}
