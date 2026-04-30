package fleet

import "testing"

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	err := r.Register("static", func() Provider { return &mockProvider{name: "static"} })
	if err != nil {
		t.Fatal(err)
	}
	prov, err := r.Get("static")
	if err != nil {
		t.Fatal(err)
	}
	if prov.Name() != "static" {
		t.Errorf("name = %q", prov.Name())
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	r := NewRegistry()
	_ = r.Register("x", func() Provider { return &mockProvider{name: "x"} })
	err := r.Register("x", func() Provider { return &mockProvider{name: "x"} })
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryNames(t *testing.T) {
	r := NewRegistry()
	_ = r.Register("beta", func() Provider { return &mockProvider{name: "beta"} })
	_ = r.Register("alpha", func() Provider { return &mockProvider{name: "alpha"} })
	names := r.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("names = %v", names)
	}
}

func TestRegistryInstanceIndependence(t *testing.T) {
	r := NewRegistry()
	_ = r.Register("test", func() Provider { return &mockProvider{name: "test"} })
	p1, _ := r.Get("test")
	p2, _ := r.Get("test")
	if p1 == p2 {
		t.Error("expected distinct instances")
	}
}
