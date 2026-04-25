// Package main demonstrates using gorege as an HTTP authorization middleware.
//
// The engine is held in an atomic.Pointer so SIGHUP reloads do not block
// in-flight requests.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/yplog/gorege"
)

type authzEngine struct {
	ptr atomic.Pointer[gorege.Engine]
}

func (a *authzEngine) load(path string) error {
	e, warnings, err := gorege.LoadFileWithOptions(path)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		log.Printf("authz warning [%s]: %s", w.Kind, w.Message)
	}
	a.ptr.Store(e)
	return nil
}

func (a *authzEngine) check(role, method, resource string) (bool, *gorege.Explanation, error) {
	e := a.ptr.Load()
	if e == nil {
		return false, nil, errors.New("authz engine not loaded")
	}
	ok, err := e.Check(role, method, resource)
	if err != nil {
		return false, nil, err
	}
	return ok, nil, nil
}

func (a *authzEngine) explain(role, method, resource string) (bool, *gorege.Explanation, error) {
	e := a.ptr.Load()
	if e == nil {
		return false, nil, errors.New("authz engine not loaded")
	}
	x, err := e.Explain(role, method, resource)
	if err != nil {
		return false, nil, err
	}
	return x.Matched && x.Allowed, &x, nil
}

// extractRole pulls role from a header; in real systems this would be
// derived from a verified JWT or session.
func extractRole(r *http.Request) string {
	role := strings.TrimSpace(r.Header.Get("X-Role"))
	if role == "" {
		return "anonymous"
	}
	return role
}

// resourceFromPath extracts the first path segment as the resource name.
// /posts/123 -> "posts"; /health -> "health".
func resourceFromPath(p string) string {
	p = strings.Trim(p, "/")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		p = p[:i]
	}
	return p
}

// Middleware returns an http.Handler that enforces gorege rules. On deny it
// writes 403 and, when debug is true, includes explanation fields in the body.
func Middleware(a *authzEngine, debug bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := extractRole(r)
		resource := resourceFromPath(r.URL.Path)

		var (
			ok  bool
			x   *gorege.Explanation
			err error
		)
		if debug {
			ok, x, err = a.explain(role, r.Method, resource)
		} else {
			ok, x, err = a.check(role, r.Method, resource)
		}
		if err != nil {
			// Engine not ready or arity mismatch (config drift). Fail closed.
			log.Printf("authz error: %v", err)
			http.Error(w, "internal authorization error", http.StatusInternalServerError)
			return
		}
		if !ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			if debug && x != nil {
				fmt.Fprintf(w, "denied: matched=%v allowed=%v rule_index=%d rule_name=%q\n",
					x.Matched, x.Allowed, x.RuleIndex, x.RuleName)
				return
			}
			fmt.Fprintln(w, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	rulesPath := "rules.json"
	if len(os.Args) > 1 {
		rulesPath = os.Args[1]
	}
	debug := os.Getenv("GOREGE_DEBUG") != ""

	a := &authzEngine{}
	if err := a.load(rulesPath); err != nil {
		log.Fatalf("initial load: %v", err)
	}

	// SIGHUP -> reload rules (preserves in-flight requests).
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func() {
		for range sigs {
			if err := a.load(rulesPath); err != nil {
				log.Printf("reload failed (keeping previous engine): %v", err)
				continue
			}
			log.Printf("reload ok: %s", rulesPath)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s %s\n", r.Method, r.URL.Path)
	})
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s %s\n", r.Method, r.URL.Path)
	})
	mux.HandleFunc("/settings/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s %s\n", r.Method, r.URL.Path)
	})

	addr := ":8080"
	srv := &http.Server{
		Addr:              addr,
		Handler:           Middleware(a, debug, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s (debug=%v)", addr, debug)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
