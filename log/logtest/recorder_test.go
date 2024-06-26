// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logtest

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
)

func TestRecorderLogger(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []Option

		loggerName    string
		loggerOptions []log.LoggerOption

		wantLogger log.Logger
	}{
		{
			name: "provides a default logger",

			wantLogger: &Recorder{
				currentScopeRecord: &ScopeRecords{},
			},
		},
		{
			name: "provides a logger with a configured scope",

			loggerName: "test",
			loggerOptions: []log.LoggerOption{
				log.WithInstrumentationVersion("logtest v42"),
				log.WithSchemaURL("https://example.com"),
			},

			wantLogger: &Recorder{
				currentScopeRecord: &ScopeRecords{
					Name:      "test",
					Version:   "logtest v42",
					SchemaURL: "https://example.com",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := NewRecorder(tt.options...).Logger(tt.loggerName, tt.loggerOptions...)
			// unset enabledFn to allow comparison
			l.(*Recorder).enabledFn = nil

			assert.Equal(t, tt.wantLogger, l)
		})
	}
}

func TestRecorderLoggerCreatesNewStruct(t *testing.T) {
	r := &Recorder{}
	assert.NotEqual(t, r, r.Logger("test"))
}

func TestRecorderEnabled(t *testing.T) {
	for _, tt := range []struct {
		name        string
		options     []Option
		ctx         context.Context
		buildRecord func() log.Record

		isEnabled bool
	}{
		{
			name: "the default option enables every log entry",
			ctx:  context.Background(),
			buildRecord: func() log.Record {
				return log.Record{}
			},

			isEnabled: true,
		},
		{
			name: "with everything disabled",
			options: []Option{
				WithEnabledFunc(func(context.Context, log.Record) bool {
					return false
				}),
			},
			ctx: context.Background(),
			buildRecord: func() log.Record {
				return log.Record{}
			},

			isEnabled: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := NewRecorder(tt.options...).Enabled(tt.ctx, tt.buildRecord())
			assert.Equal(t, tt.isEnabled, e)
		})
	}
}

func TestRecorderEnabledFnUnset(t *testing.T) {
	r := &Recorder{}
	assert.True(t, r.Enabled(context.Background(), log.Record{}))
}

func TestRecorderEmitAndReset(t *testing.T) {
	r := NewRecorder()
	assert.Len(t, r.Result()[0].Records, 0)

	r1 := log.Record{}
	r1.SetSeverity(log.SeverityInfo)
	r.Emit(context.Background(), r1)
	assert.Equal(t, r.Result()[0].Records, []log.Record{r1})

	l := r.Logger("test")
	assert.Empty(t, r.Result()[1].Records)

	r2 := log.Record{}
	r2.SetSeverity(log.SeverityError)
	l.Emit(context.Background(), r2)
	assert.Equal(t, r.Result()[0].Records, []log.Record{r1})
	assert.Equal(t, r.Result()[1].Records, []log.Record{r2})

	r.Reset()
	assert.Empty(t, r.Result()[0].Records)
	assert.Empty(t, r.Result()[1].Records)
}

func TestRecorderConcurrentSafe(t *testing.T) {
	const goRoutineN = 10

	var wg sync.WaitGroup
	wg.Add(goRoutineN)

	r := &Recorder{}

	for i := 0; i < goRoutineN; i++ {
		go func() {
			defer wg.Done()

			nr := r.Logger("test")
			nr.Enabled(context.Background(), log.Record{})
			nr.Emit(context.Background(), log.Record{})

			r.Result()
			r.Reset()
		}()
	}

	wg.Wait()
}
