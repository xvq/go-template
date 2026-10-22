package resource

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xvq/go-template/internal/model"

	base "github.com/xvq/go-template/internal/resource"
	"github.com/xvq/go-template/internal/service/settings"
)

func TestAdminPublicFields(t *testing.T) {
	result := AdminItem(model.Admin{ID: 7, Username: "admin", PasswordHash: "secret", AuthVersion: 9})
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
	raw, err := json.Marshal(base.Paginate(base.Page[model.Admin]{Total: 3, Page: 2, PageSize: 20}, AdminItem))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"items":[],"total":3,"page":2,"page_size":20}` {
		t.Fatalf("unexpected page: %s", raw)
	}
	if items := base.Collection(nil, AdminItem); items == nil {
		t.Fatal("empty collection must be an array")
	}
}

func TestSettingsSensitiveValueAndJSONTypes(t *testing.T) {
	entries := []settings.Entry{
		{Key: "secret", Type: "string", Sensitive: true, Value: json.RawMessage(`"private"`)},
		{Key: "enabled", Type: "boolean", Value: json.RawMessage(`false`)},
	}
	out := base.Collection(entries, SettingItem)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `[{"key":"secret","type":"string","sensitive":true},{"key":"enabled","type":"boolean","value":false,"sensitive":false}]` {
		t.Fatalf("unexpected settings: %s", raw)
	}
	out[1].Value[0] = 'x'
	if !reflect.DeepEqual(entries[1].Value, json.RawMessage(`false`)) {
		t.Fatal("resource mutated source")
	}
}

func TestLoginLogUnknownAdmin(t *testing.T) {
	raw, err := json.Marshal(LoginLogItem(model.AdminLoginLog{Username: "unknown"}))
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
