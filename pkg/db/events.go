package db

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type EventsRepo struct {
	db      orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// NewEventsRepo returns new repository
func NewEventsRepo(db orm.DB) EventsRepo {
	return EventsRepo{
		db:      db,
		filters: map[string][]Filter{},
		sort: map[string][]SortField{
			Tables.Event.Name: {{Column: Columns.Event.CreatedAt, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.Event.Name: {TableColumns},
		},
	}
}

// WithTransaction is a function that wraps EventsRepo with pg.Tx transaction.
func (er EventsRepo) WithTransaction(tx *pg.Tx) EventsRepo {
	er.db = tx
	return er
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func (er EventsRepo) WithEnabledOnly() EventsRepo {
	f := make(map[string][]Filter, len(er.filters))
	for i := range er.filters {
		f[i] = make([]Filter, len(er.filters[i]))
		copy(f[i], er.filters[i])
		f[i] = append(f[i], StatusEnabledFilter)
	}
	er.filters = f

	return er
}

/*** Event ***/

// FullEvent returns full joins with all columns
func (er EventsRepo) FullEvent() OpFunc {
	return WithColumns(er.join[Tables.Event.Name]...)
}

// DefaultEventSort returns default sort.
func (er EventsRepo) DefaultEventSort() OpFunc {
	return WithSort(er.sort[Tables.Event.Name]...)
}

// EventByID is a function that returns Event by ID(s) or nil.
func (er EventsRepo) EventByID(ctx context.Context, id int, ops ...OpFunc) (*Event, error) {
	return er.OneEvent(ctx, &EventSearch{ID: &id}, ops...)
}

// OneEvent is a function that returns one Event by filters. It could return pg.ErrMultiRows.
func (er EventsRepo) OneEvent(ctx context.Context, search *EventSearch, ops ...OpFunc) (*Event, error) {
	obj := &Event{}
	err := buildQuery(ctx, er.db, obj, search, er.filters[Tables.Event.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// EventsByFilters returns Event list.
func (er EventsRepo) EventsByFilters(ctx context.Context, search *EventSearch, pager Pager, ops ...OpFunc) (events []Event, err error) {
	err = buildQuery(ctx, er.db, &events, search, er.filters[Tables.Event.Name], pager, ops...).Select()
	return
}

// CountEvents returns count
func (er EventsRepo) CountEvents(ctx context.Context, search *EventSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, er.db, &Event{}, search, er.filters[Tables.Event.Name], PagerOne, ops...).Count()
}

// AddEvent adds Event to DB.
func (er EventsRepo) AddEvent(ctx context.Context, event *Event, ops ...OpFunc) (*Event, error) {
	q := er.db.ModelContext(ctx, event)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Event.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return event, err
}

// UpdateEvent updates Event in DB.
func (er EventsRepo) UpdateEvent(ctx context.Context, event *Event, ops ...OpFunc) (bool, error) {
	q := er.db.ModelContext(ctx, event).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Event.ID, Columns.Event.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteEvent deletes Event from DB.
func (er EventsRepo) DeleteEvent(ctx context.Context, id int) (deleted bool, err error) {
	event := &Event{ID: id}

	res, err := er.db.ModelContext(ctx, event).WherePK().Delete()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}
