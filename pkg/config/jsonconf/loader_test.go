package jsonconf

import "testing"

func TestJSONLoader(t *testing.T) {
	loader := New("config_test.json")
	conf, err := loader.Load()
	if err != nil {
		t.Errorf("Load error: %s\n", err)
		return
	}
	if !conf.Logger.Color {
		t.Errorf("Config data error\n")
		return
	}
}
