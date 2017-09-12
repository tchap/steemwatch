package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const collectionName = "events"

type Doc struct {
	ID              bson.ObjectId `bson:"_id"`
	Accounts        []string      `bson:"accounts"`
	Witnesses       []string      `bson:"witnesses"`
	From            []string      `bson:"from"`
	To              []string      `bson:"to"`
	Users           []string      `bson:"users"`
	AuthorBlacklist []string      `bson:"authorBlacklist"`
	Tags            []string      `bson:"tags"`
	Authors         []string      `bson:"authors"`
	Voters          []string      `bson:"voters"`
	ParentAuthors   []string      `bson:"parentAuthors"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %# v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Flags.
	dry := flag.Bool("dry", false, "dry run")
	flag.Parse()

	// Get MongoDB URL.
	args := flag.Args()
	if len(args) != 1 {
		return errors.New("Usage: mongo_fix_lists <mongo-url>")
	}
	mongoURL := args[0]

	// Connect to MongoDB.
	conn, err := mgo.Dial(mongoURL)
	if err != nil {
		return errors.Wrapf(err, "failed to dial MongoDB using URL %v", mongoURL)
	}
	defer conn.Close()

	c := conn.DB("").C(collectionName)

	separators := regexp.MustCompile(`[\t\n\v\f\r ,]+`)

	var res Doc
	iter := c.Find(nil).Iter()
	for iter.Next(&res) {
		update := bson.M{}

		check := func(list []string, key string) {
			var (
				dirty   bool
				newList []string
			)
			for _, v := range list {
				// Trim leading and trailing spaces.
				v1 := strings.TrimSpace(v)
				if v1 != v {
					dirty = true
					v = v1
				}

				// Split by any number of spaces.
				items := separators.Split(v, -1)
				if len(items) != 1 {
					dirty = true
				}
			ItemLoop:
				for _, item := range items {
					it := strings.Trim(item, "@")
					if it != item {
						dirty = true
					}
					// Handle somebody inserting strings looking like
					// - userA   - userB   - userC
					if it == "-" {
						continue ItemLoop
					}
					for _, ei := range newList {
						if ei == it {
							continue ItemLoop
						}
					}
					newList = append(newList, it)
				}
			}
			if dirty {
				update[key] = newList
				fmt.Printf("FIX %v: %# v -> %# v\n", key, pretty.Formatter(list), pretty.Formatter(newList))
			}
		}

		check(res.Accounts, "accounts")
		check(res.AuthorBlacklist, "authorBlacklist")
		check(res.Authors, "authors")
		check(res.From, "from")
		check(res.ParentAuthors, "parentAuthors")
		check(res.Tags, "tags")
		check(res.To, "to")
		check(res.Users, "users")
		check(res.Voters, "voters")
		check(res.Witnesses, "witnesses")

		if len(update) == 0 {
			continue
		}

		fmt.Printf("UPDATE %v %# v\n", res.ID, update)

		if *dry {
			continue
		}

		if err := c.UpdateId(res.ID, bson.M{"$set": update}); err != nil {
			return errors.Wrapf(err, "failed to update a doc; doc = %# v, update = %# v",
				pretty.Formatter(res), pretty.Formatter(update))
		}
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "failed to process all documents")
	}

	fmt.Println("Fixed successfully.")
	return nil
}
