package environment

import "testing"

func TestResolve(t *testing.T) {
	vars := map[string]string{"host": "example.com", "id": "42"}
	got := Resolve("https://{{host}}/users/{{id}}", vars)
	want := "https://example.com/users/42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolve_UnknownTokenLeftUntouched(t *testing.T) {
	got := Resolve("{{missing}}", map[string]string{"other": "x"})
	if got != "{{missing}}" {
		t.Errorf("expected unresolved token to be left as-is, got %q", got)
	}
}

func TestResolve_NoVars(t *testing.T) {
	got := Resolve("plain text", nil)
	if got != "plain text" {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestEnvironment_Vars_SkipsDisabled(t *testing.T) {
	e := &Environment{Values: []KeyVal{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: false},
	}}
	vars := e.Vars()
	if vars["a"] != "1" {
		t.Errorf("expected a=1, got %v", vars)
	}
	if _, ok := vars["b"]; ok {
		t.Error("expected disabled var b to be excluded")
	}
}

func TestStore_CreateLoadDelete(t *testing.T) {
	s := NewStore(t.TempDir())

	env, err := s.Create("local")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	env.Values = []KeyVal{{Key: "host", Value: "localhost", Enabled: true}}
	if err := s.Save(env); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	envs, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(envs) != 1 || envs[0].Name != "local" {
		t.Fatalf("expected 1 environment named local, got %+v", envs)
	}
	if envs[0].Vars()["host"] != "localhost" {
		t.Errorf("expected host=localhost, got %v", envs[0].Vars())
	}

	if _, err := s.Create("local"); err == nil {
		t.Error("expected error creating duplicate environment")
	}

	if err := s.Delete("local"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	envs, _ = s.LoadAll()
	if len(envs) != 0 {
		t.Errorf("expected 0 environments after delete, got %d", len(envs))
	}
}

func TestStore_Rename(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("dev")

	if err := s.Rename("dev", "staging"); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	envs, _ := s.LoadAll()
	if len(envs) != 1 || envs[0].Name != "staging" {
		t.Fatalf("expected renamed environment, got %+v", envs)
	}
}

func TestStore_ActiveEnvironment(t *testing.T) {
	s := NewStore(t.TempDir())

	name, err := s.LoadActive()
	if err != nil || name != "" {
		t.Fatalf("expected no active environment initially, got %q, err %v", name, err)
	}

	if err := s.SetActive("prod"); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}
	name, err = s.LoadActive()
	if err != nil || name != "prod" {
		t.Fatalf("expected active=prod, got %q, err %v", name, err)
	}
}
