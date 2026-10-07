package feed

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestDeploymentRawTableListMatchesOwner(t *testing.T) {
	data, err := os.ReadFile("../../../../../../deploy/scripts/feed-raw-tables.json")
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	if err = json.Unmarshal(data, &tables); err != nil {
		t.Fatal(err)
	}
	want := append([]string{}, RawTables...)
	slices.Sort(want)
	slices.Sort(tables)
	if !reflect.DeepEqual(want, tables) {
		t.Fatalf("backup raw-table list differs: %v != %v", tables, want)
	}
}
