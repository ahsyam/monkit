// Copyright (C) 2015 Space Monkey, Inc.
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

package present

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/spacemonkeygo/monitor.v2"
)

func outputDotSpan(w io.Writer, s *monitor.Span) error {
	orphaned := ""
	if s.Orphaned() {
		orphaned = "orphaned\n"
	}
	_, err := fmt.Fprintf(w,
		" f%d [label=\"%s",
		s.Id(), escapeDotLabel("%s(%s)\nelapsed: %s\n%s",
			s.Func().FullName(), strings.Join(s.Args(), ", "), s.Duration(),
			orphaned))
	if err != nil {
		return err
	}
	for _, annotation := range s.Annotations() {
		_, err = fmt.Fprint(w, escapeDotLabel("%s: %s\n",
			annotation.Name, annotation.Value))
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\"];\n")
	if err != nil {
		return err
	}
	s.Children(func(child *monitor.Span) {
		if err != nil {
			return
		}
		err = outputDotSpan(w, child)
		if err != nil {
			return
		}
		_, err = fmt.Fprintf(w, " f%d -> f%d;\n", s.Id(), child.Id())
		if err != nil {
			return
		}
	})
	return err
}

func SpansDot(r *monitor.Registry, w io.Writer) error {
	_, err := fmt.Fprintf(w, "digraph G {\n node [shape=box];\n")
	if err != nil {
		return err
	}
	r.RootSpans(func(s *monitor.Span) {
		if err != nil {
			return
		}
		err = outputDotSpan(w, s)
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "}\n")
	return err
}

func outputTextSpan(w io.Writer, s *monitor.Span, indent string) (err error) {
	orphaned := ""
	if s.Orphaned() {
		orphaned = ", orphaned"
	}
	_, err = fmt.Fprintf(w, "%s[%d] %s(%s) (elapsed: %s%s)\n",
		indent, s.Id(), s.Func().FullName(), strings.Join(s.Args(), ", "),
		s.Duration(), orphaned)
	if err != nil {
		return err
	}
	for _, annotation := range s.Annotations() {
		_, err = fmt.Fprintf(w, "%s  %s: %s\n", indent,
			annotation.Name, annotation.Value)
		if err != nil {
			return err
		}
	}
	s.Children(func(s *monitor.Span) {
		if err != nil {
			return
		}
		err = outputTextSpan(w, s, indent+" ")
	})
	return err
}

func SpansText(r *monitor.Registry, w io.Writer) (err error) {
	r.RootSpans(func(s *monitor.Span) {
		if err != nil {
			return
		}
		err = outputTextSpan(w, s, "")
		if err != nil {
			return
		}
		_, err = fmt.Fprintf(w, "\n")
	})
	return err
}

func SpansJSON(r *monitor.Registry, w io.Writer) (err error) {
	lw := newListWriter(w)
	r.AllSpans(func(s *monitor.Span) {
		lw.elem(formatSpan(s))
	})
	return lw.done()
}