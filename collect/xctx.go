// Copyright (C) 2016 Space Monkey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// WARNING: THE NON-M4 VERSIONS OF THIS FILE ARE GENERATED BY GO GENERATE!
//          ONLY MAKE CHANGES TO THE M4 FILE
//

// +build !go1.7

package collect

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"
	"gopkg.in/spacemonkeygo/monkit.v2"
)

// WatchForSpans will watch for spans that 'matcher' returns true for. As soon
// as a trace generates a matched span, all spans from that trace that finish
// from that point on are collected until the matching span completes. All
// collected spans are returned.
// To cancel this operation, simply cancel the ctx argument.
// There is a small but permanent amount of overhead added by this function to
// every trace that is started while this function is running. This only really
// affects long-running traces.
func WatchForSpans(ctx context.Context, r *monkit.Registry,
	matcher func(s *monkit.Span) bool) (spans []*FinishedSpan, err error) {
	collector := NewSpanCollector(matcher)
	defer collector.Stop()

	var mtx sync.Mutex
	var cancelers []func()
	existingTraces := map[*monkit.Trace]bool{}

	canceler := r.ObserveTraces(func(t *monkit.Trace) {
		mtx.Lock()
		defer mtx.Unlock()
		if existingTraces[t] {
			return
		}
		existingTraces[t] = true
		cancelers = append(cancelers, t.ObserveSpans(collector))
	})
	cancelers = append(cancelers, canceler)

	// pick up live traces we can find
	r.RootSpans(func(s *monkit.Span) {
		mtx.Lock()
		defer mtx.Unlock()
		t := s.Trace()
		if existingTraces[t] {
			return
		}
		existingTraces[t] = true
		cancelers = append(cancelers, t.ObserveSpans(collector))
	})

	defer func() {
		mtx.Lock()
		defer mtx.Unlock()
		for _, canceler := range cancelers {
			canceler()
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-collector.Done():
		return collector.Spans(), nil
	}
}

// CollectSpans is kind of like WatchForSpans, except that it uses the current
// span to figure out which trace to collect. It calls work(), then collects
// from the current trace until work() returns. CollectSpans won't work unless
// some ancestor function is also monitored and has modified the ctx.
func CollectSpans(ctx context.Context, work func(ctx context.Context)) (
	spans []*FinishedSpan) {
	s := monkit.SpanFromCtx(ctx)
	if s == nil {
		work(ctx)
		return nil
	}
	collector := NewSpanCollector(nil)
	defer collector.Stop()
	s.Trace().ObserveSpans(collector)
	f := s.Func()
	newF := f.Scope().FuncNamed(fmt.Sprintf("%s-TRACED", f.ShortName()))
	func() {
		defer newF.Task(&ctx)(nil)
		collector.ForceStart(monkit.SpanFromCtx(ctx))
		work(ctx)
	}()
	return collector.Spans()
}
