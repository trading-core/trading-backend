package entrystore_test

import (
	"context"
	"testing"

	"github.com/kduong/trading-backend/cmd/journal-service/internal/entrystore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEventSourcedStore(t *testing.T) {
	Convey("Given an event sourced entry store", t, func() {
		log := eventsource.NewInMemoryLog("journal_entries")
		var commandHandler entrystore.CommandHandler = entrystore.NewEventSourcedCommandHandler(entrystore.NewEventSourcedCommandHandlerInput{
			Log: log,
		})
		var queryHandler entrystore.QueryHandler = entrystore.NewEventSourcedQueryHandler(entrystore.NewEventSourcedQueryHandlerInput{
			Log: log,
		})
		userCtx := contextx.WithUserID(context.Background(), "user-1")
		otherCtx := contextx.WithUserID(context.Background(), "user-2")

		Convey("When a user upserts an entry for a date", func() {
			err := commandHandler.UpsertEntry(userCtx, &entrystore.Entry{
				Date:            "2026-04-22",
				Notes:           "Good disciplined day",
				Tags:            []string{"breakout", "AAPL"},
				Mood:            "focused",
				DisciplineScore: 8,
				CreatedAt:       "2026-04-22T10:00:00Z",
				UpdatedAt:       "2026-04-22T10:00:00Z",
			})
			So(err, ShouldBeNil)

			Convey("Then the entry is retrievable by its owner", func() {
				entry, err := queryHandler.Get(userCtx, "2026-04-22")
				So(err, ShouldBeNil)
				So(entry.Notes, ShouldEqual, "Good disciplined day")
				So(entry.Mood, ShouldEqual, "focused")
				So(entry.DisciplineScore, ShouldEqual, 8)
				So(entry.Tags, ShouldResemble, []string{"breakout", "AAPL"})
				So(entry.UserID, ShouldEqual, "user-1")
			})

			Convey("And it appears in the owner's list", func() {
				result, err := queryHandler.List(userCtx, entrystore.ListInput{PageSize: 10})
				So(err, ShouldBeNil)
				So(result.TotalCount, ShouldEqual, 1)
				So(result.Entries[0].Date, ShouldEqual, "2026-04-22")
			})

			Convey("And a different user cannot see it", func() {
				entry, err := queryHandler.Get(otherCtx, "2026-04-22")
				So(err, ShouldEqual, entrystore.ErrEntryNotFound)
				So(entry, ShouldBeNil)

				result, err := queryHandler.List(otherCtx, entrystore.ListInput{PageSize: 10})
				So(err, ShouldBeNil)
				So(result.TotalCount, ShouldEqual, 0)
			})

			Convey("When the same user upserts the same date again", func() {
				err := commandHandler.UpsertEntry(userCtx, &entrystore.Entry{
					Date:            "2026-04-22",
					Notes:           "Updated notes",
					DisciplineScore: 9,
					CreatedAt:       "2026-04-22T11:00:00Z",
					UpdatedAt:       "2026-04-22T11:00:00Z",
				})
				So(err, ShouldBeNil)

				Convey("Then the entry reflects the newest content", func() {
					entry, err := queryHandler.Get(userCtx, "2026-04-22")
					So(err, ShouldBeNil)
					So(entry.Notes, ShouldEqual, "Updated notes")
					So(entry.DisciplineScore, ShouldEqual, 9)
				})

				Convey("And CreatedAt is preserved from the first upsert", func() {
					entry, err := queryHandler.Get(userCtx, "2026-04-22")
					So(err, ShouldBeNil)
					So(entry.CreatedAt, ShouldEqual, "2026-04-22T10:00:00Z")
				})

				Convey("And there is still only one entry for that date", func() {
					result, err := queryHandler.List(userCtx, entrystore.ListInput{PageSize: 10})
					So(err, ShouldBeNil)
					So(result.TotalCount, ShouldEqual, 1)
				})
			})

			Convey("When deleting the entry", func() {
				err := commandHandler.DeleteEntry(userCtx, entrystore.DeleteEntryInput{
					Date:      "2026-04-22",
					UpdatedAt: "2026-04-22T12:00:00Z",
				})
				So(err, ShouldBeNil)

				Convey("Then it is no longer retrievable", func() {
					_, err := queryHandler.Get(userCtx, "2026-04-22")
					So(err, ShouldEqual, entrystore.ErrEntryNotFound)
				})

				Convey("And list returns empty", func() {
					result, err := queryHandler.List(userCtx, entrystore.ListInput{PageSize: 10})
					So(err, ShouldBeNil)
					So(result.TotalCount, ShouldEqual, 0)
				})
			})

			Convey("When a different user tries to delete it", func() {
				err := commandHandler.DeleteEntry(otherCtx, entrystore.DeleteEntryInput{
					Date:      "2026-04-22",
					UpdatedAt: "2026-04-22T12:00:00Z",
				})

				Convey("Then it fails with ErrEntryNotFound (owner's entry is invisible)", func() {
					So(err, ShouldEqual, entrystore.ErrEntryNotFound)
				})
			})
		})

		Convey("When multiple entries are upserted across dates", func() {
			dates := []string{"2026-04-20", "2026-04-21", "2026-04-22", "2026-04-23"}
			for _, date := range dates {
				err := commandHandler.UpsertEntry(userCtx, &entrystore.Entry{
					Date:      date,
					Notes:     "entry " + date,
					CreatedAt: date + "T10:00:00Z",
					UpdatedAt: date + "T10:00:00Z",
				})
				So(err, ShouldBeNil)
			}

			Convey("Then List filters by date range", func() {
				result, err := queryHandler.List(userCtx, entrystore.ListInput{
					From:     "2026-04-21",
					To:       "2026-04-22",
					PageSize: 10,
				})
				So(err, ShouldBeNil)
				So(result.TotalCount, ShouldEqual, 2)
			})

			Convey("And results are ordered by date descending", func() {
				result, err := queryHandler.List(userCtx, entrystore.ListInput{PageSize: 10})
				So(err, ShouldBeNil)
				So(result.Entries[0].Date, ShouldEqual, "2026-04-23")
				So(result.Entries[3].Date, ShouldEqual, "2026-04-20")
			})

			Convey("And pagination works", func() {
				result, err := queryHandler.List(userCtx, entrystore.ListInput{PageSize: 2, Page: 0})
				So(err, ShouldBeNil)
				So(len(result.Entries), ShouldEqual, 2)
				So(result.TotalPages, ShouldEqual, 2)
			})
		})

		Convey("When deleting a nonexistent entry", func() {
			err := commandHandler.DeleteEntry(userCtx, entrystore.DeleteEntryInput{
				Date:      "2026-04-22",
				UpdatedAt: "2026-04-22T12:00:00Z",
			})
			Convey("Then it fails with ErrEntryNotFound", func() {
				So(err, ShouldEqual, entrystore.ErrEntryNotFound)
			})
		})
	})
}
