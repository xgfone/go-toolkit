// Copyright 2026 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package module

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

type pipelineModule struct {
	name               string
	dependencies       []string
	start, stop, close func(context.Context, *app.App) error
}

func (m pipelineModule) Name() string                            { return m.name }
func (m pipelineModule) Dependencies() []string                  { return m.dependencies }
func (m pipelineModule) Init(context.Context, *app.App) error    { return nil }
func (m pipelineModule) Prepare(context.Context, *app.App) error { return nil }
func (m pipelineModule) Start(ctx context.Context, a *app.App) error {
	if m.start != nil {
		return m.start(ctx, a)
	}
	return nil
}
func (m pipelineModule) Stop(ctx context.Context, a *app.App) error {
	if m.stop != nil {
		return m.stop(ctx, a)
	}
	return nil
}
func (m pipelineModule) Wait(context.Context, *app.App) error { return nil }
func (m pipelineModule) Close(ctx context.Context, a *app.App) error {
	if m.close != nil {
		return m.close(ctx, a)
	}
	return nil
}

func TestHTTPWorkerCacheDatabaseAndFileLoggerPipeline(t *testing.T) {
	a := app.New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context) error { return nil })

	parent, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started := make(chan struct{})
	release := make(chan struct{})
	committed := make(chan struct{})
	finishRequest := sync.OnceFunc(func() { close(release) })
	t.Cleanup(finishRequest)
	dirty := make(chan struct{}, 1)
	var databaseClosed atomic.Bool
	path := filepath.Join(t.TempDir(), "app.log")
	var logMu sync.Mutex
	var output *bufio.Writer
	log := func(message string) {
		logMu.Lock()
		defer logMu.Unlock()
		_, _ = fmt.Fprintln(output, message)
	}

	a.On(app.StageInit, "logger", func(context.Context) error {
		file, err := os.Create(path)
		if err != nil {
			return err
		}

		output = bufio.NewWriter(file)
		a.Finalize("logger", func(context.Context) error {
			log("logger.close")
			if err := output.Flush(); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Sync(); err != nil {
				_ = file.Close()
				return err
			}
			return file.Close()
		})
		return nil
	})

	a.Use(pipelineModule{
		name: "database",
		close: func(context.Context, *app.App) error {
			select {
			case <-committed:
			default:
				t.Error("database closed before cached writes committed")
			}
			databaseClosed.Store(true)
			log("database.close")
			return nil
		},
	})
	a.Use(pipelineModule{
		name:         "cache",
		dependencies: []string{"database"},
		start: func(context.Context, *app.App) error {
			a.TaskGroup("cache").Go("writeback", func(ctx context.Context) error {
				<-ctx.Done()
				select {
				case <-dirty:
				default:
					t.Error("final worker write was lost")
				}

				if databaseClosed.Load() {
					t.Error("cache flushed after database closed")
				}

				log("cache.commit")
				close(committed)
				return nil
			})
			return nil
		},
	})
	a.Use(pipelineModule{
		name:         "worker",
		dependencies: []string{"cache"},
		stop: func(context.Context, *app.App) error {
			if a.TaskGroup("cache").Context().Err() != nil {
				t.Error("cache stopped before worker")
			}
			return nil
		},
	})
	s := NewHttpServer(
		"http",
		func() string { return "127.0.0.1:0" },
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
			if r.Context().Err() != nil {
				t.Error("App.Stop canceled a draining request")
			}

			workers := a.TaskGroup("worker")
			if workers.Context().Err() != nil {
				t.Error("worker stopped before HTTP request")
			}

			workers.Go("final-job", func(context.Context) error {
				dirty <- struct{}{}
				log("worker.finished")
				return nil
			})

			log("request.finished")
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	a.Use(app.WithDependencies(s, "worker"))
	a.On(app.StageStopping, "finish-request", func(context.Context) error {
		finishRequest()
		return nil
	})
	a.On(app.StageExited, "notification", func(context.Context) error {
		log("app.exited")
		return nil
	})

	clientErr := make(chan error, 1)
	a.On(app.StageReady, "request", func(context.Context) error {
		go func() {
			client := &http.Client{Timeout: 3 * time.Second}
			url := "http://" + s.listen.Addr().String()
			response, err := client.Get(url)
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode != http.StatusNoContent {
					err = fmt.Errorf("status %d", response.StatusCode)
				}
			}
			clientErr <- err
		}()

		select {
		case <-started:
			a.Stop()
			return nil
		case <-parent.Done():
			return parent.Err()
		}
	})
	if err := a.Run(parent); err != nil {
		t.Fatal(err)
	}
	if err := <-clientErr; err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Fields(string(contents))
	before := func(x, y string) {
		t.Helper()
		ix, iy := -1, -1
		for i, line := range lines {
			if line == x {
				ix = i
			}
			if line == y {
				iy = i
			}
		}
		if ix < 0 || iy < 0 || ix >= iy {
			t.Errorf("%s must precede %s: %v", x, y, lines)
		}
	}

	before("worker.finished", "cache.commit")
	before("request.finished", "cache.commit")
	before("cache.commit", "database.close")
	before("database.close", "app.exited")
	before("app.exited", "logger.close")
}
