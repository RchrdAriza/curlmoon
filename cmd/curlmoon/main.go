package main

import (
	"curlmoon/internal/collection"
	"curlmoon/internal/environment"
	"curlmoon/internal/tui"
	"flag"
	"fmt"
	"os"
	"strings"
)

// stringList collects repeated occurrences of a flag, e.g. -c a.json -c b.json.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var collectionFiles, envFiles stringList
	var demo bool

	flag.Var(&collectionFiles, "collection", "path to a Postman v2.1 collection JSON file to import (repeatable)")
	flag.Var(&collectionFiles, "c", "shorthand for -collection")
	flag.Var(&envFiles, "env", "path to a .env file to import as an environment (repeatable)")
	flag.Var(&envFiles, "e", "shorthand for -env")
	flag.BoolVar(&demo, "demo", false, "seed the store with example collections (httpbin.org, JSON Placeholder, GitHub API)")
	flag.Parse()

	store := collection.DefaultStore()
	envStore := environment.NewStore(store.BaseDir)

	var demoCollections []*collection.Collection
	if demo {
		demoCollections = collection.ExampleCollections()
	}

	for _, path := range collectionFiles {
		if _, err := store.Import(path); err != nil {
			fmt.Fprintln(os.Stderr, "curlmoon: importing collection:", err)
			os.Exit(1)
		}
	}

	var lastEnv string
	for _, path := range envFiles {
		env, err := envStore.ImportDotenv(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "curlmoon: importing env:", err)
			os.Exit(1)
		}
		lastEnv = env.Name
	}
	if lastEnv != "" {
		if err := envStore.SetActive(lastEnv); err != nil {
			fmt.Fprintln(os.Stderr, "curlmoon: activating environment:", err)
			os.Exit(1)
		}
	}

	if err := tui.Run(store, demoCollections...); err != nil {
		fmt.Fprintln(os.Stderr, "curlmoon:", err)
		os.Exit(1)
	}
}
