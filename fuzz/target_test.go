package fuzz

import (
	"strings"
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	target := validTarget()
	if err := target.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	target := validTarget()
	target.Name = ""
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention name: %v", err)
	}
}

func TestValidate_MissingConnector(t *testing.T) {
	target := validTarget()
	target.Connector = ""
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
	if !strings.Contains(err.Error(), "connector") {
		t.Errorf("error should mention connector: %v", err)
	}
}

func TestValidate_MissingAction(t *testing.T) {
	target := validTarget()
	target.Action = ""
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error for missing action")
	}
	if !strings.Contains(err.Error(), "action") {
		t.Errorf("error should mention action: %v", err)
	}
}

func TestValidate_NoFuzzFields(t *testing.T) {
	target := validTarget()
	target.FuzzFields = nil
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error for no fuzz fields")
	}
	if !strings.Contains(err.Error(), "fuzz field") {
		t.Errorf("error should mention fuzz field: %v", err)
	}
}

func TestValidate_EmptyFuzzFields(t *testing.T) {
	target := validTarget()
	target.FuzzFields = []string{}
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error for empty fuzz fields")
	}
}

func TestValidate_FuzzFieldNotInParams(t *testing.T) {
	target := validTarget()
	target.FuzzFields = []string{"nonexistent"}
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error for fuzz field not in parameters")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the missing field: %v", err)
	}
}

func TestValidate_MultipleFuzzFields(t *testing.T) {
	target := validTarget()
	target.FuzzFields = []string{"field1", "field2"}
	if err := target.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_PartialBadFuzzFields(t *testing.T) {
	target := validTarget()
	target.FuzzFields = []string{"field1", "missing"}
	err := target.Validate()
	if err == nil {
		t.Fatal("expected error when one fuzz field missing")
	}
}
