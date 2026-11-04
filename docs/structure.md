# Repo structure

The decided layout for this repo. New code goes where this says it goes.

```
app/                    Go module (francodominguez.com/gmail-webhook/app)
  function.go           root stub — blank-imports each function package below
  functions/
    unsub/              one folder per Cloud Run function, package named after it
      function.go       handler + functions.HTTP registration
      unsub.go          the function's core logic, named after the function
      envelope.go       other helpers: separate files, same package, unexported
  utils/                package utils — code shared by more than one function
  cmd/                  regular binaries (package main): local server, oauth quickstart,
                        and any future workload that is NOT a function (e.g. VM jobs)
terrafrom/              all infrastructure
docs/                   decisions and notes like this one
```

## Rules

- **One folder per function** under `functions/`. The folder is its own package.
  The handler + registration file is always `function.go` (unless that ever causes
  a problem); the core logic lives in a file named after the function (`unsub.go`);
  further helpers are sibling files. Everything is unexported — other functions
  cannot reach any of it, and that privacy is the point.
- **Adding a function** = new folder under `functions/` + one blank-import line in
  `app/function.go`. The root stub exists because the GCF buildpack only imports
  the module root (`pack build --path ./app` in CI); a function whose `init()`
  isn't reachable from the root never registers.
- **A helper graduates to `utils/`** only when a second function actually needs
  it — not when it merely looks reusable.
- **Anything with a `main()`** goes under `cmd/<name>/`. These are plain binaries:
  they may import `utils/`, but the buildpack ignores them and they don't register
  functions.
- Growing deeper is the same pattern again: a function folder that gets big can
  split further (e.g. `functions/unsub/internal/…`), since `internal/` under the
  function's folder is only importable within it.
