package logger

import "testing"

func TestInitAndSync(t *testing.T) {
	Init()
	if Logger == nil {
		t.Error("Logger não foi inicializado")
	}
	Sync()
}
