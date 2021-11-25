// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package epochs

import (
	"context"
	"errors"

	"github.com/penguintop/penguin/pkg/feeds"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
)

var _ feeds.Lookup = (*finder)(nil)
var _ feeds.Lookup = (*asyncFinder)(nil)

// finder encapsulates a chunk store getter and a feed and provides
//  non-concurrent lookup methods
type finder struct {
	getter *feeds.Getter
}

// NewFinder constructs an AsyncFinder
func NewFinder(getter storage.Getter, feed *feeds.Feed) feeds.Lookup {
	return &finder{feeds.NewGetter(getter, feed)}
}

// At looks up the version valid at time `at`
// after is a unix time hint of the latest known update
func (f *finder) At(ctx context.Context, at, after int64) (penguin.Chunk, feeds.Index, feeds.Index, error) {
	e, ch, err := f.common(ctx, at, after)
	if err != nil {
		return nil, nil, nil, err
	}
	ch, err = f.at(ctx, uint64(at), e, ch)
	return ch, nil, nil, err
}

// common returns the lowest common ancestor for which a feed update chunk is found in the chunk store
func (f *finder) common(ctx context.Context, at, after int64) (*epoch, penguin.Chunk, error) {
	for e := lca(at, after); ; e = e.parent() {
		ch, err := f.getter.Get(ctx, e)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				if e.level == maxLevel {
					return e, nil, nil
				}
				continue
			}
			return e, nil, err
		}
		ts, err := feeds.UpdatedAt(ch)
		if err != nil {
			return e, nil, err
		}
		if ts <= uint64(at) {
			return e, ch, nil
		}
	}
}

// at is a non-concurrent recursive Finder function to find the version update chunk at time `at`
func (f *finder) at(ctx context.Context, at uint64, e *epoch, ch penguin.Chunk) (penguin.Chunk, error) {
	uch, err := f.getter.Get(ctx, e)
	if err != nil {
		// Error retrieving
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		// Epoch not found on branch
		if e.isLeft() { // no lower resolution
			return ch, nil
		}
		// Traverse earlier branch
		return f.at(ctx, e.start-1, e.left(), ch)
	}
	// Epoch found
	// Check if timestamp is later then target
	ts, err := feeds.UpdatedAt(uch)
	if err != nil {
		return nil, err
	}
	if ts > at {
		if e.isLeft() {
			return ch, nil
		}
		return f.at(ctx, e.start-1, e.left(), ch)
	}
	if e.level == 0 { // Matching update time or finest resolution
		return uch, nil
	}
	// Continue traversing based on at
	return f.at(ctx, at, e.childAt(at), uch)
}

type result struct {
	path  *path
	chunk penguin.Chunk
	*epoch
}

// asyncFinder encapsulates a chunk store getter and a feed and provides
//  non-concurrent lookup methods
type asyncFinder struct {
	getter *feeds.Getter
}

type path struct {
	at     int64
	top    *result
	bottom *result
	cancel chan struct{}
}

func newPath(at int64) *path {
	return &path{at, nil, nil, make(chan struct{})}
}

// NewAsyncFinder constructs an AsyncFinder
func NewAsyncFinder(getter storage.Getter, feed *feeds.Feed) feeds.Lookup {
	return &asyncFinder{feeds.NewGetter(getter, feed)}
}

func (f *asyncFinder) get(ctx context.Context, at int64, e *epoch) (penguin.Chunk, error) {
	u, err := f.getter.Get(ctx, e)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		return nil, nil
	}
	ts, err := feeds.UpdatedAt(u)
	if err != nil {
		return nil, err
	}
	diff := at - int64(ts)
	if diff < 0 {
		return nil, nil
	}
	return u, nil
}

// at attempts to retrieve all epoch chunks on the path for `at` concurrently
func (f *asyncFinder) at(ctx context.Context, at int64, p *path, e *epoch, c chan<- *result) {
	for ; ; e = e.childAt(uint64(at)) {
		select {
		case <-p.cancel:
			return
		default:
		}
		go func(e *epoch) {
			uch, err := f.get(ctx, at, e)
			if err != nil {
				return
			}
			select {
			case c <- &result{p, uch, e}:
			case <-p.cancel:
			}
		}(e)
		if e.level == 0 {
			return
		}
	}
}
func (f *asyncFinder) At(ctx context.Context, at, after int64) (penguin.Chunk, feeds.Index, feeds.Index, error) {
	// TODO: current and next index return values need to be implemented
	ch, err := f.asyncAt(ctx, at, after)
	return ch, nil, nil, err
}

// At looks up the version valid at time `at`
// after is a unix time hint of the latest known update
func (f *asyncFinder) asyncAt(ctx context.Context, at, after int64) (penguin.Chunk, error) {
	c := make(chan *result)
	go f.at(ctx, at, newPath(at), &epoch{0, maxLevel}, c)
LOOP:
	for r := range c {
		p := r.path
		// Ignore result from paths already  cancelled
		select {
		case <-p.cancel:
			continue LOOP
		default:
		}
		if r.chunk != nil { // Update chunk for epoch found
			if r.level == 0 { // Return if deepest level epoch
				return r.chunk, nil
			}
			// Ignore if higher level than the deepest epoch found
			if p.top != nil && p.top.level < r.level {
				continue LOOP
			}
			p.top = r
		} else { // Update chunk for epoch not found
			// If top level than return with no update found
			if r.level == 32 {
				close(p.cancel)
				return nil, nil
			}
			// If topmost epoch not found, then set bottom
			if p.bottom == nil || p.bottom.level < r.level {
				p.bottom = r
			}
		}

		// Found - not found for two consecutive epochs
		if p.top != nil && p.bottom != nil && p.top.level == p.bottom.level+1 {
			// Cancel path
			close(p.cancel)
			if p.bottom.isLeft() {
				return p.top.chunk, nil
			}
			// Recursively call on new path through left sister
			np := newPath(at)
			np.top = &result{np, p.top.chunk, p.top.epoch}
			go f.at(ctx, int64(p.bottom.start-1), np, p.bottom.left(), c)
		}
	}
	return nil, nil
}
