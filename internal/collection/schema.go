package collection

import "sort"

// Schema is a simplified Postman Collection v2.1-compatible model:
// https://schema.getpostman.com/json/collection/v2.1.0/collection.json

const schemaURL = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

type Collection struct {
	Info Info   `json:"info"`
	Item []Item `json:"item"`
}

type Info struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type Item struct {
	Name    string   `json:"name"`
	Item    []Item   `json:"item,omitempty"`
	Request *Request `json:"request,omitempty"`
}

// IsFolder reports whether the item is a container rather than a request leaf.
func (i Item) IsFolder() bool {
	return i.Request == nil
}

type Request struct {
	Method string   `json:"method"`
	Header []KeyVal `json:"header,omitempty"`
	Body   *Body    `json:"body,omitempty"`
	URL    URL      `json:"url"`
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Body struct {
	Mode    string       `json:"mode"`
	Raw     string       `json:"raw,omitempty"`
	Options *BodyOptions `json:"options,omitempty"`
}

type BodyOptions struct {
	Raw *RawOptions `json:"raw,omitempty"`
}

type RawOptions struct {
	Language string `json:"language,omitempty"`
}

type URL struct {
	Raw string `json:"raw"`
}

// NewCollection creates an empty, schema-tagged collection ready to be saved.
func NewCollection(name string) *Collection {
	return &Collection{Info: Info{Name: name, Schema: schemaURL}}
}

// NewRequestItem builds a leaf item from the fields the request editor works with.
func NewRequestItem(name, method, rawURL string, headers map[string]string, body, bodyType string) Item {
	req := &Request{Method: method, URL: URL{Raw: rawURL}}
	for k, v := range headers {
		req.Header = append(req.Header, KeyVal{Key: k, Value: v})
	}
	sort.Slice(req.Header, func(i, j int) bool { return req.Header[i].Key < req.Header[j].Key })

	if bodyType != "" && bodyType != "none" {
		lang := "text"
		if bodyType == "JSON" {
			lang = "json"
		}
		req.Body = &Body{Mode: "raw", Raw: body, Options: &BodyOptions{Raw: &RawOptions{Language: lang}}}
	}

	return Item{Name: name, Request: req}
}

// AddItemAt inserts item as a child of the folder located at path.
// A nil/empty path inserts at the collection root.
func (c *Collection) AddItemAt(path []int, item Item) bool {
	items, ok := addAt(c.Item, path, item)
	if !ok {
		return false
	}
	c.Item = items
	return true
}

// RemoveItem deletes the item located at path.
func (c *Collection) RemoveItem(path []int) bool {
	items, ok := removeAt(c.Item, path)
	if !ok {
		return false
	}
	c.Item = items
	return true
}

// RenameItem renames the item located at path.
func (c *Collection) RenameItem(path []int, name string) bool {
	return renameAt(c.Item, path, name)
}

func addAt(items []Item, path []int, item Item) ([]Item, bool) {
	if len(path) == 0 {
		return append(items, item), true
	}
	idx := path[0]
	if idx < 0 || idx >= len(items) {
		return items, false
	}
	children, ok := addAt(items[idx].Item, path[1:], item)
	if !ok {
		return items, false
	}
	items[idx].Item = children
	return items, true
}

func removeAt(items []Item, path []int) ([]Item, bool) {
	if len(path) == 0 {
		return items, false
	}
	idx := path[0]
	if idx < 0 || idx >= len(items) {
		return items, false
	}
	if len(path) == 1 {
		out := append([]Item{}, items[:idx]...)
		out = append(out, items[idx+1:]...)
		return out, true
	}
	children, ok := removeAt(items[idx].Item, path[1:])
	if !ok {
		return items, false
	}
	items[idx].Item = children
	return items, true
}

func renameAt(items []Item, path []int, name string) bool {
	if len(path) == 0 {
		return false
	}
	idx := path[0]
	if idx < 0 || idx >= len(items) {
		return false
	}
	if len(path) == 1 {
		items[idx].Name = name
		return true
	}
	return renameAt(items[idx].Item, path[1:], name)
}
