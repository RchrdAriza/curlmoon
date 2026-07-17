package collection

import "testing"

func TestNewRequestItem(t *testing.T) {
	item := NewRequestItem("Get todo", "GET", "https://example.com/todo", map[string]string{"X-Api-Key": "abc"}, "", "")

	if item.Request == nil {
		t.Fatal("expected item to have a request")
	}
	if item.Request.Method != "GET" {
		t.Errorf("expected GET, got %s", item.Request.Method)
	}
	if item.Request.URL.Raw != "https://example.com/todo" {
		t.Errorf("expected URL preserved, got %s", item.Request.URL.Raw)
	}
	if len(item.Request.Header) != 1 || item.Request.Header[0].Key != "X-Api-Key" {
		t.Errorf("expected header to be carried over, got %v", item.Request.Header)
	}
	if item.Request.Body != nil {
		t.Errorf("expected no body for bodyType none, got %v", item.Request.Body)
	}
	if item.IsFolder() {
		t.Error("expected request item to not be a folder")
	}
}

func TestNewRequestItem_JSONBody(t *testing.T) {
	item := NewRequestItem("Create thing", "POST", "https://example.com", nil, `{"a":1}`, "JSON")

	if item.Request.Body == nil {
		t.Fatal("expected body to be set")
	}
	if item.Request.Body.Raw != `{"a":1}` {
		t.Errorf("expected raw body preserved, got %s", item.Request.Body.Raw)
	}
	if item.Request.Body.Options.Raw.Language != "json" {
		t.Errorf("expected json language, got %s", item.Request.Body.Options.Raw.Language)
	}
}

func TestCollection_AddRemoveRenameItem(t *testing.T) {
	c := NewCollection("My collection")
	c.AddItemAt(nil, NewRequestItem("Req 1", "GET", "https://a.com", nil, "", ""))
	c.AddItemAt(nil, NewRequestItem("Req 2", "GET", "https://b.com", nil, "", ""))

	if len(c.Item) != 2 {
		t.Fatalf("expected 2 items, got %d", len(c.Item))
	}

	if !c.RenameItem([]int{1}, "Req 2 renamed") {
		t.Fatal("expected rename to succeed")
	}
	if c.Item[1].Name != "Req 2 renamed" {
		t.Errorf("expected renamed item, got %s", c.Item[1].Name)
	}

	if !c.RemoveItem([]int{0}) {
		t.Fatal("expected remove to succeed")
	}
	if len(c.Item) != 1 || c.Item[0].Name != "Req 2 renamed" {
		t.Errorf("expected only the renamed item to remain, got %v", c.Item)
	}

	if c.RemoveItem([]int{5}) {
		t.Error("expected out-of-range remove to fail")
	}
	if c.RenameItem(nil, "x") {
		t.Error("expected empty-path rename to fail")
	}
}

func TestCollection_NestedFolders(t *testing.T) {
	c := NewCollection("Nested")
	c.AddItemAt(nil, Item{Name: "Folder"})
	c.AddItemAt([]int{0}, NewRequestItem("Nested req", "GET", "https://a.com", nil, "", ""))

	if len(c.Item[0].Item) != 1 {
		t.Fatalf("expected nested item under folder, got %v", c.Item[0])
	}

	if !c.RemoveItem([]int{0, 0}) {
		t.Fatal("expected nested remove to succeed")
	}
	if len(c.Item[0].Item) != 0 {
		t.Errorf("expected folder to be empty, got %v", c.Item[0].Item)
	}
}
