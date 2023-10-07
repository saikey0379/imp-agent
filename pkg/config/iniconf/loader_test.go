package iniconf

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestINILoader(t *testing.T) {
	loader := New("config_test.ini")
	conf, err := loader.Load()
	if err != nil {
		t.Errorf("Load error: %s\n", err)
		return
	}

	Convey("Logger config 校验", t, func() {
		So(conf.Logger.Color, ShouldEqual, true)
	})
}

func TestINILoaderContent(t *testing.T) {
	var iniContent = `[Logger]
color = true`

	loader := NewContent([]byte(iniContent))
	conf, err := loader.Load()
	if err != nil {
		t.Errorf("Load error: %s\n", err)
		return
	}

	Convey("Logger config 校验", t, func() {
		So(conf.Logger.Color, ShouldEqual, true)
	})
}
