package eventsource_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/eventsource"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryLogFactory(t *testing.T) {
	Convey("Given an in-memory log factory", t, func() {
		factory := eventsource.NewInMemoryLogFactory()
		Convey("When creating logs for the same and different channels", func() {
			first, err := factory.Create("orders")
			So(err, ShouldBeNil)
			second, err := factory.Create("orders")
			So(err, ShouldBeNil)
			other, err := factory.Create("fills")
			So(err, ShouldBeNil)
			Convey("Then same-channel logs are reused and different channels are distinct", func() {
				So(first, ShouldEqual, second)
				So(first, ShouldNotEqual, other)
			})
		})
	})
}

func TestLogFactoryFromEnvInMemory(t *testing.T) {
	Convey("Given an in-memory factory alias in the environment", t, func() {
		t.Setenv("LOG_FACTORY", "INMEMORY")
		Convey("When building a log factory from env", func() {
			factory, err := eventsource.LogFactoryFromEnv("LOG", "REDIS")
			So(err, ShouldBeNil)
			Convey("Then an in-memory log factory is returned", func() {
				_, ok := factory.(*eventsource.InMemoryLogFactory)
				So(ok, ShouldBeTrue)
			})
		})
	})
}
