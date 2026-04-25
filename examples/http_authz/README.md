# HTTP authorization middleware

Demonstrates using gorege as net/http authorization middleware:

- `Engine` held in `atomic.Pointer` for lock-free reads.
- SIGHUP reloads rules without dropping in-flight requests.
- `Explain` surfaces the matched rule on `403` when `GOREGE_DEBUG` is set.
- No third-party HTTP framework dependency.

## Run

```sh
go run . rules.json
```

In another terminal:

```sh
# Public health endpoint (allowed for everyone via *,GET,health rule).
curl -i http://localhost:8080/health

# Anonymous read of posts: allowed (anon-read-posts).
curl -i http://localhost:8080/posts/42

# Anonymous write: denied (deny-anon-writes).
curl -i -X POST http://localhost:8080/posts

# Editor write: allowed (editor-posts-write).
curl -i -X POST -H "X-Role: editor" http://localhost:8080/posts

# Debug mode shows which rule fired.
GOREGE_DEBUG=1 go run . rules.json &
curl -i -X DELETE -H "X-Role: viewer" http://localhost:8080/posts
# -> 403 with: denied: matched=false allowed=false rule_index=-1 rule_name=""
```

## Hot reload

Edit `rules.json`, then:

```sh
kill -HUP $(pgrep -f 'go run . rules.json')
```

In-flight requests use the old engine; new requests use the new one.

## Production notes

- `extractRole` uses `X-Role` header for demo purposes only. Replace with
  verified JWT claims or session lookup.
- Failures from authorization checks (engine not loaded, arity drift) fail
  **closed** with 500, never silently allow.
- For high-throughput services, `Check` avoids allocating `Explanation`.
  This example only uses `Explain` when `GOREGE_DEBUG` is enabled.
