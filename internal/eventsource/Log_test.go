package eventsource_test

import (
	"testing"
	"time"

	"github.com/kduong/trading-backend/internal/eventsource"
	. "github.com/smartystreets/goconvey/convey"
)

func testLog(t *testing.T, implementation string, setup func(channel string) (eventsource.Log, func(), error)) {
	Convey("Given a "+implementation+" log", t, func() {
		log, cleanup, err := setup("orders")
		So(err, ShouldBeNil)
		if cleanup != nil {
			defer cleanup()
		}
		Convey("When events are appended and read from cursor 0", func() {
			event1, err := log.Append([]byte("first"))
			So(err, ShouldBeNil)
			event2, err := log.Append([]byte("second"))
			So(err, ShouldBeNil)
			events, cursor, err := log.Read(0, 10, 0)
			So(err, ShouldBeNil)
			Convey("Then append should produce sequential events", func() {
				So(event1.LogID, ShouldEqual, "orders")
				So(event1.Sequence, ShouldEqual, int64(1))
				So(event1.Data, ShouldResemble, []byte("first"))
				So(event2.Sequence, ShouldEqual, int64(2))
				So(event2.Data, ShouldResemble, []byte("second"))
			})
			Convey("Then read should return events after the cursor", func() {
				So(cursor, ShouldEqual, int64(2))
				So(len(events), ShouldEqual, 2)
				So(events[0].Sequence, ShouldEqual, int64(1))
				So(events[1].Sequence, ShouldEqual, int64(2))
				next, nextCursor, err := log.Read(1, 10, 0)
				So(err, ShouldBeNil)
				So(nextCursor, ShouldEqual, int64(2))
				So(len(next), ShouldEqual, 1)
				So(next[0].Sequence, ShouldEqual, int64(2))
			})
		})
		Convey("When appending three events", func() {
			_, err := log.Append([]byte("a"))
			So(err, ShouldBeNil)
			_, err = log.Append([]byte("b"))
			So(err, ShouldBeNil)
			_, err = log.Append([]byte("c"))
			So(err, ShouldBeNil)
			Convey("And reading with limit 2", func() {
				events, cursor, err := log.Read(0, 2, 0)
				So(err, ShouldBeNil)
				Convey("Then read should respect the limit and update cursor", func() {
					So(len(events), ShouldEqual, 2)
					So(cursor, ShouldEqual, int64(2))
				})
			})
			Convey("And reading from an up-to-date cursor", func() {
				events, cursor, err := log.Read(3, 10, 0)
				So(err, ShouldBeNil)
				Convey("Then read should return no events and keep cursor unchanged", func() {
					So(len(events), ShouldEqual, 0)
					So(cursor, ShouldEqual, int64(3))
				})
			})
		})
		Convey("When appending a mutable payload", func() {
			payload := []byte("copy-me")
			_, err := log.Append(payload)
			So(err, ShouldBeNil)
			Convey("And the original payload is mutated after append", func() {
				payload[0] = 'X'
				events, _, err := log.Read(0, 10, 0)
				So(err, ShouldBeNil)
				Convey("Then stored event data should remain unchanged", func() {
					So(events[0].Data, ShouldResemble, []byte("copy-me"))
				})
			})
			Convey("And read results are mutated by the caller", func() {
				events, _, err := log.Read(0, 10, 0)
				So(err, ShouldBeNil)
				events[0].Data[0] = 'Y'
				again, _, err := log.Read(0, 10, 0)
				So(err, ShouldBeNil)
				Convey("Then subsequent reads should still return original data", func() {
					So(again[0].Data, ShouldResemble, []byte("copy-me"))
				})
			})
		})
		Convey("Given a blocked reader", func() {
			resultCh := make(chan []*eventsource.Event, 1)
			errCh := make(chan error, 1)
			cursorCh := make(chan int64, 1)
			go func() {
				events, cursor, err := log.Read(0, 10, 500)
				errCh <- err
				cursorCh <- cursor
				resultCh <- events
			}()
			Convey("When a new event is appended", func() {
				time.Sleep(25 * time.Millisecond)
				_, err := log.Append([]byte("wakeup"))
				So(err, ShouldBeNil)
				Convey("Then the blocked read should return that event", func() {
					select {
					case err := <-errCh:
						So(err, ShouldBeNil)
						So(<-cursorCh, ShouldEqual, int64(1))
						events := <-resultCh
						So(len(events), ShouldEqual, 1)
						So(events[0].Data, ShouldResemble, []byte("wakeup"))
					case <-time.After(1 * time.Second):
						So("timed out waiting for blocked read", ShouldEqual, "")
					}
				})
			})
		})
	})
}
