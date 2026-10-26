package resource

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xvq/go-template/internal/model"
	"github.com/xvq/go-template/internal/util/query"

	"github.com/xvq/go-template/internal/service/settings"
)

func TestAdminPublicFields(t *testing.T) {
	result := AdminResource.Item(&model.Admin{ID: 7, Username: "admin", PasswordHash: "secret"})
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var admin map[string]json.RawMessage
	if err := json.Unmarshal(raw, &admin); err != nil {
		t.Fatal(err)
	}
	expected := []string{"id", "username", "display_name", "enabled", "created_at", "updated_at"}
	if len(admin) != len(expected) {
		t.Fatalf("unexpected public fields: %s", raw)
	}
	for _, key := range expected {
		if _, ok := admin[key]; !ok {
			t.Errorf("missing field %s", key)
		}
	}
	if string(admin["id"]) != "7" {
		t.Fatalf("unexpected admin response: %s", raw)
	}
}

func TestEmptyPageKeepsTotal(t *testing.T) {
	raw, err := json.Marshal(AdminResource.Page(query.NewPage[model.Admin](nil, 3, 2, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"data":[],"pagination":{"current_page":2,"last_page":1,"per_page":20,"total":3,"count":0}}` {
		t.Fatalf("unexpected page: %s", raw)
	}
	if items := AdminResource.Collection(nil); items == nil {
		t.Fatal("empty collection must be an array")
	}
}

func TestSettingsValuesDefaultsAndJSONTypes(t *testing.T) {
	entries := []settings.Entry{
		{Key: "enabled", Type: "boolean", Value: json.RawMessage(`false`), Default: json.RawMessage(`true`)},
		{Key: "count", Type: "integer", Value: json.RawMessage(`20`), Default: json.RawMessage(`10`)},
		{Key: "extensions", Type: "string_array", Value: json.RawMessage(`[]`), Default: json.RawMessage(`["jpg","png"]`)},
	}

	out := SettingResource.Collection(entries)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `[{"key":"enabled","type":"boolean","value":false,"default":true},{"key":"count","type":"integer","value":20,"default":10},{"key":"extensions","type":"string_array","value":[],"default":["jpg","png"]}]` {
		t.Fatalf("unexpected settings: %s", raw)
	}
	out[0].Value[0] = 'x'
	out[2].Default[0] = 'x'
	if !reflect.DeepEqual(entries[0].Value, json.RawMessage(`false`)) || !reflect.DeepEqual(entries[2].Default, json.RawMessage(`["jpg","png"]`)) {
		t.Fatal("resource mutated source")
	}

}

func TestLoginLogUnknownAdmin(t *testing.T) {
	raw, err := json.Marshal(AdminLoginLogResource.Item(&model.AdminLoginLog{Username: "unknown"}))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if string(doc["admin_id"]) != "null" {
		t.Fatalf("unknown administrator must be null: %s", raw)
	}
}
