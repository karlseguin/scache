# scache
A simple concurrent friendly cache

```go
// a cache that'll hold 1000 users, users expire 1 minute after being inserted
userCache := scacne.New(1000, time.Minute)

// GET
u := userCache.Get("leto")
if u == nil {
  return nil
}
user := u.(*User)

// SET
userCache.Set("leto", user)
```
