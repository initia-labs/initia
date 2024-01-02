package types

import (
	"context"

	"cosmossdk.io/collections"
	dbm "github.com/cosmos/cosmos-db"
)

type VMStore struct {
	ctx   context.Context
	store collections.Map[[]byte, []byte]
}

func NewVMStore(ctx context.Context, store collections.Map[[]byte, []byte]) VMStore {
	return VMStore{ctx, store}
}

func (s VMStore) Get(key []byte) []byte {
	bz, err := s.store.Get(s.ctx, key)
	if err != nil {
		panic(err)
	}

	return bz
}

func (s VMStore) Set(key, value []byte) {
	err := s.store.Set(s.ctx, key, value)
	if err != nil {
		panic(err)
	}
}

func (s VMStore) Delete(key []byte) {
	err := s.store.Remove(s.ctx, key)
	if err != nil {
		panic(err)
	}
}

func (s VMStore) Iterator(start, end []byte) dbm.Iterator {
	iterator, err := s.store.Iterate(s.ctx, new(collections.Range[[]byte]).StartInclusive(start).EndExclusive(end))
	if err != nil {
		panic(err)
	}

	return NewVMIterator(iterator, start, end)
}

func (s VMStore) ReverseIterator(start, end []byte) dbm.Iterator {
	iterator, err := s.store.Iterate(s.ctx, new(collections.Range[[]byte]).StartInclusive(start).EndExclusive(end).Descending())
	if err != nil {
		panic(err)
	}

	return NewVMIterator(iterator, start, end)
}

type VMIterator struct {
	collections.Iterator[[]byte, []byte]
	start, end []byte
}

func NewVMIterator(iter collections.Iterator[[]byte, []byte], start, end []byte) VMIterator {
	return VMIterator{Iterator: iter, start: start, end: end}
}

func (iter VMIterator) Domain() ([]byte, []byte) {
	return iter.start, iter.end
}

func (iter VMIterator) Error() error {
	return nil
}

func (iter VMIterator) Key() []byte {
	key, err := iter.Iterator.Key()
	if err != nil {
		panic(err)
	}

	return key
}

func (iter VMIterator) Value() []byte {
	value, err := iter.Iterator.Value()
	if err != nil {
		panic(err)
	}

	return value
}
