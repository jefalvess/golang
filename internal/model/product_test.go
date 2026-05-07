package model

import "testing"

func TestProduct_FieldMap(t *testing.T) {
	p := Product{ID: "1", Name: "Teste"}
	fields := p.FieldMap()
	if fields["id"] != "1" || fields["name"] != "Teste" {
		t.Errorf("FieldMap retornou valores incorretos: %v", fields)
	}
}
